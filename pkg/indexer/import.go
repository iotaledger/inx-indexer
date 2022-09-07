package indexer

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormLogger "gorm.io/gorm/logger"

	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const batchSize = 1_000

func (i *Indexer) ImportTransaction() *ImportTransaction {
	return newImportTransaction(i.db)
}

type ImportTransaction struct {
	db *gorm.DB

	basic   []*basicOutput
	nft     []*nft
	alias   []*alias
	foundry []*foundry
}

func newImportTransaction(db *gorm.DB) *ImportTransaction {
	t := &ImportTransaction{
		db: db.Session(&gorm.Session{
			PrepareStmt:            true,
			SkipHooks:              true,
			SkipDefaultTransaction: true,
			Logger:                 gormLogger.Discard,
		}),
	}
	t.resetPendingBatch()
	return t
}

func (i *ImportTransaction) AddOutput(output *inx.LedgerOutput) error {

	op, err := opForOutput(output)
	if err != nil {
		return err
	}

	switch o := op.(type) {
	case *basicOutput:
		i.basic = append(i.basic, o)
		if len(i.basic) == batchSize {
			i.db.Create(i.basic)
			i.basic = make([]*basicOutput, 0, batchSize)
		}
	case *nft:
		i.nft = append(i.nft, o)
		if len(i.nft) == batchSize {
			i.db.Create(i.nft)
			i.nft = make([]*nft, 0, batchSize)
		}
	case *alias:
		i.alias = append(i.alias, o)
		if len(i.nft) == batchSize {
			i.db.Create(i.alias)
			i.alias = make([]*alias, 0, batchSize)
		}
	case *foundry:
		i.foundry = append(i.foundry, o)
		if len(i.foundry) == batchSize {
			i.db.Create(i.foundry)
			i.foundry = make([]*foundry, 0, batchSize)
		}
	}

	return nil
}

func (i *ImportTransaction) resetPendingBatch() {
	i.basic = make([]*basicOutput, 0, batchSize)
	i.nft = make([]*nft, 0, batchSize)
	i.alias = make([]*alias, 0, batchSize)
	i.foundry = make([]*foundry, 0, batchSize)
}

func (i *ImportTransaction) insertPendingBatch() error {
	i.db.Create(i.basic)
	i.db.Create(i.nft)
	i.db.Create(i.alias)
	i.db.Create(i.foundry)
	i.resetPendingBatch()
	return i.db.Error
}

func (i *ImportTransaction) Finalize(ledgerIndex uint32, protoParams *iotago.ProtocolParameters) error {
	// Insert last batch if necessary
	if err := i.insertPendingBatch(); err != nil {
		return err
	}

	// Update the indexer status
	status := &Status{
		ID:              1,
		LedgerIndex:     ledgerIndex,
		ProtocolVersion: protoParams.Version,
		NetworkName:     protoParams.NetworkName,
	}
	i.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&status)

	return i.db.Error
}
