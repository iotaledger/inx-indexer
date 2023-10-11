package indexer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/inx-indexer/pkg/database"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

type indexerTestsuite struct {
	T       *testing.T
	Indexer *indexer.Indexer
}

func newTestSuite(t *testing.T) *indexerTestsuite {
	dbParams := database.Params{
		Engine: database.EngineSQLite,
		Path:   t.TempDir(),
	}

	rootLogger, err := logger.NewRootLogger(logger.DefaultCfg)
	require.NoError(t, err)

	idx, err := indexer.NewIndexer(dbParams, rootLogger.Named(t.Name()))
	require.NoError(t, err)

	require.NoError(t, idx.CreateTables())

	tx := idx.ImportTransaction(context.Background())
	require.NoError(t, tx.Finalize(0, t.Name(), 1))

	require.NoError(t, idx.AutoMigrate())

	return &indexerTestsuite{
		T:       t,
		Indexer: idx,
	}
}

func (t *indexerTestsuite) CurrentSlot() iotago.SlotIndex {
	status, err := t.Indexer.Status()
	require.NoError(t.T, err)

	return status.LedgerIndex
}

func (t *indexerTestsuite) AddOutput(output iotago.Output, outputID iotago.OutputID) {
	currentSlot := t.CurrentSlot()

	update := &indexer.LedgerUpdate{
		Slot: currentSlot + 1,
		Created: []*indexer.LedgerOutput{
			{
				OutputID:  outputID,
				Output:    output,
				CreatedAt: currentSlot + 1,
			},
		},
	}

	require.NoError(t.T, t.Indexer.UpdatedLedger(update))
}

func (t *indexerTestsuite) DeleteOutput(outputID iotago.OutputID) {
	currentSlot := t.CurrentSlot()

	update := &indexer.LedgerUpdate{
		Slot: currentSlot + 1,
		Consumed: []*indexer.LedgerOutput{
			{
				OutputID: outputID,
				SpentAt:  currentSlot + 1,
			},
		},
	}

	require.NoError(t.T, t.Indexer.UpdatedLedger(update))
}

func TestIndexer_BasicOutput(t *testing.T) {
	ts := newTestSuite(t)

	output := iotago_tpkg.RandBasicOutput(iotago.AddressEd25519)
	outputID := iotago_tpkg.RandOutputID(0)

	ts.AddOutput(output, outputID)

	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters().OutputIDs)
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputUnlockableByAddress(output.UnlockConditionSet().Address().Address)).OutputIDs)
}
