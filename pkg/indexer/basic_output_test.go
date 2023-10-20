package indexer_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

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

	outputSet := ts.AddOutputOnCommitment(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// Type
	outputSet.requireBasicFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()
	outputSet.requireFoundryNotFound()

	// Native Tokens
	outputSet.requireBasicFound(indexer.BasicHasNativeToken(false))
	outputSet.requireBasicNotFound(indexer.BasicHasNativeToken(true))
	outputSet.requireBasicNotFound(indexer.BasicNativeToken(iotago_tpkg.RandNativeTokenID()))

	// Address
	outputSet.requireBasicFound(indexer.BasicUnlockAddress(address))
	outputSet.requireBasicNotFound(indexer.BasicUnlockAddress(randomAddress))

	// Sender
	outputSet.requireBasicFound(indexer.BasicSender(senderAddress))
	outputSet.requireBasicNotFound(indexer.BasicSender(randomAddress))

	// Storage Deposit Return
	outputSet.requireBasicFound(indexer.BasicHasStorageDepositReturnCondition(true))
	outputSet.requireBasicNotFound(indexer.BasicHasStorageDepositReturnCondition(false))

	outputSet.requireBasicFound(indexer.BasicStorageDepositReturnAddress(storageReturnAddress))
	outputSet.requireBasicNotFound(indexer.BasicStorageDepositReturnAddress(randomAddress))

	// Tag
	outputSet.requireBasicFound(indexer.BasicTag(tag))
	outputSet.requireBasicNotFound(indexer.BasicTag([]byte("otherTag")))

	// Creation Slot
	outputSet.requireBasicFound(indexer.BasicCreatedAfter(0))
	outputSet.requireBasicNotFound(indexer.BasicCreatedAfter(1))

	outputSet.requireBasicNotFound(indexer.BasicCreatedBefore(0))
	outputSet.requireBasicNotFound(indexer.BasicCreatedBefore(1))
	outputSet.requireBasicFound(indexer.BasicCreatedBefore(2))

	// Expiration
	outputSet.requireBasicFound(indexer.BasicExpirationReturnAddress(expirationReturnAddress))
	outputSet.requireBasicNotFound(indexer.BasicExpirationReturnAddress(randomAddress))

	outputSet.requireBasicFound(indexer.BasicHasExpirationCondition(true))
	outputSet.requireBasicNotFound(indexer.BasicHasExpirationCondition(false))

	outputSet.requireBasicNotFound(indexer.BasicExpiresBefore(6987))
	outputSet.requireBasicNotFound(indexer.BasicExpiresBefore(6988))
	outputSet.requireBasicFound(indexer.BasicExpiresBefore(6989))

	outputSet.requireBasicFound(indexer.BasicExpiresAfter(6987))
	outputSet.requireBasicNotFound(indexer.BasicExpiresAfter(6988))
	outputSet.requireBasicNotFound(indexer.BasicExpiresAfter(6989))

	// Timelock
	outputSet.requireBasicFound(indexer.BasicHasTimelockCondition(true))
	outputSet.requireBasicNotFound(indexer.BasicHasTimelockCondition(false))

	outputSet.requireBasicNotFound(indexer.BasicTimelockedBefore(6899))
	outputSet.requireBasicNotFound(indexer.BasicTimelockedBefore(6900))
	outputSet.requireBasicFound(indexer.BasicTimelockedBefore(6901))

	outputSet.requireBasicFound(indexer.BasicTimelockedAfter(6899))
	outputSet.requireBasicNotFound(indexer.BasicTimelockedAfter(6900))
	outputSet.requireBasicNotFound(indexer.BasicTimelockedAfter(6901))

	//TODO: storageReturnAddress should not unlock it. Maybe fix this or clear up the naming

	// Unlockable by the following addresses
	for _, addr := range []iotago.Address{address, expirationReturnAddress, storageReturnAddress} {
		outputSet.requireBasicFound(indexer.BasicUnlockableByAddress(addr))
	}

	// Not unlockable by the following addresses
	for _, addr := range []iotago.Address{senderAddress, randomAddress} {
		outputSet.requireBasicNotFound(indexer.BasicUnlockableByAddress(addr))
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

	outputSet := ts.AddOutputOnCommitment(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// Type
	outputSet.requireBasicFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()
	outputSet.requireFoundryNotFound()

	// Native Tokens
	outputSet.requireBasicFound(indexer.BasicHasNativeToken(true))
	outputSet.requireBasicNotFound(indexer.BasicHasNativeToken(false))

	outputSet.requireBasicFound(indexer.BasicNativeToken(nativeTokenID))
	outputSet.requireBasicNotFound(indexer.BasicNativeToken(iotago_tpkg.RandNativeTokenID()))
}
