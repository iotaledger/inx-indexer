package indexer_test

import (
	"context"
	"math"
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
	outputID := iotago_tpkg.RandOutputID(0)

	ts.AddOutput(output, outputID)

	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters().OutputIDs)

	// Check if the output is indexed correctly
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasNativeToken(true)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputNativeToken(iotago_tpkg.RandNativeTokenID())).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputUnlockAddress(address)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputUnlockAddress(randomAddress)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputSender(senderAddress)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputSender(randomAddress)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpirationReturnAddress(expirationReturnAddress)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpirationReturnAddress(randomAddress)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputStorageDepositReturnAddress(storageReturnAddress)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputStorageDepositReturnAddress(randomAddress)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTag(tag)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTag([]byte("otherTag"))).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputCreatedAfter(0)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputCreatedAfter(1)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputCreatedBefore(0)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputCreatedBefore(1)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputCreatedBefore(2)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasExpirationCondition(true)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasExpirationCondition(false)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpiresBefore(6987)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpiresBefore(6988)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpiresBefore(6989)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpiresAfter(6987)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpiresAfter(6988)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputExpiresAfter(6989)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasTimelockCondition(true)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasTimelockCondition(false)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTimelockedBefore(6899)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTimelockedBefore(6900)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTimelockedBefore(6901)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTimelockedAfter(6899)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTimelockedAfter(6900)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputTimelockedAfter(6901)).OutputIDs)

	//TODO: storageReturnAddress should not unlock it. Maybe fix this or clear up the naming

	// Unlockable by the following addresses
	for _, addr := range []iotago.Address{address, expirationReturnAddress, storageReturnAddress} {
		require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputUnlockableByAddress(addr)).OutputIDs)
	}

	// Not unlockable by the following addresses
	for _, addr := range []iotago.Address{senderAddress, randomAddress} {
		require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputUnlockableByAddress(addr)).OutputIDs)
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
	outputID := iotago_tpkg.RandOutputID(0)

	ts.AddOutput(output, outputID)

	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters().OutputIDs)

	// Check if the output is indexed correctly
	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasNativeToken(true)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputHasNativeToken(false)).OutputIDs)

	require.Equal(t, iotago.OutputIDs{outputID}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputNativeToken(nativeTokenID)).OutputIDs)
	require.Equal(t, iotago.OutputIDs{}, ts.Indexer.BasicOutputsWithFilters(indexer.BasicOutputNativeToken(iotago_tpkg.RandNativeTokenID())).OutputIDs)

}
