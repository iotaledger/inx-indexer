package indexer

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotaledger/hive.go/db"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/log"
	"github.com/iotaledger/hive.go/sql"
	iotago "github.com/iotaledger/iota.go/v4"
)

var (
	AllowedEngines = []db.Engine{db.EngineSQLite, db.EnginePostgreSQL}
)

var (
	ErrStatusNotFound      = ierrors.New("status not found")
	ErrLedgerUpdateSkipped = ierrors.New("ledger update skipped")

	dbTables = append([]interface{}{
		&Status{},
		&multiaddress{},
	}, outputTables...)

	outputTables = []interface{}{
		&basic{},
		&nft{},
		&foundry{},
		&account{},
		&anchor{},
		&delegation{},
	}
)

type Indexer struct {
	log.Logger
	db     *gorm.DB
	engine db.Engine

	lastCommittedSlot      iotago.SlotIndex
	lastCommittedSlotMutex sync.RWMutex
}

func NewIndexer(dbParams sql.DatabaseParameters, logger log.Logger) (*Indexer, error) {
	db, engine, err := sql.New(logger, dbParams, true, AllowedEngines)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		Logger: logger,
		db:     db,
		engine: engine,
	}, nil
}

func addressesInOutput(output iotago.Output) []iotago.Address {
	var foundAddresses []iotago.Address

	// Check for addresses in features
	features := output.FeatureSet()
	if senderBlock := features.SenderFeature(); senderBlock != nil {
		foundAddresses = append(foundAddresses, senderBlock.Address)
	}

	// Check for addresses in unlock conditions
	conditions := output.UnlockConditionSet()
	if addressUnlock := conditions.Address(); addressUnlock != nil {
		foundAddresses = append(foundAddresses, addressUnlock.Address)
	}
	if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
		foundAddresses = append(foundAddresses, storageDepositReturn.ReturnAddress)
	}
	if expiration := conditions.Expiration(); expiration != nil {
		foundAddresses = append(foundAddresses, expiration.ReturnAddress)
	}
	if accountUnlock := conditions.ImmutableAccount(); accountUnlock != nil {
		foundAddresses = append(foundAddresses, accountUnlock.Address)
	}
	if stateController := conditions.StateControllerAddress(); stateController != nil {
		foundAddresses = append(foundAddresses, stateController.Address)
	}
	if governor := conditions.GovernorAddress(); governor != nil {
		foundAddresses = append(foundAddresses, governor.Address)
	}

	// Check for addresses in immutable features
	if chainOutput, ok := output.(iotago.ChainOutputImmutable); ok {
		immutableFeatures := chainOutput.ImmutableFeatureSet()

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			foundAddresses = append(foundAddresses, issuerBlock.Address)
		}
	}

	// Check for addresses in delegation output
	if delegationOutput, ok := output.(*iotago.DelegationOutput); ok {
		foundAddresses = append(foundAddresses, delegationOutput.ValidatorAddress)
	}

	return foundAddresses
}

func tableForOutput(output iotago.Output) interface{} {
	switch output.(type) {
	case *iotago.BasicOutput:
		return &basic{}
	case *iotago.AccountOutput:
		return &account{}
	case *iotago.AnchorOutput:
		return &anchor{}
	case *iotago.NFTOutput:
		return &nft{}
	case *iotago.FoundryOutput:
		return &foundry{}
	case *iotago.DelegationOutput:
		return &delegation{}
	default:
		panic("unexpected output type")
	}
}

func processSpent(output *LedgerOutput, committed bool, tx *gorm.DB) error {
	// Properly delete the outputs if they were committed
	if committed {
		if err := tx.Where("output_id = ?", output.OutputID[:]).Delete(tableForOutput(output.Output)).Error; err != nil {
			return err
		}

		// Delete committed MultiAddress deletions
		return deleteMultiAddressesFromAddresses(tx, addressesInOutput(output.Output))
	}

	// Mark them as deleted but leave them in for now
	if err := tx.Model(tableForOutput(output.Output)).Where("output_id = ?", output.OutputID[:]).Update("deleted_at_slot", output.SpentAt).Error; err != nil {
		return err
	}

	return nil
}

func removeUncommittedChangesUpUntilSlot(committedSlot iotago.SlotIndex, tx *gorm.DB) error {
	for _, table := range outputTables {
		// Remove the uncommitted insertions (this does not delete the outputs that were already marked to be deleted at a later point in time)
		if err := tx.Where("created_at_slot <= ? AND committed = false AND deleted_at_slot <= ?", committedSlot, committedSlot).Delete(table).Error; err != nil {
			return err
		}

		// Revert all uncommitted deletions
		if err := tx.Model(table).Where("deleted_at_slot > 0 AND deleted_at_slot <= ?", committedSlot).Update("deleted_at_slot", 0).Error; err != nil {
			return err
		}
	}

	return nil
}

func (i *Indexer) RemoveUncommittedChanges() error {
	return i.db.Transaction(func(tx *gorm.DB) error {
		// Remove all MultiAddresses with only pending references
		if err := deleteMultiAddressesWithOnlyUncommittedReferences(tx); err != nil {
			return err
		}

		// Remove all uncommitted outputs
		return removeUncommittedChangesUpUntilSlot(iotago.MaxSlotIndex, tx)
	})
}

func processOutput(output *LedgerOutput, committed bool, tx *gorm.DB) error {
	entry, err := entryForOutput(output.OutputID, output.Output, output.BookedAt, committed)
	if err != nil {
		return err
	}

	var createQuery *gorm.DB
	if committed {
		// This output might still be in the database from a previous uncommitted state but it was already marked as deleted in a later slot, so we will only update the committed flag
		createQuery = tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "output_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"committed": true,
			})}).Create(entry)
	} else {
		createQuery = tx.Create(entry)
	}

	if err := createQuery.Error; err != nil {
		return err
	}

	return insertMultiAddressesFromAddresses(tx, addressesInOutput(output.Output), committed)
}

func entryForOutput(outputID iotago.OutputID, output iotago.Output, slotBooked iotago.SlotIndex, committed bool) (interface{}, error) {
	var entry interface{}
	switch iotaOutput := output.(type) {
	case *iotago.BasicOutput:
		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		basic := &basic{
			Amount:        iotaOutput.Amount,
			OutputID:      make([]byte, iotago.OutputIDLength),
			CreatedAtSlot: slotBooked,
			Committed:     committed,
		}
		copy(basic.OutputID, outputID[:])

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			basic.Sender = senderBlock.Address.ID()
		}

		if tagBlock := features.Tag(); tagBlock != nil {
			basic.Tag = make([]byte, len(tagBlock.Tag))
			copy(basic.Tag, tagBlock.Tag)
		}

		if nativeToken := features.NativeToken(); nativeToken != nil {
			basic.NativeToken = nativeToken.ID[:]
			amount := hexutil.EncodeBig(nativeToken.Amount)
			basic.NativeTokenAmount = &amount
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			basic.Address = addressUnlock.Address.ID()
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			basic.StorageDepositReturn = &storageDepositReturn.Amount
			basic.StorageDepositReturnAddress = storageDepositReturn.ReturnAddress.ID()
		}

		if timelock := conditions.Timelock(); timelock != nil {
			basic.TimelockSlot = &timelock.Slot
		}

		if expiration := conditions.Expiration(); expiration != nil {
			basic.ExpirationSlot = &expiration.Slot
			basic.ExpirationReturnAddress = expiration.ReturnAddress.ID()
		}

		entry = basic

	case *iotago.AccountOutput:
		accountID := iotaOutput.AccountID
		if accountID.Empty() {
			// Use implicit AccountID
			accountID = iotago.AccountIDFromOutputID(outputID)
		}

		features := iotaOutput.FeatureSet()
		immutableFeatures := iotaOutput.ImmutableFeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		acc := &account{
			Amount:        iotaOutput.Amount,
			AccountID:     make([]byte, iotago.AccountIDLength),
			OutputID:      make([]byte, iotago.OutputIDLength),
			CreatedAtSlot: slotBooked,
			Committed:     committed,
		}
		copy(acc.AccountID, accountID[:])
		copy(acc.OutputID, outputID[:])

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			acc.Issuer = issuerBlock.Address.ID()
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			acc.Sender = senderBlock.Address.ID()
		}

		if address := conditions.Address(); address != nil {
			acc.Address = address.Address.ID()
		}

		entry = acc

	case *iotago.AnchorOutput:
		anchorID := iotaOutput.AnchorID
		if anchorID.Empty() {
			// Use implicit AnchorID
			anchorID = iotago.AnchorIDFromOutputID(outputID)
		}

		features := iotaOutput.FeatureSet()
		immutableFeatures := iotaOutput.ImmutableFeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		anc := &anchor{
			Amount:        iotaOutput.Amount,
			AnchorID:      make([]byte, iotago.AnchorIDLength),
			OutputID:      make([]byte, iotago.OutputIDLength),
			CreatedAtSlot: slotBooked,
			Committed:     committed,
		}
		copy(anc.AnchorID, anchorID[:])
		copy(anc.OutputID, outputID[:])

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			anc.Issuer = issuerBlock.Address.ID()
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			anc.Sender = senderBlock.Address.ID()
		}

		if stateController := conditions.StateControllerAddress(); stateController != nil {
			anc.StateController = stateController.Address.ID()
		}

		if governor := conditions.GovernorAddress(); governor != nil {
			anc.Governor = governor.Address.ID()
		}

		entry = anc

	case *iotago.NFTOutput:
		features := iotaOutput.FeatureSet()
		immutableFeatures := iotaOutput.ImmutableFeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		nftID := iotaOutput.NFTID
		if nftID.Empty() {
			// Use implicit NFTID
			nftAddr := iotago.NFTAddressFromOutputID(outputID)
			nftID = nftAddr.NFTID()
		}

		nft := &nft{
			Amount:        iotaOutput.Amount,
			NFTID:         make([]byte, iotago.NFTIDLength),
			OutputID:      make([]byte, iotago.OutputIDLength),
			CreatedAtSlot: slotBooked,
			Committed:     committed,
		}
		copy(nft.NFTID, nftID[:])
		copy(nft.OutputID, outputID[:])

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			nft.Issuer = issuerBlock.Address.ID()
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			nft.Sender = senderBlock.Address.ID()
		}

		if tagBlock := features.Tag(); tagBlock != nil {
			nft.Tag = make([]byte, len(tagBlock.Tag))
			copy(nft.Tag, tagBlock.Tag)
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			nft.Address = addressUnlock.Address.ID()
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			amount := uint64(storageDepositReturn.Amount)
			nft.StorageDepositReturn = &amount
			nft.StorageDepositReturnAddress = storageDepositReturn.ReturnAddress.ID()
		}

		if timelock := conditions.Timelock(); timelock != nil {
			nft.TimelockSlot = &timelock.Slot
		}

		if expiration := conditions.Expiration(); expiration != nil {
			nft.ExpirationSlot = &expiration.Slot
			nft.ExpirationReturnAddress = expiration.ReturnAddress.ID()
		}

		entry = nft

	case *iotago.FoundryOutput:
		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		foundryID, err := iotaOutput.FoundryID()
		if err != nil {
			return nil, err
		}

		foundry := &foundry{
			Amount:        iotaOutput.Amount,
			FoundryID:     foundryID[:],
			OutputID:      make([]byte, iotago.OutputIDLength),
			CreatedAtSlot: slotBooked,
			Committed:     committed,
		}
		copy(foundry.OutputID, outputID[:])

		if nativeToken := features.NativeToken(); nativeToken != nil {
			amount := hexutil.EncodeBig(nativeToken.Amount)
			foundry.NativeTokenAmount = &amount
		}

		if accountUnlock := conditions.ImmutableAccount(); accountUnlock != nil {
			foundry.AccountAddress = accountUnlock.Address.ID()
		}

		entry = foundry

	case *iotago.DelegationOutput:
		conditions := iotaOutput.UnlockConditionSet()

		delegationID := iotaOutput.DelegationID
		if delegationID.Empty() {
			// Use implicit DelegationID
			delegationID = iotago.DelegationIDFromOutputID(outputID)
		}

		delegation := &delegation{
			Amount:        iotaOutput.Amount,
			DelegationID:  make([]byte, iotago.DelegationIDLength),
			OutputID:      make([]byte, iotago.OutputIDLength),
			CreatedAtSlot: slotBooked,
			Committed:     committed,
		}
		copy(delegation.DelegationID, delegationID[:])
		copy(delegation.OutputID, outputID[:])

		delegation.Validator = iotaOutput.ValidatorAddress.ID()

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			delegation.Address = addressUnlock.Address.ID()
		}

		entry = delegation

	default:
		return nil, ierrors.New("unknown output type")
	}

	return entry, nil
}

func (i *Indexer) IsInitialized() bool {
	return i.db.Migrator().HasTable(&Status{})
}

func (i *Indexer) CreateTables() error {
	return i.db.Migrator().CreateTable(dbTables...)
}

func (i *Indexer) DropIndexes() error {
	m := i.db.Migrator()
	for _, table := range dbTables {
		stmt := &gorm.Statement{DB: i.db}
		if err := stmt.ParseWithSpecialTableName(table, ""); err != nil {
			return err
		}

		for name := range stmt.Schema.ParseIndexes() {
			if m.HasIndex(table, name) {
				if err := m.DropIndex(table, name); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (i *Indexer) AutoMigrate() error {
	// Create the tables and indexes if needed
	return i.db.AutoMigrate(dbTables...)
}

func (i *Indexer) AcceptLedgerUpdate(update *LedgerUpdate) error {
	return i.db.Transaction(func(tx *gorm.DB) error {
		i.lastCommittedSlotMutex.RLock()
		lastCommitted := i.lastCommittedSlot
		i.lastCommittedSlotMutex.RUnlock()

		if update.Slot <= lastCommitted {
			return ierrors.Wrapf(ErrLedgerUpdateSkipped, "accepted slot %d is not greater than last committed slot %d", update.Slot, lastCommitted)
		}

		spentOutputs := make(map[iotago.OutputID]struct{})
		for _, output := range update.Consumed {
			spentOutputs[output.OutputID] = struct{}{}
			if err := processSpent(output, false, tx); err != nil {
				return err
			}
		}

		for _, output := range update.Created {
			if _, wasSpentInSameSlot := spentOutputs[output.OutputID]; wasSpentInSameSlot {
				// We only care about the end-result of the confirmation, so outputs that were already spent in the same update can be ignored
				continue
			}
			if err := processOutput(output, false, tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (i *Indexer) CommitLedgerUpdate(update *LedgerUpdate) error {
	if err := i.db.Transaction(func(tx *gorm.DB) error {
		// Cleanup uncommitted changes for this update
		if err := removeUncommittedChangesUpUntilSlot(update.Slot, tx); err != nil {
			return err
		}

		spentOutputs := make(map[iotago.OutputID]struct{})
		for _, output := range update.Consumed {
			spentOutputs[output.OutputID] = struct{}{}
			if err := processSpent(output, true, tx); err != nil {
				return err
			}
		}

		for _, output := range update.Created {
			if _, wasSpentInSameSlot := spentOutputs[output.OutputID]; wasSpentInSameSlot {
				// We only care about the end-result of the confirmation, so outputs that were already spent in the same update can be ignored
				continue
			}
			if err := processOutput(output, true, tx); err != nil {
				return err
			}
		}

		tx.Model(&Status{}).Where("id = ?", 1).Update("committed_slot", update.Slot)

		return nil
	}); err != nil {
		return err
	}

	i.lastCommittedSlotMutex.Lock()
	defer i.lastCommittedSlotMutex.Unlock()

	if i.lastCommittedSlot < update.Slot {
		i.lastCommittedSlot = update.Slot
	}

	return nil
}

func (i *Indexer) Status() (*Status, error) {
	status := &Status{}
	if err := i.db.Take(&status).Error; err != nil {
		if ierrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStatusNotFound
		}

		return nil, err
	}

	i.lastCommittedSlotMutex.RLock()
	val := i.lastCommittedSlot
	i.lastCommittedSlotMutex.RUnlock()

	// Only get write lock if the new committed slot is greater than the last committed slot
	if status.CommittedSlot > val {
		i.lastCommittedSlotMutex.Lock()
		defer i.lastCommittedSlotMutex.Unlock()

		if status.CommittedSlot > i.lastCommittedSlot {
			i.lastCommittedSlot = status.CommittedSlot
		}
	}

	return status, nil
}

func (i *Indexer) Clear() error {
	i.lastCommittedSlotMutex.Lock()
	defer i.lastCommittedSlotMutex.Unlock()

	i.lastCommittedSlot = 0

	// Drop all tables
	if err := i.db.Migrator().DropTable(dbTables...); err != nil {
		return err
	}
	// Re-create tables
	return i.CreateTables()
}

func (i *Indexer) CloseDatabase() error {
	sqlDB, err := i.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
