package indexer_test

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/inx-indexer/pkg/database"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
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

func (os *indexerOutputSet) requireFoundWithBasicFilters(filters ...options.Option[indexer.BasicOutputFilterOptions]) {
	require.Equal(os.ts.T, os.Outputs, os.ts.Indexer.BasicOutputsWithFilters(filters...).OutputIDs)
}

func (os *indexerOutputSet) requireNotFoundWithBasicFilters(filters ...options.Option[indexer.BasicOutputFilterOptions]) {
	require.NotEqual(os.ts.T, os.Outputs, os.ts.Indexer.BasicOutputsWithFilters(filters...).OutputIDs)
}

func TestIndexer_BasicOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()

	address := iotago_tpkg.RandEd25519Address()
	storageReturnAddress := iotago_tpkg.RandEd25519Address()
	expirationReturnAddress := iotago_tpkg.RandEd25519Address()
	senderAddress := iotago_tpkg.RandEd25519Address()
	tag := iotago_tpkg.RandBytes(20)

	output := &iotago.BasicOutput{
		Amount: iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.TestAPI.ProtocolParameters().TokenSupply()))),
		Mana:   iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		Conditions: iotago.BasicOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
			&iotago.StorageDepositReturnUnlockCondition{
				ReturnAddress: storageReturnAddress,
				Amount:        65586,
			},
			&iotago.ExpirationUnlockCondition{
				ReturnAddress: expirationReturnAddress,
				Slot:          6988,
			},
			&iotago.TimelockUnlockCondition{
				Slot: 6900,
			},
		},
		Features: iotago.BasicOutputFeatures{
			&iotago.SenderFeature{
				Address: senderAddress,
			},
			&iotago.TagFeature{
				Tag: tag,
			},
		},
	}

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())
	outputSet.requireFoundWithBasicFilters()

	// Check if the output is indexed correctly
	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputHasNativeToken(false))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputHasNativeToken(true))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputNativeToken(iotago_tpkg.RandNativeTokenID()))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputUnlockAddress(address))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputUnlockAddress(randomAddress))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputSender(senderAddress))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputSender(randomAddress))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputExpirationReturnAddress(expirationReturnAddress))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputExpirationReturnAddress(randomAddress))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputStorageDepositReturnAddress(storageReturnAddress))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputStorageDepositReturnAddress(randomAddress))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputTag(tag))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputTag([]byte("otherTag")))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputCreatedAfter(0))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputCreatedAfter(1))

	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputCreatedBefore(0))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputCreatedBefore(1))
	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputCreatedBefore(2))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputHasExpirationCondition(true))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputHasExpirationCondition(false))

	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputExpiresBefore(6987))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputExpiresBefore(6988))
	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputExpiresBefore(6989))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputExpiresAfter(6987))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputExpiresAfter(6988))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputExpiresAfter(6989))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputHasTimelockCondition(true))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputHasTimelockCondition(false))

	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputTimelockedBefore(6899))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputTimelockedBefore(6900))
	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputTimelockedBefore(6901))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputTimelockedAfter(6899))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputTimelockedAfter(6900))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputTimelockedAfter(6901))

	//TODO: storageReturnAddress should not unlock it. Maybe fix this or clear up the naming

	// Unlockable by the following addresses
	for _, addr := range []iotago.Address{address, expirationReturnAddress, storageReturnAddress} {
		outputSet.requireFoundWithBasicFilters(indexer.BasicOutputUnlockableByAddress(addr))
	}

	// Not unlockable by the following addresses
	for _, addr := range []iotago.Address{senderAddress, randomAddress} {
		outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputUnlockableByAddress(addr))
	}
}

func TestIndexer_BasicOutput_NativeToken(t *testing.T) {
	ts := newTestSuite(t)

	address := iotago_tpkg.RandEd25519Address()
	nativeTokenID := iotago_tpkg.RandNativeTokenID()

	output := &iotago.BasicOutput{
		Amount: iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.TestAPI.ProtocolParameters().TokenSupply()))),
		Mana:   iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		Conditions: iotago.BasicOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
		Features: iotago.BasicOutputFeatures{
			&iotago.NativeTokenFeature{
				ID:     nativeTokenID,
				Amount: iotago_tpkg.RandUint256(),
			},
		},
	}

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())
	outputSet.requireFoundWithBasicFilters()

	// Check if the output is indexed correctly
	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputHasNativeToken(true))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputHasNativeToken(false))

	outputSet.requireFoundWithBasicFilters(indexer.BasicOutputNativeToken(nativeTokenID))
	outputSet.requireNotFoundWithBasicFilters(indexer.BasicOutputNativeToken(iotago_tpkg.RandNativeTokenID()))
}
