package indexer

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	"github.com/iotaledger/inx-indexer/pkg/database"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

var (
	ErrNotFound = errors.New("output not found for given filter")

	tables = []interface{}{
		&Status{},
		&basicOutput{},
		&nft{},
		&foundry{},
		&alias{},
	}
)

type Indexer struct {
	*logger.WrappedLogger
	db *gorm.DB
}

func NewIndexer(dbPath string, log *logger.Logger) (*Indexer, error) {

	db, err := database.DatabaseWithDefaultSettings(dbPath, true, log)
	if err != nil {
		return nil, err
	}

	// Create the tables and indexes if needed
	if err := db.AutoMigrate(tables...); err != nil {
		return nil, err
	}

	return &Indexer{
		WrappedLogger: logger.NewWrappedLogger(log),
		db:            db,
	}, nil
}

func processSpent(spent *inx.LedgerSpent, tx *gorm.DB) error {
	iotaOutput, err := spent.GetOutput().UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return err
	}

	outputID := spent.GetOutput().GetOutputId().Unwrap()
	switch iotaOutput.(type) {
	case *iotago.BasicOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&basicOutput{}).Error
	case *iotago.AliasOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&alias{}).Error
	case *iotago.NFTOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&nft{}).Error
	case *iotago.FoundryOutput:
		return tx.Where("output_id = ?", outputID[:]).Delete(&foundry{}).Error
	}
	return nil
}

func processOutput(output *inx.LedgerOutput, tx *gorm.DB) error {
	unwrapped, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return err
	}

	outputID := output.GetOutputId().Unwrap()
	switch iotaOutput := unwrapped.(type) {
	case *iotago.BasicOutput:
		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		basic := &basicOutput{
			OutputID:         make(outputIDBytes, iotago.OutputIDLength),
			NativeTokenCount: len(iotaOutput.NativeTokens),
			CreatedAt:        unixTime(output.GetMilestoneTimestampBooked()),
		}
		copy(basic.OutputID, outputID[:])

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			basic.Sender, err = addressBytesForAddress(senderBlock.Address)
			if err != nil {
				return err
			}
		}

		if tagBlock := features.TagFeature(); tagBlock != nil {
			basic.Tag = make([]byte, len(tagBlock.Tag))
			copy(basic.Tag, tagBlock.Tag)
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			basic.Address, err = addressBytesForAddress(addressUnlock.Address)
			if err != nil {
				return err
			}
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			basic.StorageDepositReturn = &storageDepositReturn.Amount
			basic.StorageDepositReturnAddress, err = addressBytesForAddress(storageDepositReturn.ReturnAddress)
			if err != nil {
				return err
			}
		}

		if timelock := conditions.Timelock(); timelock != nil {
			time := unixTime(timelock.UnixTime)
			basic.TimelockTime = &time
		}

		if expiration := conditions.Expiration(); expiration != nil {
			time := unixTime(expiration.UnixTime)
			basic.ExpirationTime = &time
			basic.ExpirationReturnAddress, err = addressBytesForAddress(expiration.ReturnAddress)
			if err != nil {
				return err
			}
		}

		if err := tx.Create(basic).Error; err != nil {
			return err
		}

	case *iotago.AliasOutput:
		aliasID := iotaOutput.AliasID
		if aliasID.Empty() {
			// Use implicit AliasID
			aliasID = iotago.AliasIDFromOutputID(outputID)
		}

		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		alias := &alias{
			AliasID:          make(aliasIDBytes, iotago.AliasIDLength),
			OutputID:         make(outputIDBytes, iotago.OutputIDLength),
			NativeTokenCount: len(iotaOutput.NativeTokens),
			CreatedAt:        unixTime(output.GetMilestoneTimestampBooked()),
		}
		copy(alias.AliasID, aliasID[:])
		copy(alias.OutputID, outputID[:])

		if issuerBlock := features.IssuerFeature(); issuerBlock != nil {
			alias.Issuer, err = addressBytesForAddress(issuerBlock.Address)
			if err != nil {
				return err
			}
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			alias.Sender, err = addressBytesForAddress(senderBlock.Address)
			if err != nil {
				return err
			}
		}

		if stateController := conditions.StateControllerAddress(); stateController != nil {
			alias.StateController, err = addressBytesForAddress(stateController.Address)
			if err != nil {
				return err
			}
		}

		if governor := conditions.GovernorAddress(); governor != nil {
			alias.Governor, err = addressBytesForAddress(governor.Address)
			if err != nil {
				return err
			}
		}

		if err := tx.Create(alias).Error; err != nil {
			return err
		}

	case *iotago.NFTOutput:
		features := iotaOutput.FeatureSet()
		conditions := iotaOutput.UnlockConditionSet()

		nftID := iotaOutput.NFTID
		if nftID.Empty() {
			// Use implicit NFTID
			nftAddr := iotago.NFTAddressFromOutputID(outputID)
			nftID = nftAddr.NFTID()
		}

		nft := &nft{
			NFTID:            make(nftIDBytes, iotago.NFTIDLength),
			OutputID:         make(outputIDBytes, iotago.OutputIDLength),
			NativeTokenCount: len(iotaOutput.NativeTokens),
			CreatedAt:        unixTime(output.GetMilestoneTimestampBooked()),
		}
		copy(nft.NFTID, nftID[:])
		copy(nft.OutputID, outputID[:])

		if issuerBlock := features.IssuerFeature(); issuerBlock != nil {
			nft.Issuer, err = addressBytesForAddress(issuerBlock.Address)
			if err != nil {
				return err
			}
		}

		if senderBlock := features.SenderFeature(); senderBlock != nil {
			nft.Sender, err = addressBytesForAddress(senderBlock.Address)
			if err != nil {
				return err
			}
		}

		if tagBlock := features.TagFeature(); tagBlock != nil {
			nft.Tag = make([]byte, len(tagBlock.Tag))
			copy(nft.Tag, tagBlock.Tag)
		}

		if addressUnlock := conditions.Address(); addressUnlock != nil {
			nft.Address, err = addressBytesForAddress(addressUnlock.Address)
			if err != nil {
				return err
			}
		}

		if storageDepositReturn := conditions.StorageDepositReturn(); storageDepositReturn != nil {
			nft.StorageDepositReturn = &storageDepositReturn.Amount
			nft.StorageDepositReturnAddress, err = addressBytesForAddress(storageDepositReturn.ReturnAddress)
			if err != nil {
				return err
			}
		}

		if timelock := conditions.Timelock(); timelock != nil {
			time := unixTime(timelock.UnixTime)
			nft.TimelockTime = &time
		}

		if expiration := conditions.Expiration(); expiration != nil {
			time := unixTime(expiration.UnixTime)
			nft.ExpirationTime = &time
			nft.ExpirationReturnAddress, err = addressBytesForAddress(expiration.ReturnAddress)
			if err != nil {
				return err
			}
		}

		if err := tx.Create(nft).Error; err != nil {
			return err
		}

	case *iotago.FoundryOutput:
		conditions := iotaOutput.UnlockConditionSet()

		foundryID, err := iotaOutput.ID()
		if err != nil {
			return err
		}

		foundry := &foundry{
			FoundryID:        foundryID[:],
			OutputID:         make(outputIDBytes, iotago.OutputIDLength),
			NativeTokenCount: len(iotaOutput.NativeTokens),
			CreatedAt:        unixTime(output.GetMilestoneTimestampBooked()),
		}
		copy(foundry.OutputID, outputID[:])

		if aliasUnlock := conditions.ImmutableAlias(); aliasUnlock != nil {
			foundry.AliasAddress, err = addressBytesForAddress(aliasUnlock.Address)
			if err != nil {
				return err
			}
		}

		if err := tx.Create(foundry).Error; err != nil {
			return err
		}

	default:
		panic("Unknown output type")
	}

	return nil
}

func (i *Indexer) UpdatedLedger(update *inx.LedgerUpdate) error {

	tx := i.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	spentOutputs := make(map[string]struct{})
	for _, spent := range update.GetConsumed() {
		outputID := spent.GetOutput().GetOutputId().GetId()
		spentOutputs[string(outputID)] = struct{}{}
		if err := processSpent(spent, tx); err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, output := range update.GetCreated() {
		if _, wasSpentInSameMilestone := spentOutputs[string(output.GetOutputId().GetId())]; wasSpentInSameMilestone {
			// We only care about the end-result of the confirmation, so outputs that were already spent in the same milestone can be ignored
			continue
		}
		if err := processOutput(output, tx); err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Model(&Status{}).Where("id = ?", 1).Update("ledger_index", update.GetMilestoneIndex())

	return tx.Commit().Error
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
	for _, table := range tables {
		if err := i.db.Migrator().DropTable(table); err != nil {
			return err
		}
	}
	// Re-create tables
	return i.db.AutoMigrate(tables...)
}

func (i *Indexer) CloseDatabase() error {
	sqlDB, err := i.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
