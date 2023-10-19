package indexer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/inx-indexer/pkg/database"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
)

type indexerTestsuite struct {
	T       *testing.T
	Indexer *indexer.Indexer
}

type indexerOutputSet struct {
	ts      *indexerTestsuite
	Outputs iotago.OutputIDs
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

func (ts *indexerTestsuite) CurrentSlot() iotago.SlotIndex {
	status, err := ts.Indexer.Status()
	require.NoError(ts.T, err)

	return status.LedgerIndex
}

func (ts *indexerTestsuite) AddOutput(output iotago.Output, outputID iotago.OutputID) *indexerOutputSet {
	currentSlot := ts.CurrentSlot()

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

	require.NoError(ts.T, ts.Indexer.UpdatedLedger(update))

	return &indexerOutputSet{
		ts:      ts,
		Outputs: iotago.OutputIDs{outputID},
	}
}

func (ts *indexerTestsuite) DeleteOutput(outputID iotago.OutputID) {
	currentSlot := ts.CurrentSlot()

	update := &indexer.LedgerUpdate{
		Slot: currentSlot + 1,
		Consumed: []*indexer.LedgerOutput{
			{
				OutputID: outputID,
				SpentAt:  currentSlot + 1,
			},
		},
	}

	require.NoError(ts.T, ts.Indexer.UpdatedLedger(update))
}

func (os *indexerOutputSet) requireBasicFound(filters ...options.Option[indexer.BasicFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.Basic(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireBasicNotFound(filters ...options.Option[indexer.BasicFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.Basic(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireAccountFound(filters ...options.Option[indexer.AccountFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.Account(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireAccountNotFound(filters ...options.Option[indexer.AccountFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.Account(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireAccountFoundByID(accountID iotago.AccountID) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.AccountByID(accountID).OutputIDs)
}

func (os *indexerOutputSet) requireAccountNotFoundByID(accountID iotago.AccountID) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.AccountByID(accountID).OutputIDs)
}

func (os *indexerOutputSet) requireNFTFound(filters ...options.Option[indexer.NFTFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.NFT(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireNFTNotFound(filters ...options.Option[indexer.NFTFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.NFT(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireNFTFoundByID(nftID iotago.NFTID) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.NFTByID(nftID).OutputIDs)
}

func (os *indexerOutputSet) requireNFTNotFoundByID(nftID iotago.NFTID) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.NFTByID(nftID).OutputIDs)
}

func (os *indexerOutputSet) requireDelegationFound(filters ...options.Option[indexer.DelegationFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.Delegation(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireDelegationNotFound(filters ...options.Option[indexer.DelegationFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.Delegation(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireDelegationFoundByID(delegationID iotago.DelegationID) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.DelegationByID(delegationID).OutputIDs)
}

func (os *indexerOutputSet) requireDelegationNotFoundByID(delegationID iotago.DelegationID) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.DelegationByID(delegationID).OutputIDs)
}
