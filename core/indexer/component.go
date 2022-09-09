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

	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/hive.go/core/app/core/shutdown"
	"github.com/iotaledger/inx-app/httpserver"
	"github.com/iotaledger/inx-app/nodebridge"
	"github.com/iotaledger/inx-indexer/pkg/daemon"
	"github.com/iotaledger/inx-indexer/pkg/database"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	"github.com/iotaledger/inx-indexer/pkg/server"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	APIRoute = "indexer/v1"
)

// supportedProtocolVersion is the supported protocol version
// the application will exit if the node protocol version is not matched.
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
		CoreComponent.LogInfo("Setting up database ...")

		engine, err := database.EngineFromString(ParamsIndexer.Database.Engine)
		if err != nil {
			return nil, err
		}

		return indexer.NewIndexer(database.Params{
			Engine:   engine,
			Path:     ParamsIndexer.Database.Sqlite.Path,
			Host:     ParamsIndexer.Database.Postgres.Host,
			Port:     ParamsIndexer.Database.Postgres.Port,
			Database: ParamsIndexer.Database.Postgres.Database,
			Username: ParamsIndexer.Database.Postgres.Username,
			Password: ParamsIndexer.Database.Postgres.Password,
		}, CoreComponent.Logger())
	}); err != nil {
		return err
	}

	if err := c.Provide(func() *echo.Echo {
		return httpserver.NewEcho(
			CoreComponent.Logger(),
			nil,
			ParamsRestAPI.DebugRequestLoggerEnabled,
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
	if err := CoreComponent.Daemon().BackgroundWorker("Indexer", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting Indexer")
		defer func() {
			if err := deps.Indexer.CloseDatabase(); err != nil {
				CoreComponent.LogErrorf("Failed to close database: %s", err.Error())
			}
		}()

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
	}, daemon.PriorityStopIndexer); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	// create a background worker that handles the API
	if err := CoreComponent.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting API")

		// we need to wait until the indexer is initialized before starting the API
		indexerInitWaitGroup.Wait()
		CoreComponent.LogInfo("Starting API ... done")

		CoreComponent.LogInfo("Starting API server ...")

		_ = server.NewIndexerServer(deps.Indexer, deps.Echo.Group(""), deps.NodeBridge.ProtocolParameters().Bech32HRP, ParamsRestAPI.MaxPageSize)

		go func() {
			CoreComponent.LogInfof("You can now access the API using: http://%s", ParamsRestAPI.BindAddress)
			if err := deps.Echo.Start(ParamsRestAPI.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				CoreComponent.LogErrorfAndExit("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := ParamsRestAPI.BindAddress
		if ParamsRestAPI.AdvertiseAddress != "" {
			advertisedAddress = ParamsRestAPI.AdvertiseAddress
		}

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, APIRoute, advertisedAddress); err != nil {
			CoreComponent.LogErrorfAndExit("Registering INX api route failed: %s", err)
		}
		cancelRegister()

		CoreComponent.LogInfo("Starting API server ... done")
		<-ctx.Done()
		CoreComponent.LogInfo("Stopping API ...")

		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, APIRoute); err != nil {
			CoreComponent.LogWarnf("Unregistering INX api route failed: %s", err)
		}

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		if err := deps.Echo.Shutdown(shutdownCtx); err != nil {
			CoreComponent.LogWarn(err)
		}

		CoreComponent.LogInfo("Stopping API ... done")
	}, daemon.PriorityStopIndexerAPI); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}

func checkIndexerStatus(ctx context.Context) (*indexer.Status, error) {
	var status *indexer.Status
	var err error

	needsToFillIndexer := false
	needsToClearIndexer := false

	protocolParams := deps.NodeBridge.ProtocolParameters()
	// check protocol version
	if protocolParams.Version != supportedProtocolVersion {
		return nil, fmt.Errorf("the supported protocol version is %d but the node protocol is %d", supportedProtocolVersion, protocolParams.Version)
	}

	if !deps.Indexer.IsInitialized() {
		// Starting indexer without a database
		if err := deps.Indexer.CreateTables(); err != nil {
			return nil, err
		}
		needsToFillIndexer = true
	} else {
		// Checking current indexer state to see if it needs a reset or not
		nodeStatus := deps.NodeBridge.NodeStatus()
		status, err = deps.Indexer.Status()
		if err != nil {
			if !errors.Is(err, indexer.ErrNotFound) {
				return nil, fmt.Errorf("reading ledger index from Indexer failed! Error: %w", err)
			}
			CoreComponent.LogInfo("Indexer is empty, so import initial ledger...")
			needsToFillIndexer = true
		} else {
			switch {
			case status.ProtocolVersion != protocolParams.Version:
				CoreComponent.LogInfof("> Network protocol version changed: %d vs %d", status.ProtocolVersion, protocolParams.Version)
				needsToClearIndexer = true

			case status.NetworkName != protocolParams.NetworkName:
				CoreComponent.LogInfof("> Network name changed: %s vs %s", status.NetworkName, protocolParams.NetworkName)
				needsToClearIndexer = true

			case nodeStatus.LedgerIndex < status.LedgerIndex:
				CoreComponent.LogInfo("> Network has been reset: indexer index > ledger index")
				needsToClearIndexer = true

			case nodeStatus.GetLedgerPruningIndex() > status.LedgerIndex:
				CoreComponent.LogInfo("> Node has an newer pruning index than our current ledgerIndex")
				needsToClearIndexer = true
			}
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
		status, err = deps.Indexer.Status()
		if err != nil {
			return nil, fmt.Errorf("reading ledger index from Indexer failed! Error: %w", err)
		}
		CoreComponent.LogInfo("Re-creating indexes")
		if err := deps.Indexer.AutoMigrate(); err != nil {
			return nil, err
		}
		CoreComponent.LogInfof("Importing initial ledger with %d outputs at index %d took %s", count, status.LedgerIndex, duration.Truncate(time.Millisecond))
	} else {
		CoreComponent.LogInfo("Checking database schema")
		if err := deps.Indexer.AutoMigrate(); err != nil {
			return nil, err
		}
		CoreComponent.LogInfof("> Indexer started at ledgerIndex %d", status.LedgerIndex)
	}

	// Run auto migrate to make sure all required tables and indexes are there
	return status, nil
}

func fillIndexer(ctx context.Context, indexer *indexer.Indexer, protoParams *iotago.ProtocolParameters) (int, error) {
	// Drop the indexes while doing bulk inserts to speed-up insertion.
	indexer.DropIndexes()

	var innerErr error
	receiveCtx, receiveCancel := context.WithCancel(ctx)

	importer := indexer.ImportTransaction()

	stream, err := deps.NodeBridge.Client().ReadUnspentOutputs(receiveCtx, &inx.NoParams{})
	if err != nil {
		return 0, err
	}

	var ledgerIndex uint32
	var countReceive int
	go func() {
		tsReceive := time.Now()
		for {
			unspentOutput, err := stream.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					innerErr = err
				}
				receiveCancel()
				break
			}

			if err := importer.AddOutput(unspentOutput.GetOutput()); err != nil {
				innerErr = err
				receiveCancel()
				return
			}
			outputLedgerIndex := unspentOutput.GetLedgerIndex()
			if ledgerIndex < outputLedgerIndex {
				ledgerIndex = outputLedgerIndex
			}

			countReceive++
			if countReceive%100000 == 0 {
				CoreComponent.LogInfof("received %d ... in %s", countReceive, time.Since(tsReceive).Truncate(time.Millisecond))
				tsReceive = time.Now()
			}
		}
	}()

	<-receiveCtx.Done()

	if innerErr != nil {
		return 0, innerErr
	}

	if err := importer.Finalize(ledgerIndex, protoParams); err != nil {
		return 0, err
	}

	// Re-create the indexes.
	return countReceive, indexer.CreateIndexes()
}
