package indexer_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

func TestIndexer_NFTOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomNFTID := iotago_tpkg.RandNFTAddress().NFTID()

	address := iotago_tpkg.RandEd25519Address()
	storageReturnAddress := iotago_tpkg.RandEd25519Address()
	expirationReturnAddress := iotago_tpkg.RandEd25519Address()
	senderAddress := iotago_tpkg.RandEd25519Address()
	issuerAddress := iotago_tpkg.RandEd25519Address()
	tag := iotago_tpkg.RandBytes(20)

	output := &iotago.NFTOutput{
		Amount: iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.TestAPI.ProtocolParameters().TokenSupply()))),
		Mana:   iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		UnlockConditions: iotago.NFTOutputUnlockConditions{
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
		Features: iotago.NFTOutputFeatures{
			&iotago.SenderFeature{
				Address: senderAddress,
			},
			&iotago.TagFeature{
				Tag: tag,
			},
		},
		ImmutableFeatures: iotago.NFTOutputImmFeatures{
			&iotago.IssuerFeature{
				Address: issuerAddress,
			},
		},
	}

	outputID := iotago_tpkg.RandOutputID(0)
	nftID := iotago.NFTIDFromOutputID(outputID)

	outputSet := ts.AddOutputOnCommitment(output, outputID)
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireNFTFoundByID(nftID)
	outputSet.requireNFTNotFoundByID(randomNFTID)

	// Type
	outputSet.requireNFTFound()
	outputSet.requireBasicNotFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireFoundryNotFound()
	outputSet.requireAnchorNotFound()

	// Address
	outputSet.requireNFTFound(indexer.NFTUnlockAddress(address))
	outputSet.requireNFTNotFound(indexer.NFTUnlockAddress(randomAddress))

	// Sender
	outputSet.requireNFTFound(indexer.NFTSender(senderAddress))
	outputSet.requireNFTNotFound(indexer.NFTSender(randomAddress))

	// Issuer
	outputSet.requireNFTFound(indexer.NFTIssuer(issuerAddress))
	outputSet.requireNFTNotFound(indexer.NFTIssuer(randomAddress))

	// Storage Deposit Return
	outputSet.requireNFTFound(indexer.NFTHasStorageDepositReturnCondition(true))
	outputSet.requireNFTNotFound(indexer.NFTHasStorageDepositReturnCondition(false))

	outputSet.requireNFTFound(indexer.NFTStorageDepositReturnAddress(storageReturnAddress))
	outputSet.requireNFTNotFound(indexer.NFTStorageDepositReturnAddress(randomAddress))

	// Tag
	outputSet.requireNFTFound(indexer.NFTTag(tag))
	outputSet.requireNFTNotFound(indexer.NFTTag([]byte("otherTag")))

	// Creation Slot
	outputSet.requireNFTFound(indexer.NFTCreatedAfter(0))
	outputSet.requireNFTNotFound(indexer.NFTCreatedAfter(1))

	outputSet.requireNFTNotFound(indexer.NFTCreatedBefore(0))
	outputSet.requireNFTNotFound(indexer.NFTCreatedBefore(1))
	outputSet.requireNFTFound(indexer.NFTCreatedBefore(2))

	// Expiration
	outputSet.requireNFTFound(indexer.NFTHasExpirationCondition(true))
	outputSet.requireNFTNotFound(indexer.NFTHasExpirationCondition(false))

	outputSet.requireNFTFound(indexer.NFTExpirationReturnAddress(expirationReturnAddress))
	outputSet.requireNFTNotFound(indexer.NFTExpirationReturnAddress(randomAddress))

	outputSet.requireNFTNotFound(indexer.NFTExpiresBefore(6987))
	outputSet.requireNFTNotFound(indexer.NFTExpiresBefore(6988))
	outputSet.requireNFTFound(indexer.NFTExpiresBefore(6989))

	outputSet.requireNFTFound(indexer.NFTExpiresAfter(6987))
	outputSet.requireNFTNotFound(indexer.NFTExpiresAfter(6988))
	outputSet.requireNFTNotFound(indexer.NFTExpiresAfter(6989))

	// Timelock
	outputSet.requireNFTFound(indexer.NFTHasTimelockCondition(true))
	outputSet.requireNFTNotFound(indexer.NFTHasTimelockCondition(false))

	outputSet.requireNFTNotFound(indexer.NFTTimelockedBefore(6899))
	outputSet.requireNFTNotFound(indexer.NFTTimelockedBefore(6900))
	outputSet.requireNFTFound(indexer.NFTTimelockedBefore(6901))

	outputSet.requireNFTFound(indexer.NFTTimelockedAfter(6899))
	outputSet.requireNFTNotFound(indexer.NFTTimelockedAfter(6900))
	outputSet.requireNFTNotFound(indexer.NFTTimelockedAfter(6901))

	//TODO: storageReturnAddress should not unlock it. Maybe fix this or clear up the naming

	// Unlockable by the following addresses
	for _, addr := range []iotago.Address{address, expirationReturnAddress, storageReturnAddress} {
		outputSet.requireNFTFound(indexer.NFTUnlockableByAddress(addr))
	}

	// Not unlockable by the following addresses
	for _, addr := range []iotago.Address{senderAddress, issuerAddress, randomAddress} {
		outputSet.requireNFTNotFound(indexer.NFTUnlockableByAddress(addr))
	}
}
