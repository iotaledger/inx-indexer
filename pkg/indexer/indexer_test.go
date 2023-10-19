package indexer_test

import (
	"crypto/ed25519"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	hive_ed25519 "github.com/iotaledger/hive.go/crypto/ed25519"
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

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// Type
	outputSet.requireBasicFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()

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

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// Type
	outputSet.requireBasicFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()

	// Native Tokens
	outputSet.requireBasicFound(indexer.BasicHasNativeToken(true))
	outputSet.requireBasicNotFound(indexer.BasicHasNativeToken(false))

	outputSet.requireBasicFound(indexer.BasicNativeToken(nativeTokenID))
	outputSet.requireBasicNotFound(indexer.BasicNativeToken(iotago_tpkg.RandNativeTokenID()))
}

func TestIndexer_AccountOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomAccountID := iotago_tpkg.RandAccountAddress().AccountID()

	accountAddress := iotago_tpkg.RandAccountAddress()
	senderAddress := iotago_tpkg.RandEd25519Address()
	issuerAddress := iotago_tpkg.RandEd25519Address()
	stateControllerAddress := iotago_tpkg.RandEd25519Address()
	governorAddress := iotago_tpkg.RandEd25519Address()

	output := &iotago.AccountOutput{
		Amount:         iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.TestAPI.ProtocolParameters().TokenSupply()))),
		Mana:           iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		AccountID:      accountAddress.AccountID(),
		StateIndex:     0,
		StateMetadata:  nil,
		FoundryCounter: 0,
		Conditions: iotago.AccountOutputUnlockConditions{
			&iotago.StateControllerAddressUnlockCondition{
				Address: stateControllerAddress,
			},
			&iotago.GovernorAddressUnlockCondition{
				Address: governorAddress,
			},
		},
		Features: iotago.AccountOutputFeatures{
			&iotago.SenderFeature{
				Address: senderAddress,
			},
			&iotago.StakingFeature{
				StakedAmount: 6598,
				FixedCost:    0,
				StartEpoch:   0,
				EndEpoch:     0,
			},
			&iotago.BlockIssuerFeature{
				BlockIssuerKeys: iotago.BlockIssuerKeys{
					&iotago.Ed25519PublicKeyBlockIssuerKey{
						PublicKey: hive_ed25519.PublicKey(iotago_tpkg.RandEd25519PrivateKey().Public().(ed25519.PublicKey)),
					},
				},
				ExpirySlot: 0,
			},
		},
		ImmutableFeatures: iotago.AccountOutputImmFeatures{
			&iotago.IssuerFeature{
				Address: issuerAddress,
			},
		},
	}

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireAccountFoundByID(accountAddress.AccountID())
	outputSet.requireAccountNotFoundByID(randomAccountID)

	// Type
	outputSet.requireAccountFound()
	outputSet.requireBasicNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()

	// Creation Slot
	outputSet.requireAccountFound(indexer.AccountCreatedAfter(0))
	outputSet.requireAccountNotFound(indexer.AccountCreatedAfter(1))

	outputSet.requireAccountNotFound(indexer.AccountCreatedBefore(0))
	outputSet.requireAccountNotFound(indexer.AccountCreatedBefore(1))
	outputSet.requireAccountFound(indexer.AccountCreatedBefore(2))

	// State Controller
	outputSet.requireAccountFound(indexer.AccountStateController(stateControllerAddress))
	outputSet.requireAccountNotFound(indexer.AccountStateController(governorAddress))

	// Governor
	outputSet.requireAccountFound(indexer.AccountGovernor(governorAddress))
	outputSet.requireAccountNotFound(indexer.AccountGovernor(stateControllerAddress))

	// Sender
	outputSet.requireAccountFound(indexer.AccountSender(senderAddress))
	outputSet.requireAccountNotFound(indexer.AccountSender(randomAddress))

	// Issuer
	outputSet.requireAccountFound(indexer.AccountIssuer(issuerAddress))
	outputSet.requireAccountNotFound(indexer.AccountIssuer(randomAddress))

	// Unlockable by the following addresses
	for _, addr := range []iotago.Address{stateControllerAddress, governorAddress} {
		outputSet.requireAccountFound(indexer.AccountUnlockableByAddress(addr))
	}

	// Not unlockable by the following addresses
	for _, addr := range []iotago.Address{senderAddress, issuerAddress, accountAddress} {
		outputSet.requireAccountNotFound(indexer.AccountUnlockableByAddress(addr))
	}
}

func TestIndexer_DelegationOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomValidatorAddress := iotago_tpkg.RandAccountAddress()
	randomDelegationID := iotago_tpkg.RandDelegationID()

	address := iotago_tpkg.RandEd25519Address()
	validatorAddress := iotago_tpkg.RandAccountAddress()
	delegationID := iotago_tpkg.RandDelegationID()
	amount := iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.TestAPI.ProtocolParameters().TokenSupply())))

	output := &iotago.DelegationOutput{
		Amount:           amount,
		DelegatedAmount:  amount,
		DelegationID:     delegationID,
		ValidatorAddress: validatorAddress,
		StartEpoch:       0,
		EndEpoch:         0,
		Conditions: iotago.DelegationOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
	}

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireDelegationFoundByID(delegationID)
	outputSet.requireDelegationNotFoundByID(randomDelegationID)

	// Type
	outputSet.requireDelegationFound()
	outputSet.requireAccountNotFound()
	outputSet.requireBasicNotFound()
	outputSet.requireNFTNotFound()

	// Creation Slot
	outputSet.requireDelegationFound(indexer.DelegationCreatedAfter(0))
	outputSet.requireDelegationNotFound(indexer.DelegationCreatedAfter(1))

	outputSet.requireDelegationNotFound(indexer.DelegationCreatedBefore(0))
	outputSet.requireDelegationNotFound(indexer.DelegationCreatedBefore(1))
	outputSet.requireDelegationFound(indexer.DelegationCreatedBefore(2))

	// Address
	outputSet.requireDelegationFound(indexer.DelegationAddress(address))
	outputSet.requireDelegationNotFound(indexer.DelegationAddress(randomAddress))

	// Validator
	outputSet.requireDelegationFound(indexer.DelegationValidator(validatorAddress))
	outputSet.requireDelegationNotFound(indexer.DelegationValidator(randomValidatorAddress))
}

func TestIndexer_NFTOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomNFTID := iotago_tpkg.RandNFTAddress().NFTID()

	nftID := iotago_tpkg.RandNFTAddress().NFTID()
	address := iotago_tpkg.RandEd25519Address()
	storageReturnAddress := iotago_tpkg.RandEd25519Address()
	expirationReturnAddress := iotago_tpkg.RandEd25519Address()
	senderAddress := iotago_tpkg.RandEd25519Address()
	issuerAddress := iotago_tpkg.RandEd25519Address()
	tag := iotago_tpkg.RandBytes(20)

	output := &iotago.NFTOutput{
		Amount: iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.TestAPI.ProtocolParameters().TokenSupply()))),
		Mana:   iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		NFTID:  nftID,
		Conditions: iotago.NFTOutputUnlockConditions{
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

	outputSet := ts.AddOutput(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireNFTFoundByID(nftID)
	outputSet.requireNFTNotFoundByID(randomNFTID)

	// Type
	outputSet.requireNFTFound()
	outputSet.requireBasicNotFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()

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
