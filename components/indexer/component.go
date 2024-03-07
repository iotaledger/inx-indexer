package indexer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/dig"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/db"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/sql"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-indexer/pkg/daemon"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	"github.com/iotaledger/inx-indexer/pkg/server"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	DBVersion uint32 = 2
)

func init() {
	Component = &app.Component{
		Name:     "Indexer",
		DepsFunc: func(cDeps dependencies) { deps = cDeps },
		Params:   params,
		Provide:  provide,
		Run:      run,
	}
}

type dependencies struct {
	dig.In
	NodeBridge      nodebridge.NodeBridge
	Indexer         *indexer.Indexer
	ShutdownHandler *shutdown.ShutdownHandler
	Echo            *echo.Echo
}

var (
	Component *app.Component
	deps      dependencies
)

func provide(c *dig.Container) error {

	if err := c.Provide(func() (*indexer.Indexer, error) {
		Component.LogInfo("Setting up database ...")

		engine := db.EngineFromString(ParamsIndexer.Database.Engine)

		dbParams := sql.DatabaseParameters{
			Engine: engine,
		}

		//nolint:exhaustive // we already checked the values is one of the valid ones
		switch engine {
		case db.EngineSQLite:
			dbParams.Path = ParamsIndexer.Database.SQLite.Path
			dbParams.Filename = "indexer.db"

		case db.EnginePostgreSQL:
			dbParams.Host = ParamsIndexer.Database.PostgreSQL.Host
			dbParams.Port = ParamsIndexer.Database.PostgreSQL.Port
			dbParams.Database = ParamsIndexer.Database.PostgreSQL.Database
			dbParams.Username = ParamsIndexer.Database.PostgreSQL.Username
			dbParams.Password = ParamsIndexer.Database.PostgreSQL.Password

		default:
			return nil, ierrors.Errorf("unknown database engine: %s, supported engines: %s", dbParams.Engine, db.GetSupportedEnginesString(indexer.AllowedEngines))
		}

		return indexer.NewIndexer(dbParams, Component.Logger)
	}); err != nil {
		return err
	}

	return c.Provide(func() *echo.Echo {
		return httpserver.NewEcho(
			Component.Logger,
			nil,
			ParamsRestAPI.DebugRequestLoggerEnabled,
		)
	})
}

func run() error {

	indexerInitWait := make(chan struct{})

	// create a background worker that handles the indexer events
	if err := Component.Daemon().BackgroundWorker("Indexer", func(ctx context.Context) {
		Component.LogInfo("Starting Indexer")
		defer func() {
			if err := deps.Indexer.CloseDatabase(); err != nil {
				Component.LogErrorf("Failed to close database: %s", err.Error())
			}
		}()

		indexerStatus, err := checkIndexerStatus(ctx)
		if err != nil {
			Component.LogFatalf("Checking initial Indexer state failed: %s", err.Error())

			return
		}
		close(indexerInitWait)

		Component.LogInfo("Starting LedgerUpdates ... done")

		if err := deps.NodeBridge.ListenToLedgerUpdates(ctx, indexerStatus.CommittedSlot+1, 0, func(update *nodebridge.LedgerUpdate) error {
			ts := time.Now()
			ledgerUpdate, err := LedgerUpdateFromNodeBridge(update)
			if err != nil {
				return err
			}
			if err := deps.Indexer.CommitLedgerUpdate(ledgerUpdate); err != nil {
				return err
			}

			Component.LogInfof("Applying slot %d with %d new and %d consumed outputs took %s", update.CommitmentID.Slot(), len(update.Created), len(update.Consumed), time.Since(ts).Truncate(time.Millisecond))

			return nil
		}); err != nil {
			deps.ShutdownHandler.SelfShutdown(fmt.Sprintf("Listening to LedgerUpdates failed, error: %s", err), false)
		}

		Component.LogInfo("Stopping LedgerUpdates ... done")
		Component.LogInfo("Stopped Indexer")
	}, daemon.PriorityStopIndexer); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	// create a background worker that handles the indexer events
	if err := Component.Daemon().BackgroundWorker("Indexer - AcceptedTransactions", func(ctx context.Context) {
		Component.LogInfo("Starting AcceptedTransactions")

		// we need to wait until the indexer is initialized before starting to listen to accepted transactions.
		select {
		case <-ctx.Done():
			return
		case <-indexerInitWait:
		}

		Component.LogInfo("Starting AcceptedTransactions ... done")
		if err := deps.NodeBridge.ListenToAcceptedTransactions(ctx, func(tx *nodebridge.AcceptedTransaction) error {
			ts := time.Now()
			ledgerUpdate, err := LedgerUpdateFromNodeBridgeAcceptedTransaction(tx)
			if err != nil {
				return err
			}
			if err := deps.Indexer.AcceptLedgerUpdate(ledgerUpdate); err != nil {
				return err
			}

			Component.LogInfof("Applying accepted transaction %s at slot %d with %d new and %d consumed outputs took %s", tx.TransactionID.ToHex(), tx.Slot, len(tx.Created), len(tx.Consumed), time.Since(ts).Truncate(time.Millisecond))

			return nil
		}); err != nil {
			deps.ShutdownHandler.SelfShutdown(fmt.Sprintf("Listening to AcceptedTransactions failed, error: %s", err), false)
		}

		Component.LogInfo("Stopping AcceptedTransactions ... done")
	}, daemon.PriorityStopIndexerAcceptedTransactions); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	// create a background worker that handles the API
	if err := Component.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		Component.LogInfo("Starting API")

		// we need to wait until the indexer is initialized before starting the API or the daemon is canceled before that is done.
		select {
		case <-ctx.Done():
			return
		case <-indexerInitWait:
		}
		Component.LogInfo("Starting API ... done")

		Component.LogInfo("Starting API server ...")

		_ = server.NewIndexerServer(deps.Indexer, deps.Echo, deps.NodeBridge, ParamsRestAPI.MaxPageSize)

		go func() {
			Component.LogInfof("You can now access the API using: http://%s", ParamsRestAPI.BindAddress)
			if err := deps.Echo.Start(ParamsRestAPI.BindAddress); err != nil && !ierrors.Is(err, http.ErrServerClosed) {
				Component.LogFatalf("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := ParamsRestAPI.BindAddress
		if ParamsRestAPI.AdvertiseAddress != "" {
			advertisedAddress = ParamsRestAPI.AdvertiseAddress
		}

		routeName := strings.Replace(server.APIRoute, "/api/", "", 1)
		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, routeName, advertisedAddress, server.APIRoute); err != nil {
			Component.LogFatalf("Registering INX api route failed: %s", err)
		}
		cancelRegister()

		Component.LogInfo("Starting API server ... done")
		<-ctx.Done()
		Component.LogInfo("Stopping API ...")

		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, routeName); err != nil {
			Component.LogWarnf("Unregistering INX api route failed: %s", err)
		}

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		if err := deps.Echo.Shutdown(shutdownCtx); err != nil {
			Component.LogWarn(err.Error())
		}

		Component.LogInfo("Stopping API ... done")
	}, daemon.PriorityStopIndexerAPI); err != nil {
		Component.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}

func checkIndexerStatus(ctx context.Context) (*indexer.Status, error) {
	var status *indexer.Status
	var err error

	needsToFillIndexer := false
	needsToClearIndexer := false

	nodeStatus := deps.NodeBridge.NodeStatus()

	if !deps.Indexer.IsInitialized() {
		// Starting indexer without a database
		if err := deps.Indexer.CreateTables(); err != nil {
			return nil, err
		}
		needsToFillIndexer = true
	} else {
		// Checking current indexer state to see if it needs a reset or not
		status, err = deps.Indexer.Status()
		if err != nil {
			if !ierrors.Is(err, indexer.ErrStatusNotFound) {
				return nil, fmt.Errorf("reading committedSlot from Indexer failed! Error: %w", err)
			}
			Component.LogInfo("Indexer is empty, so import initial ledger...")
			needsToFillIndexer = true
		} else {
			switch {
			case status.NetworkName != deps.NodeBridge.APIProvider().CommittedAPI().ProtocolParameters().NetworkName():
				Component.LogInfof("> Network name changed: %s vs %s", status.NetworkName, deps.NodeBridge.APIProvider().CommittedAPI().ProtocolParameters().NetworkName())
				needsToClearIndexer = true

			case status.DatabaseVersion != DBVersion:
				Component.LogInfof("> Indexer database version changed: %d vs %d", status.DatabaseVersion, DBVersion)
				needsToClearIndexer = true

			case nodeStatus.HasPruned && deps.NodeBridge.APIProvider().LatestAPI().TimeProvider().EpochStart(iotago.EpochIndex(nodeStatus.GetPruningEpoch())) > status.CommittedSlot:
				Component.LogInfo("> Node has an newer pruning slot than our current committedSlot")
				needsToClearIndexer = true
			}
		}
	}

	if needsToClearIndexer {
		Component.LogInfo("Re-import initial ledger...")
		if err := deps.Indexer.Clear(); err != nil {
			return nil, fmt.Errorf("clearing Indexer failed! Error: %w", err)
		}
		needsToFillIndexer = true
	}

	if needsToFillIndexer {
		// Indexer is empty, so import initial ledger state from the node
		timeStart := time.Now()
		var count int
		if count, err = fillIndexer(ctx, deps.Indexer); err != nil {
			return nil, fmt.Errorf("filling Indexer failed! Error: %w", err)
		}
		duration := time.Since(timeStart)
		// Read new committedSlot after filling up the indexer
		status, err = deps.Indexer.Status()
		if err != nil {
			return nil, fmt.Errorf("reading committedSlot from Indexer failed! Error: %w", err)
		}
		Component.LogInfo("Re-creating indexes")
		// Run auto migrate to make sure all required tables and indexes are there
		if err := deps.Indexer.AutoMigrate(); err != nil {
			return nil, err
		}
		Component.LogInfof("Importing initial ledger with %d outputs at slot %d took %s", count, status.CommittedSlot, duration.Truncate(time.Millisecond))
	} else {
		Component.LogInfo("Checking database schema")
		// Run auto migrate to make sure all required tables and indexes are there
		if err := deps.Indexer.AutoMigrate(); err != nil {
			return nil, err
		}

		Component.LogInfo("Cleaning up all uncommitted changes")
		// Clean up indexer from all uncommitted changes
		if err := deps.Indexer.RemoveUncommittedChanges(); err != nil {
			return nil, err
		}

		Component.LogInfof("> Indexer started at committedSlot %d", status.CommittedSlot)
	}

	return status, nil
}

func fillIndexer(ctx context.Context, indexer *indexer.Indexer) (int, error) {

	// Drop indexes to speed up data insertion
	if err := deps.Indexer.DropIndexes(); err != nil {
		return 0, err
	}

	importerCtx, importCancel := context.WithCancel(ctx)
	defer importCancel()

	importer := indexer.ImportTransaction(importerCtx)

	receiveCtx, receiveCancel := context.WithCancel(ctx)
	defer receiveCancel()

	stream, err := deps.NodeBridge.Client().ReadUnspentOutputs(receiveCtx, &inx.NoParams{})
	if err != nil {
		return 0, err
	}

	tsStart := time.Now()
	p := message.NewPrinter(language.English)
	var innerErr error
	var committedSlot iotago.SlotIndex
	var countReceive int
	go func() {
		for {
			unspentOutput, err := stream.Recv()
			if err != nil {
				if !ierrors.Is(err, io.EOF) {
					innerErr = err
				}
				receiveCancel()

				break
			}

			output := unspentOutput.GetOutput()
			slotBooked := iotago.SlotIndex(output.GetSlotBooked())

			unwrapped, err := output.UnwrapOutput(deps.NodeBridge.APIProvider().APIForSlot(slotBooked))
			if err != nil {
				innerErr = err
				receiveCancel()

				break
			}

			if err := importer.AddOutput(output.GetOutputId().Unwrap(), unwrapped, slotBooked); err != nil {
				innerErr = err
				receiveCancel()

				return
			}
			outputLedgerSlot := unspentOutput.GetLatestCommitmentId().Unwrap().Index()
			if committedSlot < outputLedgerSlot {
				committedSlot = outputLedgerSlot
			}

			countReceive++
			if countReceive%1_000_000 == 0 {
				Component.LogInfo(p.Sprintf("received total=%d @ %.2f per second", countReceive, float64(countReceive)/float64(time.Since(tsStart)/time.Second)))
			}
		}
	}()

	<-receiveCtx.Done()

	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	if innerErr != nil {
		return 0, innerErr
	}

	Component.LogInfo(p.Sprintf("received total=%d in %s @ %.2f per second", countReceive, time.Since(tsStart).Truncate(time.Millisecond), float64(countReceive)/float64(time.Since(tsStart)/time.Second)))

	if err := importer.Finalize(committedSlot, deps.NodeBridge.APIProvider().CommittedAPI().ProtocolParameters().NetworkName(), DBVersion); err != nil {
		return 0, err
	}

	return countReceive, nil
}

func LedgerUpdateFromNodeBridge(update *nodebridge.LedgerUpdate) (*indexer.LedgerUpdate, error) {
	consumed := make([]*indexer.LedgerOutput, len(update.Consumed))
	for i, output := range update.Consumed {
		consumed[i] = &indexer.LedgerOutput{
			OutputID: output.OutputID,
			Output:   output.Output,
			BookedAt: output.Metadata.Included.Slot,
			SpentAt:  output.Metadata.Spent.Slot,
		}
	}

	created := make([]*indexer.LedgerOutput, len(update.Created))
	for i, output := range update.Created {
		created[i] = &indexer.LedgerOutput{
			OutputID: output.OutputID,
			Output:   output.Output,
			BookedAt: output.Metadata.Included.Slot,
		}
	}

	return &indexer.LedgerUpdate{
		Slot:     update.CommitmentID.Slot(),
		Consumed: consumed,
		Created:  created,
	}, nil
}

func LedgerUpdateFromNodeBridgeAcceptedTransaction(tx *nodebridge.AcceptedTransaction) (*indexer.LedgerUpdate, error) {
	consumed := make([]*indexer.LedgerOutput, len(tx.Consumed))
	for i, output := range tx.Consumed {
		consumed[i] = &indexer.LedgerOutput{
			OutputID: output.OutputID,
			Output:   output.Output,
			BookedAt: output.Metadata.Included.Slot,
			SpentAt:  output.Metadata.Spent.Slot,
		}
	}

	created := make([]*indexer.LedgerOutput, len(tx.Created))
	for i, output := range tx.Created {
		created[i] = &indexer.LedgerOutput{
			OutputID: output.OutputID,
			Output:   output.Output,
			BookedAt: output.Metadata.Included.Slot,
		}
	}

	return &indexer.LedgerUpdate{
		Slot:     tx.Slot,
		Consumed: consumed,
		Created:  created,
	}, nil
}
