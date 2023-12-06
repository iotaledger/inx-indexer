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

func TestIndexer_AccountOutput(t *testing.T) {
	ts := newTestSuite(t)

	randomAddress := iotago_tpkg.RandEd25519Address()
	randomAccountID := iotago_tpkg.RandAccountAddress().AccountID()

	senderAddress := iotago_tpkg.RandEd25519Address()
	issuerAddress := iotago_tpkg.RandEd25519Address()
	address := iotago_tpkg.RandEd25519Address()

	output := &iotago.AccountOutput{
		Amount:         iotago.BaseToken(iotago_tpkg.RandUint64(uint64(iotago_tpkg.ZeroCostTestAPI.ProtocolParameters().TokenSupply()))),
		Mana:           iotago.Mana(iotago_tpkg.RandUint64(math.MaxUint64)),
		FoundryCounter: 0,
		UnlockConditions: iotago.AccountOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
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

	outputID := iotago_tpkg.RandOutputID(0)
	accountAddress := iotago.AccountAddressFromOutputID(outputID)

	outputSet := ts.AddOutputOnCommitment(output, outputID)
	require.Equal(t, iotago.SlotIndex(1), ts.CurrentSlot())

	// By ID
	outputSet.requireAccountFoundByID(accountAddress.AccountID())
	outputSet.requireAccountNotFoundByID(randomAccountID)

	// Type
	outputSet.requireAccountFound()
	outputSet.requireBasicNotFound()
	outputSet.requireDelegationNotFound()
	outputSet.requireNFTNotFound()
	outputSet.requireFoundryNotFound()
	outputSet.requireAnchorNotFound()

	// Creation Slot
	outputSet.requireAccountFound(indexer.AccountCreatedAfter(0))
	outputSet.requireAccountNotFound(indexer.AccountCreatedAfter(1))

	outputSet.requireAccountNotFound(indexer.AccountCreatedBefore(0))
	outputSet.requireAccountNotFound(indexer.AccountCreatedBefore(1))
	outputSet.requireAccountFound(indexer.AccountCreatedBefore(2))

	// Address
	outputSet.requireAccountFound(indexer.AccountUnlockAddress(address))
	outputSet.requireAccountNotFound(indexer.AccountUnlockAddress(randomAddress))

	// Sender
	outputSet.requireAccountFound(indexer.AccountSender(senderAddress))
	outputSet.requireAccountNotFound(indexer.AccountSender(randomAddress))

	// Issuer
	outputSet.requireAccountFound(indexer.AccountIssuer(issuerAddress))
	outputSet.requireAccountNotFound(indexer.AccountIssuer(randomAddress))
}
