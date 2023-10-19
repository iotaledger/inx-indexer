package indexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

func basicOutputWithAddress(address iotago.Address) iotago.Output {
	return &iotago.BasicOutput{
		Amount: 100000,
		Mana:   0,
		Conditions: iotago.BasicOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
		Features: nil,
	}
}

func nftOutputWithAddressAndSender(address iotago.Address) iotago.Output {
	return &iotago.NFTOutput{
		Amount: 100000,
		Mana:   0,
		Conditions: iotago.NFTOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
		Features: iotago.NFTOutputFeatures{
			&iotago.SenderFeature{
				Address: address,
			},
		},
	}
}

func TestIndexer_MultiAddress(t *testing.T) {
	ts := newTestSuite(t)

	multiaddress := &iotago.MultiAddress{
		Addresses: iotago.AddressesWithWeight{
			&iotago.AddressWithWeight{
				Address: iotago_tpkg.RandEd25519Address(),
				Weight:  1,
			},
			&iotago.AddressWithWeight{
				Address: iotago_tpkg.RandEd25519Address(),
				Weight:  1,
			},
		},
		Threshold: 2,
	}

	output1 := basicOutputWithAddress(multiaddress)
	output1ID := iotago_tpkg.RandOutputID(0)
	output2 := basicOutputWithAddress(multiaddress)
	output2ID := iotago_tpkg.RandOutputID(1)

	ts.AddOutput(output1, output1ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.AddOutput(output2, output2ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.DeleteOutput(output1ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.DeleteOutput(output2ID)

	require.False(t, ts.MultiAddressExists(multiaddress))

	output3 := nftOutputWithAddressAndSender(multiaddress)
	output3ID := iotago_tpkg.RandOutputID(2)
	ts.AddOutput(output3, output3ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.DeleteOutput(output3ID)

	require.False(t, ts.MultiAddressExists(multiaddress))
}
