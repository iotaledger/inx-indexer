package main

const (
	Name = "snap-to-db"
)

func main() {
	/*
		if err := convert(); err != nil {
			if !ierrors.Is(err, flag.ErrHelp) {
				fmt.Printf("\nERROR: %s\n", err.Error())
			}
			os.Exit(1)
		}
	*/
	panic("This tool needs to be updated to the iota-core snapshot format")
}

type ParametersSnapshot struct {
	FullPath string `default:"testnet/snapshots/full_snapshot.bin" usage:"path to the full snapshot file"`
}

func convert() error {
	panic("This tool needs to be updated to the iota-core snapshot format")
	/*
		indexerParams := indexerComponent.ParametersIndexer{}
		dbParams := &indexerParams.Database
		snapshotParams := &ParametersSnapshot{}

		config := configuration.New()
		fs := configuration.NewUnsortedFlagSet("db", flag.ExitOnError)

		fs.Usage = func() {
			_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", Name)
			fs.PrintDefaults()
		}

		config.BindParameters(fs, "db", dbParams)
		config.BindParameters(fs, "snapshot", snapshotParams)
		if err := fs.Parse(os.Args[1:]); err != nil {
			return err
		}
		if err := config.LoadFlagSet(fs); err != nil {
			return err
		}
		config.UpdateBoundParameters()
		if err := logger.InitGlobalLogger(config); err != nil {
			return err
		}

		log := logger.NewLogger("snap-to-db")

		engine, err := database.EngineFromStringAllowed(dbParams.Engine, database.EngineSQLite, database.EnginePostgreSQL)
		if err != nil {
			return err
		}

		// open snapshot file
		snapshotFileRead, err := os.OpenFile(snapshotParams.FullPath, os.O_RDONLY, 0666)
		if err != nil {
			return err
		}

		indexerDBParams := database.Params{
			Engine: engine,
		}

		//nolint:exhaustive // we already checked the values is one of the valid ones
		switch engine {
		case database.EngineSQLite:
			indexerDBParams.Path = dbParams.SQLite.Path

		case database.EnginePostgreSQL:
			indexerDBParams.Host = dbParams.PostgreSQL.Host
			indexerDBParams.Port = dbParams.PostgreSQL.Port
			indexerDBParams.Database = dbParams.PostgreSQL.Database
			indexerDBParams.Username = dbParams.PostgreSQL.Username
			indexerDBParams.Password = dbParams.PostgreSQL.Password
		}

		config.Print()

		idx, err := indexer.NewIndexer(indexerDBParams, log)
		if err != nil {
			return err
		}

		if idx.IsInitialized() {
			return ierrors.New("indexer database already initialized")
		}

		log.Info("> Creating tables ...")
		if err := idx.CreateTables(); err != nil {
			return err
		}

		// Drop indexes to speed up data insertion
		log.Info("> Dropping indexes to speed up insertion ...")
		if err := idx.DropIndexes(); err != nil {
			return err
		}

		importer := idx.ImportTransaction(context.Background())

		ts := time.Now()
		log.Info("> Importing snapshot ...")
		var count int
		var snapshotHeader *snapshot.FullSnapshotHeader
		if err := snapshot.StreamFullSnapshotDataFrom(
			context.Background(),
			snapshotFileRead,
			func(h *snapshot.FullSnapshotHeader) error {
				snapshotHeader = h
				return nil
			},
			func(output *utxo.TreasuryOutput) error {
				return nil
			},
			func(output *utxo.Output) error {
				count++
				return importer.AddOutput(output.OutputID(), output.Output(), output.MilestoneTimestampBooked())
			},
			func(milestoneDiff *snapshot.MilestoneDiff) error {
				return nil
			},
			func(id iotago.BlockID, index iotago.MilestoneIndex) error {
				return nil
			},
			func(opt *iotago.ProtocolParamsMilestoneOpt) error {
				return nil
			},
		); err != nil {
			return err
		}
		protoParams, err := snapshotHeader.ProtocolParameters()
		if err != nil {
			return err
		}
		if err := importer.Finalize(snapshotHeader.LedgerMilestoneIndex, protoParams); err != nil {
			return err
		}

		log.Info("> Creating indexes ...")
		if err := idx.AutoMigrate(); err != nil {
			return err
		}

		status, err := idx.Status()
		if err != nil {
			return err
		}

		log.Infof("> Importing initial ledger with %d outputs at index %d took %s", count, status.LedgerIndex, time.Since(ts).Truncate(time.Millisecond))

		return idx.CloseDatabase()
	*/
}
