package indexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

func TestIndexer_DelegationOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomValidatorAddress := iotago_tpkg.RandAccountAddress()
	randomDelegationID := iotago_tpkg.RandDelegationID()

	address := iotago_tpkg.RandEd25519Address()
	validatorAddress := iotago_tpkg.RandAccountAddress()
	amount := iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.ZeroCostTestAPI.ProtocolParameters().TokenSupply())))

	output := &iotago.DelegationOutput{
		Amount:           amount,
		DelegatedAmount:  amount,
		ValidatorAddress: validatorAddress,
		StartEpoch:       0,
		EndEpoch:         0,
		UnlockConditions: iotago.DelegationOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
	}

	outputID := iotago_tpkg.RandOutputID(0)
	delegationID := iotago.DelegationIDFromOutputID(outputID)

	outputSet := ts.AddOutputOnCommitment(output, outputID)
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireDelegationFoundByID(delegationID)
	outputSet.requireDelegationNotFoundByID(randomDelegationID)

	// Type
	outputSet.requireDelegationFound()
	outputSet.requireAccountNotFound()
	outputSet.requireBasicNotFound()
	outputSet.requireNFTNotFound()
	outputSet.requireFoundryNotFound()
	outputSet.requireAnchorNotFound()

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
