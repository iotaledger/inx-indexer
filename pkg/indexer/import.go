package indexer

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

func (i *Indexer) ImportTransaction() *ImportTransaction {
	return newImportTransaction(i.db)
}

type ImportTransaction struct {
	tx *gorm.DB
}

func newImportTransaction(db *gorm.DB) *ImportTransaction {
	return &ImportTransaction{
		tx: db.Begin(),
	}
}

func (i *ImportTransaction) AddOutput(output *inx.LedgerOutput) error {
	if err := processOutput(output, i.tx); err != nil {
		i.tx.Rollback()
		return err
	}
	return nil
}

func (i *ImportTransaction) Finalize(ledgerIndex uint32, protoParams *iotago.ProtocolParameters) error {
	// Update the indexer status
	status := &Status{
		ID:              1,
		LedgerIndex:     ledgerIndex,
		ProtocolVersion: protoParams.Version,
		NetworkName:     protoParams.NetworkName,
	}
	i.tx.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&status)

	return i.tx.Commit().Error
}

func (i *ImportTransaction) Cancel() error {
	return i.tx.Rollback().Error
}
