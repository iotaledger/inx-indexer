package indexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

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

	ts.AddOutputOnCommitment(output1, output1ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.AddOutputOnCommitment(output2, output2ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.DeleteOutputOnCommitment(output1ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.DeleteOutputOnCommitment(output2ID)

	require.False(t, ts.MultiAddressExists(multiaddress))

	output3 := nftOutputWithAddressAndSender(multiaddress)
	output3ID := iotago_tpkg.RandOutputID(2)
	ts.AddOutputOnCommitment(output3, output3ID)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.DeleteOutputOnCommitment(output3ID)

	require.False(t, ts.MultiAddressExists(multiaddress))
}

func TestIndexer_MultiAddress_OnAcceptance(t *testing.T) {
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

	require.False(t, ts.MultiAddressExists(multiaddress))

	ts.AddOutputOnAcceptance(output1, output1ID, 1)

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.AddOutputOnCommitment(output1, output1ID) // Slot 1

	require.True(t, ts.MultiAddressExists(multiaddress))

	ts.AddOutputOnAcceptance(output2, output2ID, 2)

	require.True(t, ts.MultiAddressExists(multiaddress))

	// Delete output1 on commitment, which should also delete the uncommitted output2
	ts.DeleteOutputOnCommitment(output1ID) // Slot 2

	// The multiaddress should still exist, because output2 held an uncommitted reference to it
	require.True(t, ts.MultiAddressExists(multiaddress))

	// Simulate indexer restart
	require.NoError(t, ts.Indexer.RemoveUncommittedChanges()) // Reset to 2

	// The multiaddress should not exist anymore
	require.False(t, ts.MultiAddressExists(multiaddress))
}
