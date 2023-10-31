package indexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

func TestIndexer_FoundryOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAccountAddress := iotago_tpkg.RandAccountAddress()
	randomFoundryID, err := iotago.FoundryIDFromAddressAndSerialNumberAndTokenScheme(randomAccountAddress, 0, iotago.TokenSchemeSimple)
	require.NoError(t, err)

	accountAddress := iotago_tpkg.RandAccountAddress()
	foundryID, err := iotago.FoundryIDFromAddressAndSerialNumberAndTokenScheme(accountAddress, 0, iotago.TokenSchemeSimple)
	require.NoError(t, err)

	output := &iotago.FoundryOutput{
		Amount:       0,
		SerialNumber: 0,
		TokenScheme: &iotago.SimpleTokenScheme{
			MintedTokens:  iotago_tpkg.RandUint256(),
			MeltedTokens:  iotago_tpkg.RandUint256(),
			MaximumSupply: iotago_tpkg.RandUint256(),
		},
		Conditions: iotago.FoundryOutputUnlockConditions{
			&iotago.ImmutableAccountUnlockCondition{
				Address: accountAddress,
			},
		},
		Features:          iotago.FoundryOutputFeatures{},
		ImmutableFeatures: iotago.FoundryOutputImmFeatures{},
	}

	outputSet := ts.AddOutputOnCommitment(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireFoundryFoundByID(foundryID)
	outputSet.requireFoundryNotFoundByID(randomFoundryID)

	// Type
	outputSet.requireFoundryFound()
	outputSet.requireBasicNotFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()
	outputSet.requireAnchorNotFound()

	// Native Tokens
	outputSet.requireFoundryFound(indexer.FoundryHasNativeToken(false))
	outputSet.requireFoundryNotFound(indexer.FoundryHasNativeToken(true))

	outputSet.requireFoundryFound(indexer.FoundryNativeToken(foundryID)) // The foundry is always returned when filtering for that specific native token, even if it has no native token feature
	outputSet.requireFoundryNotFound(indexer.FoundryNativeToken(iotago_tpkg.RandNativeTokenID()))

	// Address
	outputSet.requireFoundryFound(indexer.FoundryWithAccountAddress(accountAddress))
	outputSet.requireFoundryNotFound(indexer.FoundryWithAccountAddress(randomAccountAddress))
}

func TestIndexer_FoundryOutput_NativeToken(t *testing.T) {
	ts := newTestSuite(t)

	accountAddress := iotago_tpkg.RandAccountAddress()
	foundryID, err := iotago.FoundryIDFromAddressAndSerialNumberAndTokenScheme(accountAddress, 0, iotago.TokenSchemeSimple)
	require.NoError(t, err)

	output := &iotago.FoundryOutput{
		Amount:       0,
		SerialNumber: 0,
		TokenScheme: &iotago.SimpleTokenScheme{
			MintedTokens:  iotago_tpkg.RandUint256(),
			MeltedTokens:  iotago_tpkg.RandUint256(),
			MaximumSupply: iotago_tpkg.RandUint256(),
		},
		Conditions: iotago.FoundryOutputUnlockConditions{
			&iotago.ImmutableAccountUnlockCondition{
				Address: accountAddress,
			},
		},
		Features: iotago.FoundryOutputFeatures{
			&iotago.NativeTokenFeature{
				ID:     foundryID,
				Amount: iotago_tpkg.RandUint256(),
			},
		},
		ImmutableFeatures: iotago.FoundryOutputImmFeatures{},
	}

	outputSet := ts.AddOutputOnCommitment(output, iotago_tpkg.RandOutputID(0))
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// Type
	outputSet.requireFoundryFound()
	outputSet.requireBasicNotFound()
	outputSet.requireAccountNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()

	// Native Tokens
	outputSet.requireFoundryFound(indexer.FoundryHasNativeToken(true))
	outputSet.requireFoundryNotFound(indexer.FoundryHasNativeToken(false))

	outputSet.requireFoundryFound(indexer.FoundryNativeToken(foundryID))
	outputSet.requireFoundryNotFound(indexer.FoundryNativeToken(iotago_tpkg.RandNativeTokenID()))
}
