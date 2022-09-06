package indexer

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	"github.com/iotaledger/inx-app/nodebridge"
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

type BulkUpdaterManager struct {
	updaterChannel chan *inx.UnspentOutput
	updatedWorkers []*BulkUpdater
	dbParams       database.Params
	workers        int
	log            *logger.Logger
}

var manager *BulkUpdaterManager

type BulkUpdater struct {
	basicOutputs []*basicOutput
	nfts         []*nft
	foundries    []*foundry
	aliases      []*alias
	bulkSize     int
	counter      int
	chanDone     chan bool
	err          error
	tx           *gorm.DB
	db           *gorm.DB
	manager      *BulkUpdaterManager
	worker       int
}

func (b *BulkUpdater) reset() {
	b.basicOutputs = make([]*basicOutput, 0, b.bulkSize)
	b.nfts = make([]*nft, 0, b.bulkSize)
	b.foundries = make([]*foundry, 0, b.bulkSize)
	b.aliases = make([]*alias, 0, b.bulkSize)
	b.counter = 0
}

func (b *BulkUpdater) init() error {
	var err error
	b.manager.log.Infof("[%d] creating db connection", b.worker)

	b.db, err = database.NewWithDefaultSettings(manager.dbParams, true, manager.log)
	return err
}

func NewBulkUpdater(worker int, manager *BulkUpdaterManager, bulkSize int) *BulkUpdater {
	b := &BulkUpdater{bulkSize: bulkSize}
	b.chanDone = make(chan bool)
	b.manager = manager
	b.worker = worker
	b.reset()
	return b
}

func NewBulkUpdaterManager(dbParams database.Params, workers int, bulkSize int, log *logger.Logger) (*BulkUpdaterManager, error) {
	manager = &BulkUpdaterManager{
		updaterChannel: make(chan *inx.UnspentOutput, 50000),
		updatedWorkers: make([]*BulkUpdater, workers),
		dbParams:       dbParams,
		workers:        workers,
		log:            log,
	}

	for i := 0; i < workers; i++ {
		manager.updatedWorkers[i] = NewBulkUpdater(i, manager, bulkSize)
		if err := manager.updatedWorkers[i].init(); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func (m *BulkUpdaterManager) Done() error {
	for i := 0; i < m.workers; i++ {
		if err := m.updatedWorkers[i].Done(); err != nil {
			return err
		}
	}
	return nil
}

func (m *BulkUpdaterManager) Start() error {
	for i := 0; i < m.workers; i++ {
		m.updatedWorkers[i].Start()
	}
	return nil
}

func (b *BulkUpdater) Start() {
	go func() {
		b.manager.log.Infof("[%d] starting worker ...", b.worker)
		defer func() {
			if r := recover(); r != nil {
				b.manager.log.Debugf("[%d] tx rollback", b.worker)
				b.tx.Rollback()
			}
		}()

		for {
			unspentOutput, more := <-manager.updaterChannel
			if more {
				if err := b.addOutput(unspentOutput.GetOutput()); err != nil {
					b.err = err
					return
				}
			} else {
				break
			}
		}
		// store the rest
		if err := b.store(); err != nil {
			b.err = err
			return
		}
		b.err = nil
		b.chanDone <- true
		b.manager.log.Infof("[%d] worker stopped", b.worker)
	}()
}

func (b *BulkUpdater) Done() error {
	close(manager.updaterChannel)
	<-b.chanDone

	sqlDB, err := b.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (b *BulkUpdater) GetError() error {
	return b.err
}

func (m *BulkUpdaterManager) Enqueue(output *inx.UnspentOutput) {
	manager.updaterChannel <- output
}

func GetBulkUpdaterManager() *BulkUpdaterManager {
	return manager
}

func (b *BulkUpdater) store() error {
	b.tx = b.db.Begin()
	//err := b.tx.Exec("set transaction isolation level read uncommitted").Error
	//if err != nil {
	//	return err
	//}
	b.manager.log.Debugf("[%d] tx begin", b.worker)

	if err := b.tx.Create(b.basicOutputs).Error; err != nil {
		return err
	}
	if err := b.tx.Create(b.nfts).Error; err != nil {
		return err
	}
	if err := b.tx.Create(b.foundries).Error; err != nil {
		return err
	}
	if err := b.tx.Create(b.aliases).Error; err != nil {
		return err
	}

	if err := b.tx.Commit().Error; err != nil {
		return err
	}
	b.manager.log.Debugf("[%d] tx committed", b.worker)

	// in case of error, transaction would be retried
	b.reset()
	return nil
}

func (b *BulkUpdater) addBasicOutput(output *basicOutput) {
	b.basicOutputs = append(b.basicOutputs, output)
}

func (b *BulkUpdater) addAlias(output *alias) {
	b.aliases = append(b.aliases, output)
}

func (b *BulkUpdater) addNFT(output *nft) {
	b.nfts = append(b.nfts, output)
}

func (b *BulkUpdater) addFoundry(output *foundry) {
	b.foundries = append(b.foundries, output)
}

func (b *BulkUpdater) addOutput(output *inx.LedgerOutput) error {
	if err := processOutput(output, b); err != nil {
		return err
	}

	b.counter += 1
	if b.counter < b.bulkSize {
		return nil
	}

	return b.store()
}

func NewIndexer(dbParams database.Params, log *logger.Logger) (*Indexer, error) {

	db, err := database.NewWithDefaultSettings(dbParams, true, log)
	if err != nil {
		return nil, err
	}

	// Create the tables and indexes if needed
	if err := db.AutoMigrate(tables...); err != nil {
		return nil, err
	}

	manager, err = NewBulkUpdaterManager(dbParams, 8, 2500, log)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		WrappedLogger: logger.NewWrappedLogger(log),
		db:            db,
	}, nil
}

func (i *Indexer) DropIndices() {
	/*
		i.db.Migrator().DropIndex(&basicOutput{}, "basic_outputs_sender_tag")
		i.db.Migrator().DropIndex(&basicOutput{}, "basic_outputs_address")

		i.db.Migrator().DropConstraint(&foundry{}, "foundries_output_id_key")
		i.db.Migrator().DropIndex(&foundry{}, "foundries_alias_address")

		i.db.Migrator().DropConstraint(&alias{}, "aliases_output_id_key")
		i.db.Migrator().DropIndex(&alias{}, "alias_state_controller")
		i.db.Migrator().DropIndex(&alias{}, "alias_governor")
		i.db.Migrator().DropIndex(&alias{}, "alias_issuer")
		i.db.Migrator().DropIndex(&alias{}, "alias_sender")

		i.db.Migrator().DropConstraint(&nft{}, "nfts_output_id_key")
		i.db.Migrator().DropIndex(&nft{}, "nfts_issuer")
		i.db.Migrator().DropIndex(&nft{}, "nfts_sender_tag")
		i.db.Migrator().DropIndex(&nft{}, "nfts_issuer")
		i.db.Migrator().DropIndex(&nft{}, "nfts_address")
	*/
}

func (i *Indexer) CreateIndices() {
	//	i.db.AutoMigrate()
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

func processOutput(output *inx.LedgerOutput, bulkUpdater *BulkUpdater) error {
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
		bulkUpdater.addBasicOutput(basic)
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
		bulkUpdater.addAlias(alias)
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
		bulkUpdater.addNFT(nft)
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
		bulkUpdater.addFoundry(foundry)
	default:
		panic("Unknown output type")
	}

	return nil
}

func (i *Indexer) UpdatedLedger(update *nodebridge.LedgerUpdate) error {
	/*
		tx := i.db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		if err := tx.Error; err != nil {
			return err
		}

		bulkUpdater := NewBulkUpdater(tx, 10000)

		spentOutputs := make(map[string]struct{})
		for _, spent := range update.Consumed {
			outputID := spent.GetOutput().GetOutputId().GetId()
			spentOutputs[string(outputID)] = struct{}{}
			if err := processSpent(spent, tx); err != nil {
				tx.Rollback()

				return err
			}
		}

		for _, output := range update.Created {
			if _, wasSpentInSameMilestone := spentOutputs[string(output.GetOutputId().GetId())]; wasSpentInSameMilestone {
				// We only care about the end-result of the confirmation, so outputs that were already spent in the same milestone can be ignored
				continue
			}

			if err := bulkUpdater.addOutput(output); err != nil {
				tx.Rollback()

				return err
			}
		}

		// process the remaining outputs not already inserted
		bulkUpdater.store()

		tx.Model(&Status{}).Where("id = ?", 1).Update("ledger_index", update.MilestoneIndex)

		return tx.Commit().Error
	*/
	return nil
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
