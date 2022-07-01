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
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/core/shutdown"
	"github.com/iotaledger/inx-app/nodebridge"
	"github.com/iotaledger/inx-indexer/pkg/daemon"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	"github.com/iotaledger/inx-indexer/pkg/server"
	inx "github.com/iotaledger/inx/go"
)

const (
	APIRoute = "indexer/v1"
)

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
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

func provide(c *dig.Container) error {

	if err := c.Provide(func() (*indexer.Indexer, error) {
		CoreComponent.LogInfo("Setting up database...")
		return indexer.NewIndexer(ParamsIndexer.Database.Path)
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

		ledgerIndex, err := checkIndexerState(ctx)
		if err != nil {
			CoreComponent.LogPanicf("Checking initial Indexer state failed: %s", err.Error())
			return
		}
		indexerInitWaitGroup.Done()

		CoreComponent.LogInfo("Starting LedgerUpdates ... done")

		if err := deps.NodeBridge.ListenToLedgerUpdates(ctx, ledgerIndex+1, 0, func(update *inx.LedgerUpdate) error {
			ts := time.Now()
			if err := deps.Indexer.UpdatedLedger(update); err != nil {
				return err
			}

			CoreComponent.LogInfof("Applying milestone %d with %d new and %d outputs took %s", update.GetMilestoneIndex(), len(update.Created), len(update.Consumed), time.Since(ts).Truncate(time.Millisecond))
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

		e := newEcho()

		apiErrorHandler := server.ErrorHandler()
		e.HTTPErrorHandler = func(err error, c echo.Context) {
			CoreComponent.LogDebugf("Error: %s", err)
			apiErrorHandler(err, c)
		}

		CoreComponent.LogInfo("Starting API server...")

		_ = server.NewIndexerServer(deps.Indexer, e.Group(""), deps.NodeBridge.NodeConfig.UnwrapProtocolParameters().Bech32HRP, ParamsIndexer.MaxPageSize)

		go func() {
			CoreComponent.LogInfof("You can now access the API using: http://%s", ParamsIndexer.BindAddress)
			if err := e.Start(ParamsIndexer.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
		if err := e.Shutdown(shutdownCtx); err != nil {
			CoreComponent.LogWarn(err)
		}
		shutdownCtxCancel()
		CoreComponent.LogInfo("Stopping API ... done")
	}, daemon.PriorityStopIndexerAPI); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}

func newEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	return e
}

func checkIndexerState(ctx context.Context) (uint32, error) {
	needsToFillIndexer := false

	// Checking initial indexer state
	ledgerIndex, err := deps.Indexer.LedgerIndex()
	if err != nil {
		if !errors.Is(err, indexer.ErrNotFound) {
			return 0, fmt.Errorf("reading ledger index from Indexer failed! Error: %s", err)
		}
		CoreComponent.LogInfo("Indexer is empty, so import initial ledger...")
		needsToFillIndexer = true
	}

	nodeStatus, err := deps.NodeBridge.NodeStatus()
	if err != nil {
		return 0, fmt.Errorf("error loading pruning index: %s", err)
	}

	if !needsToFillIndexer && nodeStatus.GetLedgerPruningIndex() > ledgerIndex {
		CoreComponent.LogInfo("> Node has an newer pruning index than our current ledgerIndex")
		CoreComponent.LogInfo("Re-import initial ledger...")

		if err := deps.Indexer.Clear(); err != nil {
			return 0, fmt.Errorf("clearing Indexer failed! Error: %w", err)
		}
		needsToFillIndexer = true
	}

	if needsToFillIndexer {
		// Indexer is empty, so import initial ledger state from the node
		if err := fillIndexer(ctx, deps.Indexer); err != nil {
			return 0, fmt.Errorf("filling Indexer failed! Error: %w", err)
		}
		// Read new ledgerIndex after filling up the indexer
		ledgerIndex, err = deps.Indexer.LedgerIndex()
		if err != nil {
			return 0, fmt.Errorf("reading ledger index from Indexer failed! Error: %s", err)
		}
		CoreComponent.LogInfof("Imported initial ledger at index %d", ledgerIndex)
	} else {
		CoreComponent.LogInfof("> Indexer started at ledgerIndex %d", ledgerIndex)
	}

	return ledgerIndex, nil
}

func fillIndexer(ctx context.Context, indexer *indexer.Indexer) error {
	importer := indexer.ImportTransaction()

	stream, err := deps.NodeBridge.Client().ReadUnspentOutputs(ctx, &inx.NoParams{})
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

	return importer.Finalize(ledgerIndex)
}