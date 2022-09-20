package indexer

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormLogger "gorm.io/gorm/logger"

	"github.com/iotaledger/hive.go/core/logger"
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

type importWorker[T any] struct {
	*logger.WrappedLogger

	db   *gorm.DB
	name string
	wg   sync.WaitGroup

	queue chan T
}

func newImportWorker[T any](db *gorm.DB, name string, log *logger.Logger) *importWorker[T] {
	w := &importWorker[T]{
		WrappedLogger: logger.NewWrappedLogger(log),
		db:            db,
		name:          name,
		queue:         make(chan T, 10*batchSize),
	}
	w.Run()
	return w
}

func (w *importWorker[T]) closeAndWait() {
	close(w.queue)
	w.wg.Wait()
}

func (w *importWorker[T]) enqueue(item T) {
	w.queue <- item
}

func (w *importWorker[T]) insertBatch(batch []T) error {
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

func (w *importWorker[T]) Run() {
	for n := 0; n < workerCount; n++ {
		workerName := fmt.Sprintf("%s-%d", w.name, n)
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()

			w.LogInfof("[%s] started", workerName)

			batch := make([]T, 0, batchSize)
			var count int

			ts := time.Now()
			for item := range w.queue {
				batch = append(batch, item)
				count++
				if count%batchSize == 0 {
					if err := w.insertBatch(batch); err != nil {
						w.LogErrorf("[%s] error: %s", workerName, err.Error())
						return
					}
					batch = make([]T, 0, batchSize)
				}
				if count%100_000 == 0 {
					w.LogInfof("[%s] Inserted: %d, took %s", workerName, count, time.Since(ts).Truncate(time.Millisecond))
					ts = time.Now()
				}
			}
			if len(batch) > 0 {
				ts := time.Now()
				// Insert last remaining
				if err := w.insertBatch(batch); err != nil {
					w.LogErrorf("[%s] error: %s", workerName, err.Error())
					return
				}
				w.LogInfof("[%s] Inserted remaining: %d, took %s", workerName, len(batch), time.Since(ts).Truncate(time.Millisecond))
			}
			w.LogInfof("[%s] ended", workerName)
		}()
	}
}

type ImportTransaction struct {
	db *gorm.DB

	basic   *importWorker[*basicOutput]
	nft     *importWorker[*nft]
	alias   *importWorker[*alias]
	foundry *importWorker[*foundry]
}

func newImportTransaction(db *gorm.DB, log *logger.Logger) *ImportTransaction {

	// use a session without logger and hooks to reduce the amount of work that needs to be done by gorm.
	dbSession := db.Session(&gorm.Session{
		SkipHooks:              true,
		SkipDefaultTransaction: true,
		Logger:                 gormLogger.Discard,
	})

	t := &ImportTransaction{
		db:      dbSession,
		basic:   newImportWorker[*basicOutput](dbSession, "basic", log),
		nft:     newImportWorker[*nft](dbSession, "nft", log),
		alias:   newImportWorker[*alias](dbSession, "alias", log),
		foundry: newImportWorker[*foundry](dbSession, "foundry", log),
	}

	return t
}

func (i *ImportTransaction) AddOutput(output *inx.LedgerOutput) error {

	entry, err := entryForOutput(output)
	if err != nil {
		return err
	}

	switch e := entry.(type) {
	case *basicOutput:
		i.basic.enqueue(e)
	case *nft:
		i.nft.enqueue(e)
	case *alias:
		i.alias.enqueue(e)
	case *foundry:
		i.foundry.enqueue(e)
	}

	return nil
}

func (i *ImportTransaction) Finalize(ledgerIndex uint32, protoParams *iotago.ProtocolParameters) error {

	// drain all workers
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
