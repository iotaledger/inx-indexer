package indexer

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/inx-indexer/pkg/database"
	iotago "github.com/iotaledger/iota.go/v4"
)

var (
	ErrStatusNotFound = errors.New("status not found")

	dbTables = append([]interface{}{
		&Status{},
		&multiaddress{},
	}, outputTables...)

	outputTables = []interface{}{
		&basic{},
		&nft{},
		&foundry{},
		&account{},
		&delegation{},
	}
)

type Indexer struct {
	*logger.WrappedLogger
	db     *gorm.DB
	engine database.Engine
}

func NewIndexer(dbParams database.Params, log *logger.Logger) (*Indexer, error) {
	db, engine, err := database.NewWithDefaultSettings(dbParams, true, log)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		WrappedLogger: logger.NewWrappedLogger(log),
		db:            db,
		engine:        engine,
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

func processSpent(output *LedgerOutput, committed bool, tx *gorm.DB) error {
	// Properly delete the outputs if they were committed
	if committed {
		switch output.Output.(type) {
		case *iotago.BasicOutput:
			if err := tx.Where("output_id = ?", output.OutputID[:]).Delete(&basic{}).Error; err != nil {
				return err
			}
		case *iotago.AccountOutput:
			if err := tx.Where("output_id = ?", output.OutputID[:]).Delete(&account{}).Error; err != nil {
				return err
			}
		case *iotago.NFTOutput:
			if err := tx.Where("output_id = ?", output.OutputID[:]).Delete(&nft{}).Error; err != nil {
				return err
			}
		case *iotago.FoundryOutput:
			if err := tx.Where("output_id = ?", output.OutputID[:]).Delete(&foundry{}).Error; err != nil {
				return err
			}
		case *iotago.DelegationOutput:
			if err := tx.Where("output_id = ?", output.OutputID[:]).Delete(&delegation{}).Error; err != nil {
				return err
			}
		}

		// Delete committed MultiAddress deletions
		return deleteMultiAddressesFromAddresses(tx, addressesInOutput(output.Output))
	}

	// Mark them as deleted but leave them in for now
	switch output.Output.(type) {
	case *iotago.BasicOutput:
		if err := tx.Model(&basic{}).Where("output_id = ?", output.OutputID[:]).Update("deleted_at_slot", output.SpentAt).Error; err != nil {
			return err
		}
	case *iotago.AccountOutput:
		if err := tx.Model(&account{}).Where("output_id = ?", output.OutputID[:]).Update("deleted_at_slot", output.SpentAt).Error; err != nil {
			return err
		}
	case *iotago.NFTOutput:
		if err := tx.Model(&nft{}).Where("output_id = ?", output.OutputID[:]).Update("deleted_at_slot", output.SpentAt).Error; err != nil {
			return err
		}
	case *iotago.FoundryOutput:
		if err := tx.Model(&foundry{}).Where("output_id = ?", output.OutputID[:]).Update("deleted_at_slot", output.SpentAt).Error; err != nil {
			return err
		}
	case *iotago.DelegationOutput:
		if err := tx.Model(&delegation{}).Where("output_id = ?", output.OutputID[:]).Update("deleted_at_slot", output.SpentAt).Error; err != nil {
			return err
		}
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

		if stateController := conditions.StateControllerAddress(); stateController != nil {
			acc.StateController = stateController.Address.ID()
		}

		if governor := conditions.GovernorAddress(); governor != nil {
			acc.Governor = governor.Address.ID()
		}

		entry = acc

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
		return nil, errors.New("unknown output type")
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
	return i.db.Transaction(func(tx *gorm.DB) error {
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
	})
}

func (i *Indexer) Status() (*Status, error) {
	status := &Status{}
	if err := i.db.Take(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStatusNotFound
		}

		return nil, err
	}

	return status, nil
}

func (i *Indexer) Clear() error {
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
