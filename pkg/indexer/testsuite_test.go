package indexer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/ds/shrinkingmap"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/log"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/inx-indexer/pkg/database"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

type indexerTestsuite struct {
	T       *testing.T
	Indexer *indexer.Indexer

	committedOutputs *shrinkingmap.ShrinkingMap[iotago.OutputID, iotago.Output]
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

	rootLogger := log.NewLogger()

	idx, err := indexer.NewIndexer(dbParams, rootLogger.NewChildLogger(t.Name()))
	require.NoError(t, err)

	require.NoError(t, idx.CreateTables())

	tx := idx.ImportTransaction(context.Background())
	require.NoError(t, tx.Finalize(0, t.Name(), 1))

	require.NoError(t, idx.AutoMigrate())

	return &indexerTestsuite{
		T:                t,
		Indexer:          idx,
		committedOutputs: shrinkingmap.New[iotago.OutputID, iotago.Output](),
	}
}

func (ts *indexerTestsuite) CurrentSlot() iotago.SlotIndex {
	status, err := ts.Indexer.Status()
	require.NoError(ts.T, err)

	return status.CommittedSlot
}

func (ts *indexerTestsuite) CommitEmptyLedgerUpdate() {
	committedSlot := ts.CurrentSlot() + 1

	update := &indexer.LedgerUpdate{
		Slot: committedSlot,
	}

	require.NoError(ts.T, ts.Indexer.CommitLedgerUpdate(update))
}

func (ts *indexerTestsuite) AddOutputOnCommitment(output iotago.Output, outputID iotago.OutputID) *indexerOutputSet {
	committedSlot := ts.CurrentSlot() + 1

	update := &indexer.LedgerUpdate{
		Slot: committedSlot,
		Created: []*indexer.LedgerOutput{
			{
				OutputID: outputID,
				Output:   output,
				BookedAt: committedSlot,
			},
		},
	}

	require.NoError(ts.T, ts.Indexer.CommitLedgerUpdate(update))

	ts.committedOutputs.Set(outputID, output)

	return &indexerOutputSet{
		ts:      ts,
		Outputs: iotago.OutputIDs{outputID},
	}
}

func (ts *indexerTestsuite) AddOutputOnAcceptance(output iotago.Output, outputID iotago.OutputID, slot iotago.SlotIndex) *indexerOutputSet {
	ts.committedOutputs.Set(outputID, output)

	update := &indexer.LedgerUpdate{
		Slot: slot,
		Created: []*indexer.LedgerOutput{
			{
				OutputID: outputID,
				Output:   output,
				BookedAt: slot,
			},
		},
	}

	require.NoError(ts.T, ts.Indexer.AcceptLedgerUpdate(update))

	return &indexerOutputSet{
		ts:      ts,
		Outputs: iotago.OutputIDs{outputID},
	}
}

func (ts *indexerTestsuite) DeleteOutputOnCommitment(outputID iotago.OutputID) {
	committedSlot := ts.CurrentSlot() + 1

	output, found := ts.committedOutputs.DeleteAndReturn(outputID)
	if !found {
		ts.T.Fatalf("output not found: %s", outputID)
	}

	update := &indexer.LedgerUpdate{
		Slot: committedSlot,
		Consumed: []*indexer.LedgerOutput{
			{
				OutputID: outputID,
				Output:   output,
				SpentAt:  committedSlot,
			},
		},
	}

	require.NoError(ts.T, ts.Indexer.CommitLedgerUpdate(update))
}

func (ts *indexerTestsuite) DeleteOutputOnAcceptance(outputID iotago.OutputID, slot iotago.SlotIndex) {
	output, found := ts.committedOutputs.Get(outputID)
	if !found {
		ts.T.Fatalf("output not found: %s", outputID)
	}

	update := &indexer.LedgerUpdate{
		Slot: slot,
		Consumed: []*indexer.LedgerOutput{
			{
				OutputID: outputID,
				Output:   output,
				SpentAt:  slot,
			},
		},
	}

	require.NoError(ts.T, ts.Indexer.AcceptLedgerUpdate(update))
}

func (ts *indexerTestsuite) MultiAddressExists(multiAddress *iotago.MultiAddress) bool {
	multiAddressBech32 := multiAddress.Bech32(iotago_tpkg.ZeroCostTestAPI.ProtocolParameters().Bech32HRP())

	_, parsedAddr, err := iotago.ParseBech32(multiAddressBech32)
	require.NoError(ts.T, err)

	multiAddressRef := parsedAddr.(*iotago.MultiAddressReference)

	fetchedAddress, err := ts.Indexer.MultiAddressForReference(multiAddressRef)
	if err != nil {
		if ierrors.Is(err, indexer.ErrMultiAddressNotFound) {
			return false
		}
		require.NoError(ts.T, err)
	}

	return multiAddress.Equal(fetchedAddress)
}

func (ts *indexerTestsuite) requireFound(outputID iotago.OutputID) {
	require.Contains(ts.T, ts.Indexer.Combined().OutputIDs, outputID)
}

func (ts *indexerTestsuite) requireNotFound(outputID iotago.OutputID) {
	require.NotContains(ts.T, ts.Indexer.Combined().OutputIDs, outputID)
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

func (os *indexerOutputSet) requireAnchorFound(filters ...options.Option[indexer.AnchorFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.Anchor(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireAnchorNotFound(filters ...options.Option[indexer.AnchorFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.Anchor(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireAnchorFoundByID(anchorID iotago.AnchorID) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.AnchorByID(anchorID).OutputIDs)
}

func (os *indexerOutputSet) requireAnchorNotFoundByID(anchorID iotago.AnchorID) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.AnchorByID(anchorID).OutputIDs)
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

func (os *indexerOutputSet) requireFoundryFound(filters ...options.Option[indexer.FoundryFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.Foundry(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireFoundryNotFound(filters ...options.Option[indexer.FoundryFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.Foundry(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireFoundryFoundByID(foundryID iotago.FoundryID) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.FoundryByID(foundryID).OutputIDs)
}

func (os *indexerOutputSet) requireFoundryNotFoundByID(foundryID iotago.FoundryID) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.FoundryByID(foundryID).OutputIDs)
}
