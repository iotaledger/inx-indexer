package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gohornet/inx-indexer/indexer"
	"github.com/gohornet/inx-indexer/server"
	"github.com/iotaledger/hive.go/configuration"
	inx "github.com/iotaledger/inx/go"
)

var (
	// Version of the app.
	Version = "0.1.2"
)

const (
	APIRoute = "indexer/v1"

	// CfgINXAddress the INX address to which to connect to.
	CfgINXAddress = "inx.address"
	// CfgIndexerBindAddress bind address on which the Indexer HTTP server listens.
	CfgIndexerBindAddress = "indexer.bindAddress"
	// CfgIndexerMaxPageSize the maximum number of results that may be returned for each page.
	CfgIndexerMaxPageSize = "indexer.maxPageSize"

	// CfgPrometheusEnabled enable prometheus metrics.
	CfgPrometheusEnabled = "prometheus.enabled"
	// CfgPrometheusBindAddress bind address on which the Prometheus HTTP server listens.
	CfgPrometheusBindAddress = "prometheus.bindAddress"
)

func fillIndexer(client inx.INXClient, indexer *indexer.Indexer) error {
	importer := indexer.ImportTransaction()
	stream, err := client.ReadUnspentOutputs(context.Background(), &inx.NoParams{})
	if err != nil {
		panic(err)
	}
	var ledgerIndex uint32
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := importer.AddOutput(message.GetOutput()); err != nil {
			return err
		}
		outputLedgerIndex := message.GetLedgerIndex()
		if ledgerIndex < outputLedgerIndex {
			ledgerIndex = outputLedgerIndex
		}
	}
	if err := importer.Finalize(ledgerIndex); err != nil {
		return err
	}
	fmt.Printf("Imported initial ledger at index %d\n", ledgerIndex)
	return nil
}

func listenToLedgerUpdates(ctx context.Context, client inx.INXClient, indexer *indexer.Indexer) error {
	ledgerIndex, err := indexer.LedgerIndex()
	if err != nil {
		return err
	}
	req := &inx.LedgerRequest{
		StartMilestoneIndex: uint32(ledgerIndex + 1),
	}
	stream, err := client.ListenToLedgerUpdates(ctx, req)
	if err != nil {
		panic(err)
	}
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := indexer.UpdatedLedger(message); err != nil {
			return err
		}
		fmt.Printf("> Updated ledgerIndex to %d with %d created and %d consumed outputs\n", message.GetMilestoneIndex(), len(message.GetCreated()), len(message.GetConsumed()))
	}
	return nil
}

func newEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	return e
}

func setupPrometheus(bindAddress string) {
	e := newEcho()
	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	go func() {
		if err := e.Start(bindAddress); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()
}

func retryBackoff(_ uint) time.Duration {
	return 2 * time.Second
}

func main() {
	fmt.Printf(">>>>> Starting Indexer %s <<<<<\n", Version)

	config, err := loadConfigFile("config.json")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(config.String(CfgINXAddress),
		grpc.WithChainUnaryInterceptor(grpc_retry.UnaryClientInterceptor(), grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := inx.NewINXClient(conn)

	fmt.Println("Connecting to node and reading protocol parameters...")
	protocolParams, err := client.ReadProtocolParameters(context.Background(), &inx.NoParams{}, grpc_retry.WithMax(10), grpc_retry.WithBackoff(retryBackoff))
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	fmt.Println("Setting up database...")
	i, err := indexer.NewIndexer(".")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	defer i.CloseDatabase()

	ledgerIndex, err := i.LedgerIndex()
	if err != nil {
		if errors.Is(err, indexer.ErrNotFound) {
			// Indexer is empty, so import initial ledger state from the node
			fmt.Println("Indexer is empty, so import initial ledger...")
			if err := fillIndexer(client, i); err != nil {
				fmt.Printf("Error: %s\n", err)
				return
			}
		} else {
			fmt.Printf("Error: %s\n", err)
			return
		}
	} else {
		fmt.Printf("> Indexer started at ledgerIndex %d\n", ledgerIndex)
	}

	fmt.Println("Reading node status...")
	resp, err := client.ReadNodeStatus(context.Background(), &inx.NoParams{})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	if resp.GetPruningIndex() > ledgerIndex {
		fmt.Println("> Node has an newer pruning index than our current ledgerIndex\nRe-import initial ledger...")
		if err := i.Clear(); err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		if err := fillIndexer(client, i); err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
	}

	e := newEcho()
	apiErrorHandler := server.ErrorHandler()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		fmt.Printf("Error: %s", err)
		apiErrorHandler(err, c)
	}

	fmt.Println("Starting API server...")
	server.NewIndexerServer(i, e.Group(""), protocolParams.NetworkPrefix(), config.Int(CfgIndexerMaxPageSize))
	go func() {
		if err := e.Start(config.String(CfgIndexerBindAddress)); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	bindAddressParts := strings.Split(config.String(CfgIndexerBindAddress), ":")
	if len(bindAddressParts) != 2 {
		panic(fmt.Sprintf("Invalid %s", CfgIndexerBindAddress))
	}
	port, err := strconv.ParseInt(bindAddressParts[1], 10, 32)
	if err != nil {
		panic(err)
	}

	apiReq := &inx.APIRouteRequest{
		Route: APIRoute,
		Host:  bindAddressParts[0],
		Port:  uint32(port),
	}
	if config.Bool(CfgPrometheusEnabled) {
		prometheusBindAddressParts := strings.Split(config.String(CfgPrometheusBindAddress), ":")
		if len(prometheusBindAddressParts) != 2 {
			panic(fmt.Sprintf("Invalid %s", CfgPrometheusBindAddress))
		}
		prometheusPort, err := strconv.ParseInt(prometheusBindAddressParts[1], 10, 32)
		if err != nil {
			panic(err)
		}
		setupPrometheus(config.String(CfgPrometheusBindAddress))
		apiReq.MetricsPort = uint32(prometheusPort)
	}

	fmt.Printf("Registering API route to http://%s:%d\n", apiReq.GetHost(), apiReq.GetPort())
	if _, err := client.RegisterAPIRoute(context.Background(), apiReq); err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	fmt.Println("Listening to ledger changes:")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := listenToLedgerUpdates(ctx, client, i); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		cancel()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		<-signalChan
		done <- true
	}()
	go func() {
		select {
		case <-signalChan:
			done <- true
		case <-ctx.Done():
			done <- true
		}
	}()
	<-done
	cancel()
	e.Close()
	fmt.Println("Removing API route...")
	if _, err := client.UnregisterAPIRoute(context.Background(), apiReq); err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	fmt.Println("exiting")
}

func flagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(CfgINXAddress, "localhost:9029", "the INX address to which to connect to")
	fs.String(CfgIndexerBindAddress, "localhost:9091", "bind address on which the Indexer HTTP server listens")
	fs.Int(CfgIndexerMaxPageSize, 1000, "the maximum number of results that may be returned for each page")
	fs.Bool(CfgPrometheusEnabled, false, "enable prometheus metrics")
	fs.String(CfgPrometheusBindAddress, "localhost:9312", "bind address on which the Prometheus HTTP server listens.")
	return fs
}

func loadConfigFile(filePath string) (*configuration.Configuration, error) {
	config := configuration.New()
	if err := config.LoadFile(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading config file failed: %w", err)
	}

	fs := flagSet()
	flag.CommandLine.AddFlagSet(fs)
	flag.Parse()

	if err := config.LoadFlagSet(fs); err != nil {
		return nil, err
	}
	return config, nil
}
