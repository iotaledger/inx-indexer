package indexer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormLogger "gorm.io/gorm/logger"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/log"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	batchSize          = 1_000
	perBatcherWorkers  = 2
	perImporterWorkers = 2
)

func typeOf[T any]() string {
	t := *new(T)
	return reflect.TypeOf(t).Elem().Name()
}

func typeIsRefCountable[T any]() bool {
	_, ok := interface{}(new(T)).(refCountable)
	return ok
}

type refCountable interface {
	primaryKeyColumn() string
	refCountDelta() int
}

type batcher[T any] struct {
	log.Logger

	name string
	wg   sync.WaitGroup

	input  chan T
	output chan []T
}

func newBatcher[T any](logger log.Logger) *batcher[T] {
	w := &batcher[T]{
		Logger: logger,
		name:   typeOf[T](),
		input:  make(chan T, 1_000*batchSize),
		output: make(chan []T, 1000),
	}

	return w
}

func (b *batcher[T]) closeAndWait() {
	close(b.input)
	b.wg.Wait()
	close(b.output)
}

func (b *batcher[T]) Run(ctx context.Context, workerCount int) {
	for n := range workerCount {
		workerName := fmt.Sprintf("batcher-%s-%d", b.name, n)
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()

			b.LogDebugf("[%s] started", workerName)
			defer b.LogDebugf("[%s] ended", workerName)

			batch := make([]T, 0, batchSize)
			var count int
			for item := range b.input {
				if ctx.Err() != nil {
					return
				}

				batch = append(batch, item)
				count++
				if count%batchSize == 0 {
					b.output <- batch
					batch = make([]T, 0, batchSize)
				}
			}
			if len(batch) > 0 {
				// Insert last remaining
				b.output <- batch
			}
		}()
	}
}

type inserter[T any] struct {
	log.Logger

	name string
	db   *gorm.DB
	wg   sync.WaitGroup
}

func newImporter[T any](db *gorm.DB, logger log.Logger) *inserter[T] {
	w := &inserter[T]{
		Logger: logger,
		name:   typeOf[T](),
		db:     db,
	}

	return w
}

func (i *inserter[T]) Run(ctx context.Context, workerCount int, input <-chan []T) {
	useRefCounts := typeIsRefCountable[T]()
	for n := range workerCount {
		workerName := fmt.Sprintf("inserter-%s-%d", i.name, n)
		i.wg.Add(1)
		go func() {
			defer i.wg.Done()

			i.LogDebugf("[%s] started", workerName)
			defer i.LogDebugf("[%s] ended", workerName)

			ts := time.Now()
			p := message.NewPrinter(language.English)

			var count int
			for batch := range input {
				if ctx.Err() != nil {
					return
				}

				if err := i.db.Transaction(func(tx *gorm.DB) error {
					if useRefCounts {
						for _, item := range batch {
							if itemWithRefCount, ok := interface{}(item).(refCountable); ok {
								if err := tx.Clauses(clause.OnConflict{
									Columns:   []clause.Column{{Name: itemWithRefCount.primaryKeyColumn()}},
									DoUpdates: clause.Assignments(map[string]interface{}{"ref_count": gorm.Expr("ref_count + ?", itemWithRefCount.refCountDelta())}),
								}).Create(item).Error; err != nil {
									return err
								}
							} else {
								return ierrors.Errorf("item %T does not implement RefCountable", item)
							}
						}

						return nil
					}

					return tx.Create(batch).Error
				}); err != nil {
					i.LogFatal(err.Error())
				}
				count += len(batch)
				if count > 0 && count%100_000 == 0 {
					i.LogInfo(p.Sprintf("[%s] insert worker=%d @ %.2f per second", workerName, count, float64(count)/float64(time.Since(ts)/time.Second)))
				}
			}
		}()
	}
}

func (i *inserter[T]) closeAndWait() {
	i.wg.Wait()
}

type processor[T fmt.Stringer] struct {
	batcher  *batcher[T]
	importer *inserter[T]
}

func newProcessor[T fmt.Stringer](ctx context.Context, db *gorm.DB, logger log.Logger) *processor[T] {
	p := &processor[T]{
		batcher:  newBatcher[T](logger),
		importer: newImporter[T](db, logger),
	}
	p.batcher.Run(ctx, perBatcherWorkers)
	p.importer.Run(ctx, perImporterWorkers, p.batcher.output)

	return p
}

func (p *processor[T]) enqueue(items ...T) {
	for _, item := range items {
		p.batcher.input <- item
	}
}

func (p *processor[T]) closeAndWait() {
	p.batcher.closeAndWait()
	p.importer.closeAndWait()
}

func (i *Indexer) ImportTransaction(ctx context.Context) *ImportTransaction {
	return newImportTransaction(ctx, i.db, i.Logger)
}

type ImportTransaction struct {
	log.Logger

	db *gorm.DB

	basic        *processor[*basic]
	nft          *processor[*nft]
	account      *processor[*account]
	anchor       *processor[*anchor]
	foundry      *processor[*foundry]
	delegation   *processor[*delegation]
	multiAddress *processor[*multiaddress]
}

func newImportTransaction(ctx context.Context, db *gorm.DB, logger log.Logger) *ImportTransaction {
	// use a session without logger and hooks to reduce the amount of work that needs to be done by gorm.
	dbSession := db.Session(&gorm.Session{
		SkipHooks:              true,
		SkipDefaultTransaction: true,
		Logger:                 gormLogger.Discard,
	})

	t := &ImportTransaction{
		Logger:       logger,
		db:           dbSession,
		basic:        newProcessor[*basic](ctx, dbSession, logger),
		nft:          newProcessor[*nft](ctx, dbSession, logger),
		account:      newProcessor[*account](ctx, dbSession, logger),
		anchor:       newProcessor[*anchor](ctx, dbSession, logger),
		foundry:      newProcessor[*foundry](ctx, dbSession, logger),
		delegation:   newProcessor[*delegation](ctx, dbSession, logger),
		multiAddress: newProcessor[*multiaddress](ctx, dbSession, logger),
	}

	return t
}

func (i *ImportTransaction) AddOutput(outputID iotago.OutputID, output iotago.Output, slotBooked iotago.SlotIndex) error {
	entry, err := entryForOutput(outputID, output, slotBooked, true)
	if err != nil {
		return err
	}

	switch e := entry.(type) {
	case *basic:
		i.basic.enqueue(e)
	case *nft:
		i.nft.enqueue(e)
	case *account:
		i.account.enqueue(e)
	case *anchor:
		i.anchor.enqueue(e)
	case *foundry:
		i.foundry.enqueue(e)
	case *delegation:
		i.delegation.enqueue(e)
	}

	multiAddresses, err := multiAddressesForAddresses(true, addressesInOutput(output)...)
	if err != nil {
		return err
	}

	i.multiAddress.enqueue(multiAddresses...)

	return nil
}

func (i *ImportTransaction) Finalize(committedSlot iotago.SlotIndex, networkName string, databaseVersion uint32) error {
	// drain all processors
	i.basic.closeAndWait()
	i.nft.closeAndWait()
	i.account.closeAndWait()
	i.anchor.closeAndWait()
	i.foundry.closeAndWait()
	i.delegation.closeAndWait()
	i.multiAddress.closeAndWait()

	i.LogDebugf("Finished insertion, update committedSlot")

	// Update the indexer status
	status := &Status{
		ID:              1,
		CommittedSlot:   committedSlot,
		NetworkName:     networkName,
		DatabaseVersion: databaseVersion,
	}
	i.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&status)

	return i.db.Error
}
