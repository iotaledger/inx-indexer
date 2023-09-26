package indexer

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-indexer/pkg/database"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
)

var (
	ErrNotFound = errors.New("output not found for given filter")

	dbTables = []interface{}{
		&Status{},
		&basicOutput{},
		&nft{},
		&foundry{},
		&account{},
		&multiaddress{},
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

func processSpent(spent *inx.LedgerSpent, api iotago.API, tx *gorm.DB) error {
	iotaOutput, err := spent.GetOutput().UnwrapOutput(api)
	if err != nil {
		return err
	}

	outputID := spent.GetOutput().GetOutputId().Unwrap()
	switch iotaOutput.(type) {
	case *iotago.BasicOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&basicOutput{}).Error
	case *iotago.AccountOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&account{}).Error
	case *iotago.NFTOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&nft{}).Error
	case *iotago.FoundryOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&foundry{}).Error
	case *iotago.DelegationOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&delegation{}).Error
	}

	return nil
}

func processOutput(output *inx.LedgerOutput, api iotago.API, tx *gorm.DB) error {
	unwrapped, err := output.UnwrapOutput(api)
	if err != nil {
		return err
	}

	outputID := output.GetOutputId().Unwrap()

	entry, multiAddresses, err := entryForOutput(outputID, unwrapped, iotago.SlotIndex(output.GetSlotBooked()))
	if err != nil {
		return err
	}

	if err := tx.Create(entry).Error; err != nil {
		return err
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(multiAddresses).Error
}

func entryForOutput(outputID iotago.OutputID, output iotago.Output, slotBooked iotago.SlotIndex) (interface{}, []*multiaddress, error) {
	var entry interface{}
	var foundAddresses []iotago.Address
	switch iotaOutput := output.(type) {
	case *iotago.BasicOutput:
		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		basic := &basicOutput{
			OutputID:         make([]byte, iotago.OutputIDLength),
			NativeTokenCount: uint32(len(iotaOutput.NativeTokens)),
			CreatedAt:        slotBooked,
		}
		copy(basic.OutputID, outputID[:])

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			basic.Sender = senderBlock.Address.ID()
			foundAddresses = append(foundAddresses, senderBlock.Address)
		}

		if tagBlock := features.Tag(); tagBlock != nil {
			basic.Tag = make([]byte, len(tagBlock.Tag))
			copy(basic.Tag, tagBlock.Tag)
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			basic.Address = addressUnlock.Address.ID()
			foundAddresses = append(foundAddresses, addressUnlock.Address)
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			basic.StorageDepositReturn = &storageDepositReturn.Amount
			basic.StorageDepositReturnAddress = storageDepositReturn.ReturnAddress.ID()
			foundAddresses = append(foundAddresses, storageDepositReturn.ReturnAddress)
		}

		if timelock := conditions.Timelock(); timelock != nil {
			basic.TimelockSlot = &timelock.SlotIndex
		}

		if expiration := conditions.Expiration(); expiration != nil {
			basic.ExpirationSlot = &expiration.SlotIndex
			basic.ExpirationReturnAddress = expiration.ReturnAddress.ID()
			foundAddresses = append(foundAddresses, expiration.ReturnAddress)
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
			AccountID:        make([]byte, iotago.AccountIDLength),
			OutputID:         make([]byte, iotago.OutputIDLength),
			NativeTokenCount: uint32(len(iotaOutput.NativeTokens)),
			CreatedAt:        slotBooked,
		}
		copy(acc.AccountID, accountID[:])
		copy(acc.OutputID, outputID[:])

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			acc.Issuer = issuerBlock.Address.ID()
			foundAddresses = append(foundAddresses, issuerBlock.Address)
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			acc.Sender = senderBlock.Address.ID()
			foundAddresses = append(foundAddresses, senderBlock.Address)
		}

		if stateController := conditions.StateControllerAddress(); stateController != nil {
			acc.StateController = stateController.Address.ID()
			foundAddresses = append(foundAddresses, stateController.Address)
		}

		if governor := conditions.GovernorAddress(); governor != nil {
			acc.Governor = governor.Address.ID()
			foundAddresses = append(foundAddresses, governor.Address)
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
			NFTID:            make([]byte, iotago.NFTIDLength),
			OutputID:         make([]byte, iotago.OutputIDLength),
			NativeTokenCount: uint32(len(iotaOutput.NativeTokens)),
			CreatedAt:        slotBooked,
		}
		copy(nft.NFTID, nftID[:])
		copy(nft.OutputID, outputID[:])

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			nft.Issuer = issuerBlock.Address.ID()
			foundAddresses = append(foundAddresses, issuerBlock.Address)
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			nft.Sender = senderBlock.Address.ID()
			foundAddresses = append(foundAddresses, senderBlock.Address)
		}

		if tagBlock := features.Tag(); tagBlock != nil {
			nft.Tag = make([]byte, len(tagBlock.Tag))
			copy(nft.Tag, tagBlock.Tag)
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			nft.Address = addressUnlock.Address.ID()
			foundAddresses = append(foundAddresses, addressUnlock.Address)
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			amount := uint64(storageDepositReturn.Amount)
			nft.StorageDepositReturn = &amount
			nft.StorageDepositReturnAddress = storageDepositReturn.ReturnAddress.ID()
			foundAddresses = append(foundAddresses, storageDepositReturn.ReturnAddress)
		}

		if timelock := conditions.Timelock(); timelock != nil {
			nft.TimelockTime = &timelock.SlotIndex
		}

		if expiration := conditions.Expiration(); expiration != nil {
			nft.ExpirationTime = &expiration.SlotIndex
			nft.ExpirationReturnAddress = expiration.ReturnAddress.ID()
			foundAddresses = append(foundAddresses, expiration.ReturnAddress)
		}

		entry = nft

	case *iotago.FoundryOutput:
		conditions := iotaOutput.UnlockConditionSet()

		foundryID, err := iotaOutput.ID()
		if err != nil {
			return nil, nil, err
		}

		foundry := &foundry{
			FoundryID:        foundryID[:],
			OutputID:         make([]byte, iotago.OutputIDLength),
			NativeTokenCount: uint32(len(iotaOutput.NativeTokens)),
			CreatedAt:        slotBooked,
		}
		copy(foundry.OutputID, outputID[:])

		if accountUnlock := conditions.ImmutableAccount(); accountUnlock != nil {
			foundry.AccountAddress = accountUnlock.Address.ID()
			foundAddresses = append(foundAddresses, accountUnlock.Address)
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
			DelegationID: make([]byte, iotago.DelegationIDLength),
			OutputID:     make([]byte, iotago.OutputIDLength),
			CreatedAt:    slotBooked,
		}
		copy(delegation.DelegationID, delegationID[:])
		copy(delegation.OutputID, outputID[:])

		delegation.Validator = iotaOutput.ValidatorAddress.ID()
		foundAddresses = append(foundAddresses, iotaOutput.ValidatorAddress)

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			delegation.Address = addressUnlock.Address.ID()
			foundAddresses = append(foundAddresses, addressUnlock.Address)
		}

		entry = delegationID

	default:
		return nil, nil, errors.New("unknown output type")
	}

	multiAddresses, err := multiAddressesForAddresses(foundAddresses...)
	if err != nil {
		return nil, nil, err
	}

	return entry, multiAddresses, nil
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

func (i *Indexer) UpdatedLedger(update *nodebridge.LedgerUpdate) error {
	return i.db.Transaction(func(tx *gorm.DB) error {
		spentOutputs := make(map[string]struct{})
		for _, spent := range update.Consumed {
			outputID := spent.GetOutput().GetOutputId().GetId()
			spentOutputs[string(outputID)] = struct{}{}
			if err := processSpent(spent, update.API, tx); err != nil {
				return err
			}
		}

		for _, output := range update.Created {
			if _, wasSpentInSameSlot := spentOutputs[string(output.GetOutputId().GetId())]; wasSpentInSameSlot {
				// We only care about the end-result of the confirmation, so outputs that were already spent in the same milestone can be ignored
				continue
			}
			if err := processOutput(output, update.API, tx); err != nil {
				return err
			}
		}

		tx.Model(&Status{}).Where("id = ?", 1).Update("ledger_index", update.SlotIndex)

		return nil
	})
}

func (i *Indexer) Status() (*Status, error) {
	status := &Status{}
	if err := i.db.Take(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
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
