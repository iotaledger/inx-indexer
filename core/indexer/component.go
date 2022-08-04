package indexer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/core/shutdown"
	"github.com/iotaledger/inx-app/httpserver"
	"github.com/iotaledger/inx-app/nodebridge"
	"github.com/iotaledger/inx-indexer/pkg/daemon"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	"github.com/iotaledger/inx-indexer/pkg/server"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	APIRoute = "indexer/v1"
)

// supportedProtocolVersion is the supported protocol version
// the application will exit if the node protocol version is not matched
const supportedProtocolVersion = 2

func init() {
	CoreComponent = &app.CoreComponent{
		Component: &app.Component{
			Name:     "Indexer",
			DepsFunc: func(cDeps dependencies) { deps = cDeps },
			Params:   params,
			Provide:  provide,
			Run:      run,
		},
	}
}

type dependencies struct {
	dig.In
	NodeBridge      *nodebridge.NodeBridge
	Indexer         *indexer.Indexer
	ShutdownHandler *shutdown.ShutdownHandler
	Echo            *echo.Echo
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

func provide(c *dig.Container) error {

	if err := c.Provide(func() (*indexer.Indexer, error) {
		CoreComponent.LogInfo("Setting up database...")
		return indexer.NewIndexer(ParamsIndexer.Database.Path, CoreComponent.Logger())
	}); err != nil {
		return err
	}

	if err := c.Provide(func() *echo.Echo {
		return httpserver.NewEcho(
			CoreComponent.Logger(),
			nil,
			ParamsIndexer.DebugRequestLoggerEnabled,
		)
	}); err != nil {
		return err
	}

	return nil
}

func run() error {

	indexerInitWaitGroup := &sync.WaitGroup{}
	indexerInitWaitGroup.Add(1)

	// create a background worker that handles the indexer events
	CoreComponent.Daemon().BackgroundWorker("Indexer", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting Indexer")
		defer deps.Indexer.CloseDatabase()

		indexerStatus, err := checkIndexerStatus(ctx)
		if err != nil {
			CoreComponent.LogErrorfAndExit("Checking initial Indexer state failed: %s", err.Error())
			return
		}
		indexerInitWaitGroup.Done()

		CoreComponent.LogInfo("Starting LedgerUpdates ... done")

		if err := deps.NodeBridge.ListenToLedgerUpdates(ctx, indexerStatus.LedgerIndex+1, 0, func(update *nodebridge.LedgerUpdate) error {
			ts := time.Now()
			if err := deps.Indexer.UpdatedLedger(update); err != nil {
				return err
			}

			CoreComponent.LogInfof("Applying milestone %d with %d new and %d consumed outputs took %s", update.MilestoneIndex, len(update.Created), len(update.Consumed), time.Since(ts).Truncate(time.Millisecond))
			return nil
		}); err != nil {
			deps.ShutdownHandler.SelfShutdown(fmt.Sprintf("Listening to LedgerUpdates failed, error: %s", err), false)
		}

		CoreComponent.LogInfo("Stopping LedgerUpdates ... done")
		CoreComponent.LogInfo("Stopped Indexer")
	}, daemon.PriorityStopIndexer)

	// create a background worker that handles the API
	if err := CoreComponent.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting API")

		// we need to wait until the indexer is initialized before starting the API
		indexerInitWaitGroup.Wait()
		CoreComponent.LogInfo("Starting API ... done")

		CoreComponent.LogInfo("Starting API server...")

		_ = server.NewIndexerServer(deps.Indexer, deps.Echo.Group(""), deps.NodeBridge.ProtocolParameters().Bech32HRP, ParamsIndexer.MaxPageSize)

		go func() {
			CoreComponent.LogInfof("You can now access the API using: http://%s", ParamsIndexer.BindAddress)
			if err := deps.Echo.Start(ParamsIndexer.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				CoreComponent.LogPanicf("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		if err := deps.NodeBridge.RegisterAPIRoute(APIRoute, ParamsIndexer.BindAddress); err != nil {
			CoreComponent.LogPanicf("Registering INX api route failed, error: %s", err)
		}

		<-ctx.Done()
		CoreComponent.LogInfo("Stopping API ...")

		if err := deps.NodeBridge.UnregisterAPIRoute(APIRoute); err != nil {
			CoreComponent.LogWarnf("Unregistering INX api route failed, error: %s", err)
		}

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := deps.Echo.Shutdown(shutdownCtx); err != nil {
			CoreComponent.LogWarn(err)
		}
		shutdownCtxCancel()
		CoreComponent.LogInfo("Stopping API ... done")
	}, daemon.PriorityStopIndexerAPI); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}

func checkIndexerStatus(ctx context.Context) (*indexer.Status, error) {
	needsToFillIndexer := false
	needsToClearIndexer := false

	protocolParams := deps.NodeBridge.ProtocolParameters()
	// check protocol version
	if protocolParams.Version != supportedProtocolVersion {
		return nil, fmt.Errorf("the supported protocol version is %d but the node protocol is %d", supportedProtocolVersion, protocolParams.Version)
	}

	nodeStatus, err := deps.NodeBridge.NodeStatus()
	if err != nil {
		return nil, fmt.Errorf("error loading node status: %s", err)
	}

	// Checking initial indexer state
	indexerStatus, err := deps.Indexer.Status()
	if err != nil {
		if !errors.Is(err, indexer.ErrNotFound) {
			return nil, fmt.Errorf("reading ledger index from Indexer failed! Error: %s", err)
		}
		CoreComponent.LogInfo("Indexer is empty, so import initial ledger...")
		needsToFillIndexer = true
	} else {
		if indexerStatus.ProtocolVersion != protocolParams.Version {
			CoreComponent.LogInfof("> Network protocol version changed: %d vs %d", indexerStatus.ProtocolVersion, protocolParams.Version)
			needsToClearIndexer = true
		} else if indexerStatus.NetworkName != protocolParams.NetworkName {
			CoreComponent.LogInfof("> Network name changed: %s vs %s", indexerStatus.NetworkName, protocolParams.NetworkName)
			needsToClearIndexer = true
		} else if nodeStatus.LedgerIndex < indexerStatus.LedgerIndex {
			CoreComponent.LogInfo("> Network has been reset: indexer index > ledger index")
			needsToClearIndexer = true
		} else if nodeStatus.GetLedgerPruningIndex() > indexerStatus.LedgerIndex {
			CoreComponent.LogInfo("> Node has an newer pruning index than our current ledgerIndex")
			needsToClearIndexer = true
		}
	}

	if needsToClearIndexer {
		CoreComponent.LogInfo("Re-import initial ledger...")
		if err := deps.Indexer.Clear(); err != nil {
			return nil, fmt.Errorf("clearing Indexer failed! Error: %w", err)
		}
		needsToFillIndexer = true
	}

	if needsToFillIndexer {
		// Indexer is empty, so import initial ledger state from the node
		timeStart := time.Now()
		var count int
		if count, err = fillIndexer(ctx, deps.Indexer, protocolParams); err != nil {
			return nil, fmt.Errorf("filling Indexer failed! Error: %w", err)
		}
		duration := time.Since(timeStart)
		// Read new ledgerIndex after filling up the indexer
		indexerStatus, err = deps.Indexer.Status()
		if err != nil {
			return nil, fmt.Errorf("reading ledger index from Indexer failed! Error: %s", err)
		}
		CoreComponent.LogInfof("Importing initial ledger with %d outputs at index %d took %s", count, indexerStatus.LedgerIndex, duration.Truncate(time.Millisecond))
	} else {
		CoreComponent.LogInfof("> Indexer started at ledgerIndex %d", indexerStatus.LedgerIndex)
	}

	return indexerStatus, nil
}

func fillIndexer(ctx context.Context, indexer *indexer.Indexer, protoParams *iotago.ProtocolParameters) (int, error) {
	importer := indexer.ImportTransaction()

	stream, err := deps.NodeBridge.Client().ReadUnspentOutputs(ctx, &inx.NoParams{})
	if err != nil {
		return 0, err
	}

	var count int
	var ledgerIndex uint32
	for {
		unspentOutput, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if err := importer.AddOutput(unspentOutput.GetOutput()); err != nil {
			return 0, err
		}
		count++
		outputLedgerIndex := unspentOutput.GetLedgerIndex()
		if ledgerIndex < outputLedgerIndex {
			ledgerIndex = outputLedgerIndex
		}
	}

	return count, importer.Finalize(ledgerIndex, protoParams)
}
