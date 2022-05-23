package nodebridge

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gohornet/inx-indexer/pkg/indexer"
	"github.com/iotaledger/hive.go/logger"
	inx "github.com/iotaledger/inx/go"
)

type NodeBridge struct {
	// the logger used to log events.
	*logger.WrappedLogger

	client             inx.INXClient
	NodeConfig         *inx.NodeConfiguration
	ledgerPruningIndex uint32
}

func NewNodeBridge(ctx context.Context, client inx.INXClient, log *logger.Logger) (*NodeBridge, error) {
	log.Info("Connecting to node and reading protocol parameters...")

	retryBackoff := func(_ uint) time.Duration {
		return 1 * time.Second
	}

	nodeConfig, err := client.ReadNodeConfiguration(ctx, &inx.NoParams{}, grpc_retry.WithMax(5), grpc_retry.WithBackoff(retryBackoff))
	if err != nil {
		return nil, err
	}

	nodeStatus, err := client.ReadNodeStatus(ctx, &inx.NoParams{})
	if err != nil {
		return nil, err
	}

	return &NodeBridge{
		WrappedLogger:      logger.NewWrappedLogger(log),
		client:             client,
		NodeConfig:         nodeConfig,
		ledgerPruningIndex: nodeStatus.GetLedgerPruningIndex(),
	}, nil
}

func (n *NodeBridge) Run(ctx context.Context) {
	<-ctx.Done()
}

func (n *NodeBridge) LedgerPruningIndex() uint32 {
	return n.ledgerPruningIndex
}

func (n *NodeBridge) FillIndexer(ctx context.Context, indexer *indexer.Indexer) error {
	importer := indexer.ImportTransaction()

	stream, err := n.client.ReadUnspentOutputs(ctx, &inx.NoParams{})
	if err != nil {
		return err
	}

	var ledgerIndex uint32
	for {
		unspentOutput, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := importer.AddOutput(unspentOutput.GetOutput()); err != nil {
			return err
		}
		outputLedgerIndex := unspentOutput.GetLedgerIndex()
		if ledgerIndex < outputLedgerIndex {
			ledgerIndex = outputLedgerIndex
		}
	}

	if err := importer.Finalize(ledgerIndex); err != nil {
		return err
	}

	return nil
}

func (n *NodeBridge) ListenToLedgerUpdates(ctx context.Context, startIndex uint32, consume func(update *inx.LedgerUpdate) error) error {

	req := &inx.MilestoneRangeRequest{
		StartMilestoneIndex: uint32(startIndex),
	}

	stream, err := n.client.ListenToLedgerUpdates(ctx, req)
	if err != nil {
		return err
	}

	for {
		update, err := stream.Recv()
		if err == io.EOF || status.Code(err) == codes.Canceled {
			break
		}
		if ctx.Err() != nil {
			// context got cancelled, so stop the updates
			return nil
		}
		if err != nil {
			return err
		}
		if err := consume(update); err != nil {
			return err
		}
	}
	return nil
}

func (n *NodeBridge) RegisterAPIRoute(route string, bindAddress string) error {
	bindAddressParts := strings.Split(bindAddress, ":")
	if len(bindAddressParts) != 2 {
		return fmt.Errorf("Invalid address %s", bindAddress)
	}
	port, err := strconv.ParseInt(bindAddressParts[1], 10, 32)
	if err != nil {
		return err
	}

	apiReq := &inx.APIRouteRequest{
		Route: route,
		Host:  bindAddressParts[0],
		Port:  uint32(port),
	}

	if err != nil {
		return err
	}
	_, err = n.client.RegisterAPIRoute(context.Background(), apiReq)
	return err
}

func (n *NodeBridge) UnregisterAPIRoute(route string) error {
	apiReq := &inx.APIRouteRequest{
		Route: route,
	}
	_, err := n.client.UnregisterAPIRoute(context.Background(), apiReq)
	return err
}
