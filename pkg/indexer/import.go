package indexer

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormLogger "gorm.io/gorm/logger"
	"sync"
	"time"

	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	batchSize   = 1_000
	workerCount = 4
)

func (i *Indexer) ImportTransaction() *ImportTransaction {
	return newImportTransaction(i.db, i.Logger())
}

type Worker[T any] struct {
	db   *gorm.DB
	name string
	wg   sync.WaitGroup

	queue chan T
}

func NewWorker[T any](db *gorm.DB, name string) *Worker[T] {
	w := &Worker[T]{
		db:    db,
		name:  name,
		queue: make(chan T, 10*batchSize),
	}
	w.Run()
	return w
}

func (w *Worker[T]) closeAndWait() {
	close(w.queue)
	w.wg.Wait()
}

func (w *Worker[T]) Enqueue(item T) {
	w.queue <- item
}

func (w *Worker[T]) insertBatch(batch []T) error {
	tx := w.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Create(batch).Error; err != nil {
		return err
	}
	return tx.Commit().Error
}

func (w *Worker[T]) Run() {
	for n := 0; n < workerCount; n++ {
		workerName := fmt.Sprintf("%s-%d", w.name, n)
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			batch := make([]T, 0, batchSize)
			var count int

			ts := time.Now()
			for item := range w.queue {
				batch = append(batch, item)
				count++
				if count%batchSize == 0 {
					if err := w.insertBatch(batch); err != nil {
						fmt.Printf("error: %s\n", err.Error())
						return
					}
					batch = make([]T, 0, batchSize)
				}
				if count%100_000 == 0 {
					fmt.Printf("[%s] Inserted: %d, took %s\n", workerName, count, time.Since(ts).Truncate(time.Millisecond))
					ts = time.Now()
				}
			}
			if len(batch) > 0 {
				ts := time.Now()
				// Insert last remaining
				if err := w.insertBatch(batch); err != nil {
					fmt.Printf("error: %s\n", err.Error())
					return
				}
				fmt.Printf("[%s] Inserted remaining: %d, took %s\n", workerName, len(batch), time.Since(ts).Truncate(time.Millisecond))
			}
			fmt.Printf("[%s] ended\n", workerName)
		}()
	}
}

type ImportTransaction struct {
	db *gorm.DB

	basic   *Worker[*basicOutput]
	nft     *Worker[*nft]
	alias   *Worker[*alias]
	foundry *Worker[*foundry]
}

func newImportTransaction(db *gorm.DB) *ImportTransaction {

	dbSession := db.Session(&gorm.Session{
		PrepareStmt:            true,
		SkipHooks:              true,
		SkipDefaultTransaction: true,
		Logger:                 gormLogger.Discard,
	})

	t := &ImportTransaction{
		db:      dbSession,
		basic:   NewWorker[*basicOutput](dbSession, "basic"),
		nft:     NewWorker[*nft](dbSession, "nft"),
		alias:   NewWorker[*alias](dbSession, "alias"),
		foundry: NewWorker[*foundry](dbSession, "foundry"),
	}

	return t
}

func (i *ImportTransaction) AddOutput(output *inx.LedgerOutput) error {

	op, err := opForOutput(output)
	if err != nil {
		return err
	}

	switch o := op.(type) {
	case *basicOutput:
		i.basic.Enqueue(o)
	case *nft:
		i.nft.Enqueue(o)
	case *alias:
		i.alias.Enqueue(o)
	case *foundry:
		i.foundry.Enqueue(o)
	}

	return nil
}

func (i *ImportTransaction) Finalize(ledgerIndex uint32, protoParams *iotago.ProtocolParameters) error {

	i.basic.closeAndWait()
	i.nft.closeAndWait()
	i.alias.closeAndWait()
	i.foundry.closeAndWait()

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
