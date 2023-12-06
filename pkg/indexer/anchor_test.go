package indexer_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

func TestIndexer_AnchorOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomAnchorID := iotago_tpkg.RandAnchorAddress().AnchorID()

	senderAddress := iotago_tpkg.RandEd25519Address()
	issuerAddress := iotago_tpkg.RandEd25519Address()
	stateControllerAddress := iotago_tpkg.RandEd25519Address()
	governorAddress := iotago_tpkg.RandEd25519Address()

	output := &iotago.AnchorOutput{
		Amount:     iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.ZeroCostTestAPI.ProtocolParameters().TokenSupply()))),
		Mana:       iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		StateIndex: 0,
		UnlockConditions: iotago.AnchorOutputUnlockConditions{
			&iotago.StateControllerAddressUnlockCondition{
				Address: stateControllerAddress,
			},
			&iotago.GovernorAddressUnlockCondition{
				Address: governorAddress,
			},
		},
		Features: iotago.AnchorOutputFeatures{
			&iotago.SenderFeature{
				Address: senderAddress,
			},
		},
		ImmutableFeatures: iotago.AnchorOutputImmFeatures{
			&iotago.IssuerFeature{
				Address: issuerAddress,
			},
		},
	}

	outputID := iotago_tpkg.RandOutputID(0)
	anchorAddress := iotago.AnchorAddressFromOutputID(outputID)

	outputSet := ts.AddOutputOnCommitment(output, outputID)
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireAnchorFoundByID(anchorAddress.AnchorID())
	outputSet.requireAnchorNotFoundByID(randomAnchorID)

	// Type
	outputSet.requireAnchorFound()
	outputSet.requireBasicNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireAccountNotFound()
	outputSet.requireNFTNotFound()
	outputSet.requireFoundryNotFound()

	// Creation Slot
	outputSet.requireAnchorFound(indexer.AnchorCreatedAfter(0))
	outputSet.requireAnchorNotFound(indexer.AnchorCreatedAfter(1))

	outputSet.requireAnchorNotFound(indexer.AnchorCreatedBefore(0))
	outputSet.requireAnchorNotFound(indexer.AnchorCreatedBefore(1))
	outputSet.requireAnchorFound(indexer.AnchorCreatedBefore(2))

	// State Controller
	outputSet.requireAnchorFound(indexer.AnchorStateController(stateControllerAddress))
	outputSet.requireAnchorNotFound(indexer.AnchorStateController(governorAddress))

	// Governor
	outputSet.requireAnchorFound(indexer.AnchorGovernor(governorAddress))
	outputSet.requireAnchorNotFound(indexer.AnchorGovernor(stateControllerAddress))

	// Sender
	outputSet.requireAnchorFound(indexer.AnchorSender(senderAddress))
	outputSet.requireAnchorNotFound(indexer.AnchorSender(randomAddress))

	// Issuer
	outputSet.requireAnchorFound(indexer.AnchorIssuer(issuerAddress))
	outputSet.requireAnchorNotFound(indexer.AnchorIssuer(randomAddress))

	// Unlockable by the following addresses
	for _, addr := range []iotago.Address{stateControllerAddress, governorAddress} {
		outputSet.requireAnchorFound(indexer.AnchorUnlockableByAddress(addr))
	}

	// Not unlockable by the following addresses
	for _, addr := range []iotago.Address{senderAddress, issuerAddress, anchorAddress} {
		outputSet.requireAnchorNotFound(indexer.AnchorUnlockableByAddress(addr))
	}
}
