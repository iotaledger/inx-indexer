package indexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	iotago "github.com/iotaledger/iota.go/v4"
	iotago_tpkg "github.com/iotaledger/iota.go/v4/tpkg"
)

var (
	tests = []*outputTest{
		{
			name:     "basic output",
			output:   basicOutputWithAddress(iotago_tpkg.RandEd25519Address()),
			outputID: iotago_tpkg.RandOutputID(0),
		},
		{
			name:     "nft output",
			output:   nftOutputWithAddressAndSender(iotago_tpkg.RandEd25519Address()),
			outputID: iotago_tpkg.RandOutputID(0),
		},
		{
			name:     "delegation output",
			output:   delegationOutputWithAddress(iotago_tpkg.RandEd25519Address()),
			outputID: iotago_tpkg.RandOutputID(0),
		},
		{
			name:     "account output",
			output:   accountOutputWithAddress(iotago_tpkg.RandEd25519Address()),
			outputID: iotago_tpkg.RandOutputID(0),
		},
		{
			name:     "anchor output",
			output:   anchorOutputWithAddress(iotago_tpkg.RandEd25519Address()),
			outputID: iotago_tpkg.RandOutputID(0),
		},
		{
			name:     "foundry output",
			output:   foundryOutputWithAddress(iotago_tpkg.RandAccountAddress()),
			outputID: iotago_tpkg.RandOutputID(0),
		},
	}
)

type outputTest struct {
	name     string
	output   iotago.Output
	outputID iotago.OutputID
}

// TestIndexer_CommitAdd_AcceptAdd tests the following scenario:
// 1. Add output on commitment
// 2. Add output on acceptance (because it was delayed by some reason)
func TestIndexer_CommitAdd_AcceptAdd(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.commitAddThenAccept)
	}
}

func (o *outputTest) commitAddThenAccept(t *testing.T) {
	ts := newTestSuite(t)

	// Commit Add
	ts.AddOutputOnCommitment(o.output, o.outputID) // Slot 1

	// Accepted outputs are found
	ts.requireFound(o.outputID)

	// Accept Add (should be skipped because it was already committed)
	ts.AddOutputOnAcceptance(o.output, o.outputID, 1, true)

	// Still needs to be found
	ts.requireFound(o.outputID)
}

// TestIndexer_AcceptAdd_CommitAdd_AcceptDelete_CommitDelete tests the following scenario:
// 1. Add output on acceptance
// 2. Add output on commitment
// 3. Delete output on acceptance
// 4. Delete output on commitment
func TestIndexer_AcceptAdd_CommitAdd_AcceptDelete_CommitDelete(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.acceptAddThenCommitAddThenAcceptDeleteThenCommitDelete)
	}
}

func (o *outputTest) acceptAddThenCommitAddThenAcceptDeleteThenCommitDelete(t *testing.T) {
	ts := newTestSuite(t)

	// Accept Add
	ts.AddOutputOnAcceptance(o.output, o.outputID, 1)

	// Accepted outputs are found
	ts.requireFound(o.outputID)

	// Commit Add
	ts.AddOutputOnCommitment(o.output, o.outputID) // Slot 1

	// Still needs to be found
	ts.requireFound(o.outputID)

	// Accept Delete
	ts.DeleteOutputOnAcceptance(o.outputID, 2)

	// Output should not be found anymore (but still in db)
	ts.requireNotFound(o.outputID)

	// Commit Delete
	ts.DeleteOutputOnCommitment(o.outputID) // Slot 2

	// Output should not be found anymore (deleted from db)
	ts.requireNotFound(o.outputID)
}

// TestIndexer_AcceptAdd_NeverCommitAdd tests the following scenario:
// 1. Add output on acceptance
// 2. Commit without adding output
func TestIndexer_AcceptAdd_NeverCommitAdd(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.acceptAddThenCommitEmpty)
	}
}

func (o *outputTest) acceptAddThenCommitEmpty(t *testing.T) {
	ts := newTestSuite(t)

	// Accept Add
	ts.AddOutputOnAcceptance(o.output, o.outputID, 1)

	// Accepted outputs are found
	ts.requireFound(o.outputID)

	// Commit (so that all uncommitted outputs are deleted)
	ts.CommitEmptyLedgerUpdate() // Slot 1

	// Output should not be found anymore (deleted from db)
	ts.requireNotFound(o.outputID)
}

// TestIndexer_AcceptAdd_NeverCommitAdd tests the following scenario:
// 1. Add output on commitment
// 2. Delete output on acceptance
// 2. Commit without deleting output
func TestIndexer_CommitAdd_AcceptDelete_NeverCommitDelete(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.commitAddThenAcceptDeleteThenCommitEmpty)
	}
}

func (o *outputTest) commitAddThenAcceptDeleteThenCommitEmpty(t *testing.T) {
	ts := newTestSuite(t)

	// Commit Add
	ts.AddOutputOnCommitment(o.output, o.outputID) // Slot 1

	// Committed outputs are found
	ts.requireFound(o.outputID)

	// Delete on acceptance
	ts.DeleteOutputOnAcceptance(o.outputID, 2)

	// Output should not be found (but still in db)
	ts.requireNotFound(o.outputID)

	// Commit (so that all uncommitted deletes are reverted)
	ts.CommitEmptyLedgerUpdate() // Slot 2

	// Output should be found again because the deletion was never committed
	ts.requireFound(o.outputID)
}

func TestIndexer_AcceptAdd_RestartIndexer(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.acceptAddThenRestartIndexer)
	}
}

func (o *outputTest) acceptAddThenRestartIndexer(t *testing.T) {
	ts := newTestSuite(t)

	// Commit something so that we are not at zero
	ts.CommitEmptyLedgerUpdate() // Slot 1

	// Accept Add
	ts.AddOutputOnAcceptance(o.output, o.outputID, 2)

	// Outputs are found
	ts.requireFound(o.outputID)

	require.NoError(t, ts.Indexer.RemoveUncommittedChanges()) // Reset to 1

	// Output should not be found again because we removed all uncommitted changes
	ts.requireNotFound(o.outputID)
}

func TestIndexer_AcceptDelete_RestartIndexer(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.acceptDeleteThenRestartIndexer)
	}
}

func (o *outputTest) acceptDeleteThenRestartIndexer(t *testing.T) {
	ts := newTestSuite(t)

	// Commit Add
	ts.AddOutputOnCommitment(o.output, o.outputID) // Slot 1

	// Outputs are found
	ts.requireFound(o.outputID)

	// Accept Delete
	ts.DeleteOutputOnAcceptance(o.outputID, 2)

	// Output should not be found anymore (but still in db)
	ts.requireNotFound(o.outputID)

	require.NoError(t, ts.Indexer.RemoveUncommittedChanges()) // Reset to 1

	// Output should be found again because we reverted the uncommitted delete
	ts.requireFound(o.outputID)
}

// TestIndexer_AcceptAdd_AcceptDelete_CommitAdd tests the following scenario:
// 1. Add output on acceptance
// 2. Delete output on acceptance
// 2. Add initial output on commitment -> should not be found because it was deleted on 2
func TestIndexer_AcceptAdd_AcceptDelete_CommitAdd(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, tt.acceptAddThenAcceptDeleteThenCommitAdd)
	}
}

func (o *outputTest) acceptAddThenAcceptDeleteThenCommitAdd(t *testing.T) {
	ts := newTestSuite(t)

	// Accept Add
	ts.AddOutputOnAcceptance(o.output, o.outputID, 1)

	// Outputs are found
	ts.requireFound(o.outputID)

	// Accept Delete
	ts.DeleteOutputOnAcceptance(o.outputID, 2)

	// Output should not be found anymore (but still in db)
	ts.requireNotFound(o.outputID)

	// Commit Add
	ts.AddOutputOnCommitment(o.output, o.outputID) // Slot 1

	// Output should still not be found because it was deleted on slot 2
	ts.requireNotFound(o.outputID)

	// Commit deletion
	ts.DeleteOutputOnCommitment(o.outputID) // Slot 2

	// Output not found
	ts.requireNotFound(o.outputID)
}

func basicOutputWithAddress(address iotago.Address) iotago.Output {
	return &iotago.BasicOutput{
		Amount: 100000,
		Mana:   0,
		UnlockConditions: iotago.BasicOutputUnlockConditions{
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
		UnlockConditions: iotago.NFTOutputUnlockConditions{
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

func delegationOutputWithAddress(address iotago.Address) iotago.Output {
	return &iotago.DelegationOutput{
		Amount:           100000,
		ValidatorAddress: iotago_tpkg.RandAccountAddress(),
		UnlockConditions: iotago.DelegationOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
	}
}

func accountOutputWithAddress(address iotago.Address) iotago.Output {
	return &iotago.AccountOutput{
		Amount: 100000,
		UnlockConditions: iotago.AccountOutputUnlockConditions{
			&iotago.AddressUnlockCondition{
				Address: address,
			},
		},
	}
}

func anchorOutputWithAddress(address iotago.Address) iotago.Output {
	return &iotago.AnchorOutput{
		Amount: 100000,
		UnlockConditions: iotago.AnchorOutputUnlockConditions{
			&iotago.StateControllerAddressUnlockCondition{
				Address: address,
			},
			&iotago.GovernorAddressUnlockCondition{
				Address: address,
			},
		},
	}
}

func foundryOutputWithAddress(accountAddress *iotago.AccountAddress) iotago.Output {
	return &iotago.FoundryOutput{
		Amount: 100000,
		TokenScheme: &iotago.SimpleTokenScheme{
			MintedTokens:  iotago_tpkg.RandUint256(),
			MeltedTokens:  iotago_tpkg.RandUint256(),
			MaximumSupply: iotago_tpkg.RandUint256(),
		},
		UnlockConditions: iotago.FoundryOutputUnlockConditions{
			&iotago.ImmutableAccountUnlockCondition{
				Address: accountAddress,
			},
		},
	}
}
