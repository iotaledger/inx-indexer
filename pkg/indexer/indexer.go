package indexer

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"gorm.io/gorm"

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

	// Check for addresses in immutable features
	if chainOutput, ok := output.(iotago.ChainOutputImmutable); ok {
		immutableFeatures := chainOutput.ImmutableFeatureSet()

		if issuerBlock := immutableFeatures.Issuer(); issuerBlock != nil {
			foundAddresses = append(foundAddresses, issuerBlock.Address)
		}

		if stateController := conditions.StateControllerAddress(); stateController != nil {
			foundAddresses = append(foundAddresses, stateController.Address)
		}

		if governor := conditions.GovernorAddress(); governor != nil {
			foundAddresses = append(foundAddresses, governor.Address)
		}
	}

	// Check for addresses in delegation output
	if delegationOutput, ok := output.(*iotago.DelegationOutput); ok {
		foundAddresses = append(foundAddresses, delegationOutput.ValidatorAddress)
	}

	return foundAddresses
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

	return deleteMultiAddressesFromAddresses(tx, addressesInOutput(iotaOutput))
}

func processOutput(output *inx.LedgerOutput, api iotago.API, tx *gorm.DB) error {
	unwrapped, err := output.UnwrapOutput(api)
	if err != nil {
		return err
	}

	outputID := output.GetOutputId().Unwrap()

	entry, err := entryForOutput(outputID, unwrapped, iotago.SlotIndex(output.GetSlotBooked()))
	if err != nil {
		return err
	}

	if err := tx.Create(entry).Error; err != nil {
		return err
	}

	return insertMultiAddressesFromAddresses(tx, addressesInOutput(unwrapped))
}

func entryForOutput(outputID iotago.OutputID, output iotago.Output, slotBooked iotago.SlotIndex) (interface{}, error) {
	var entry interface{}
	switch iotaOutput := output.(type) {
	case *iotago.BasicOutput:
		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		basic := &basicOutput{
			Amount:    iotaOutput.Amount,
			OutputID:  make([]byte, iotago.OutputIDLength),
			CreatedAt: slotBooked,
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
			basic.NativeTokenAmount = hexutil.EncodeBig(nativeToken.Amount)
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			basic.Address = addressUnlock.Address.ID()
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			basic.StorageDepositReturn = &storageDepositReturn.Amount
			basic.StorageDepositReturnAddress = storageDepositReturn.ReturnAddress.ID()
		}

		if timelock := conditions.Timelock(); timelock != nil {
			basic.TimelockSlot = &timelock.SlotIndex
		}

		if expiration := conditions.Expiration(); expiration != nil {
			basic.ExpirationSlot = &expiration.SlotIndex
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
			Amount:    iotaOutput.Amount,
			AccountID: make([]byte, iotago.AccountIDLength),
			OutputID:  make([]byte, iotago.OutputIDLength),
			CreatedAt: slotBooked,
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
			Amount:    iotaOutput.Amount,
			NFTID:     make([]byte, iotago.NFTIDLength),
			OutputID:  make([]byte, iotago.OutputIDLength),
			CreatedAt: slotBooked,
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
			nft.TimelockTime = &timelock.SlotIndex
		}

		if expiration := conditions.Expiration(); expiration != nil {
			nft.ExpirationTime = &expiration.SlotIndex
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
			Amount:    iotaOutput.Amount,
			FoundryID: foundryID[:],
			OutputID:  make([]byte, iotago.OutputIDLength),
			CreatedAt: slotBooked,
		}
		copy(foundry.OutputID, outputID[:])

		if nativeToken := features.NativeToken(); nativeToken != nil {
			foundry.NativeTokenAmount = hexutil.EncodeBig(nativeToken.Amount)
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
			Amount:       iotaOutput.Amount,
			DelegationID: make([]byte, iotago.DelegationIDLength),
			OutputID:     make([]byte, iotago.OutputIDLength),
			CreatedAt:    slotBooked,
		}
		copy(delegation.DelegationID, delegationID[:])
		copy(delegation.OutputID, outputID[:])

		delegation.Validator = iotaOutput.ValidatorAddress.ID()

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			delegation.Address = addressUnlock.Address.ID()
		}

		entry = delegationID

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

		tx.Model(&Status{}).Where("id = ?", 1).Update("ledger_index", update.Slot)

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
