package indexer

import (
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	iotago "github.com/iotaledger/iota.go/v3"
)

type basicOutput struct {
	OutputID                    []byte `gorm:"primaryKey;notnull"`
	NativeTokenCount            uint32 `gorm:"notnull;type:integer"`
	Sender                      []byte `gorm:"index:basic_outputs_sender_tag"`
	Tag                         []byte `gorm:"index:basic_outputs_sender_tag"`
	Address                     []byte `gorm:"notnull;index:basic_outputs_address"`
	StorageDepositReturn        *uint64
	StorageDepositReturnAddress []byte `gorm:"index:basic_outputs_storage_deposit_return_address"`
	TimelockTime                *time.Time
	ExpirationTime              *time.Time
	ExpirationReturnAddress     []byte    `gorm:"index:basic_outputs_expiration_return_address"`
	CreatedAt                   time.Time `gorm:"notnull;index:basic_outputs_created_at"`
}

func (o *basicOutput) String() string {
	return fmt.Sprintf("basic output => outputID: %s", hex.EncodeToString(o.OutputID))
}

type BasicOutputFilterOptions struct {
	hasNativeTokens                  *bool
	minNativeTokenCount              *uint32
	maxNativeTokenCount              *uint32
	unlockableByAddress              *iotago.Address
	address                          *iotago.Address
	hasStorageDepositReturnCondition *bool
	storageDepositReturnAddress      *iotago.Address
	hasExpirationCondition           *bool
	expirationReturnAddress          *iotago.Address
	expiresBefore                    *time.Time
	expiresAfter                     *time.Time
	hasTimelockCondition             *bool
	timelockedBefore                 *time.Time
	timelockedAfter                  *time.Time
	sender                           *iotago.Address
	tag                              []byte
	pageSize                         uint32
	cursor                           *string
	createdBefore                    *time.Time
	createdAfter                     *time.Time
}

type BasicOutputFilterOption func(*BasicOutputFilterOptions)

func BasicOutputHasNativeTokens(value bool) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func BasicOutputMinNativeTokenCount(value uint32) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func BasicOutputMaxNativeTokenCount(value uint32) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func BasicOutputUnlockableByAddress(address iotago.Address) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.unlockableByAddress = &address
	}
}

func BasicOutputUnlockAddress(address iotago.Address) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.address = &address
	}
}

func BasicOutputHasStorageDepositReturnCondition(value bool) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.hasStorageDepositReturnCondition = &value
	}
}

func BasicOutputStorageDepositReturnAddress(address iotago.Address) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.storageDepositReturnAddress = &address
	}
}

func BasicOutputHasExpirationCondition(value bool) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.hasExpirationCondition = &value
	}
}

func BasicOutputExpiresBefore(time time.Time) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.expiresBefore = &time
	}
}

func BasicOutputExpiresAfter(time time.Time) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.expiresAfter = &time
	}
}

func BasicOutputHasTimelockCondition(value bool) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.hasTimelockCondition = &value
	}
}

func BasicOutputTimelockedBefore(time time.Time) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.timelockedBefore = &time
	}
}

func BasicOutputTimelockedAfter(time time.Time) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.timelockedAfter = &time
	}
}

func BasicOutputExpirationReturnAddress(address iotago.Address) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.expirationReturnAddress = &address
	}
}

func BasicOutputSender(address iotago.Address) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.sender = &address
	}
}

func BasicOutputTag(tag []byte) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.tag = tag
	}
}

func BasicOutputPageSize(pageSize uint32) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.pageSize = pageSize
	}
}

func BasicOutputCursor(cursor string) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.cursor = &cursor
	}
}

func BasicOutputCreatedBefore(time time.Time) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.createdBefore = &time
	}
}

func BasicOutputCreatedAfter(time time.Time) BasicOutputFilterOption {
	return func(args *BasicOutputFilterOptions) {
		args.createdAfter = &time
	}
}

func basicOutputFilterOptions(optionalOptions []BasicOutputFilterOption) *BasicOutputFilterOptions {
	result := &BasicOutputFilterOptions{}

	for _, optionalOption := range optionalOptions {
		optionalOption(result)
	}

	return result
}

func (i *Indexer) basicOutputsQueryWithFilter(opts *BasicOutputFilterOptions) (*gorm.DB, error) {
	query := i.db.Model(&basicOutput{})

	if opts.hasNativeTokens != nil {
		if *opts.hasNativeTokens {
			query = query.Where("native_token_count > 0")
		} else {
			query = query.Where("native_token_count = 0")
		}
	}

	if opts.minNativeTokenCount != nil {
		query = query.Where("native_token_count >= ?", *opts.minNativeTokenCount)
	}

	if opts.maxNativeTokenCount != nil {
		query = query.Where("native_token_count <= ?", *opts.maxNativeTokenCount)
	}

	if opts.unlockableByAddress != nil {
		addr, err := addressBytesForAddress(*opts.unlockableByAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("(address = ? OR expiration_return_address = ? OR storage_deposit_return_address = ?)", addr[:], addr[:], addr[:])
	}

	if opts.address != nil {
		addr, err := addressBytesForAddress(*opts.address)
		if err != nil {
			return nil, err
		}
		query = query.Where("address = ?", addr[:])
	}

	if opts.hasStorageDepositReturnCondition != nil {
		if *opts.hasStorageDepositReturnCondition {
			query = query.Where("storage_deposit_return IS NOT NULL")
		} else {
			query = query.Where("storage_deposit_return IS NULL")
		}
	}

	if opts.storageDepositReturnAddress != nil {
		addr, err := addressBytesForAddress(*opts.storageDepositReturnAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("storage_deposit_return_address = ?", addr[:])
	}

	if opts.hasExpirationCondition != nil {
		if *opts.hasExpirationCondition {
			query = query.Where("expiration_return_address IS NOT NULL")
		} else {
			query = query.Where("expiration_return_address IS NULL")
		}
	}

	if opts.expirationReturnAddress != nil {
		addr, err := addressBytesForAddress(*opts.expirationReturnAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("expiration_return_address = ?", addr[:])
	}

	if opts.expiresBefore != nil {
		query = query.Where("expiration_time < ?", *opts.expiresBefore)
	}

	if opts.expiresAfter != nil {
		query = query.Where("expiration_time > ?", *opts.expiresAfter)
	}

	if opts.hasTimelockCondition != nil {
		if *opts.hasTimelockCondition {
			query = query.Where("timelock_time IS NOT NULL")
		} else {
			query = query.Where("timelock_time IS NULL")
		}
	}

	if opts.timelockedBefore != nil {
		query = query.Where("timelock_time < ?", *opts.timelockedBefore)
	}

	if opts.timelockedAfter != nil {
		query = query.Where("timelock_time > ?", *opts.timelockedAfter)
	}

	if opts.sender != nil {
		addr, err := addressBytesForAddress(*opts.sender)
		if err != nil {
			return nil, err
		}
		query = query.Where("sender = ?", addr[:])
	}

	if opts.tag != nil && len(opts.tag) > 0 {
		query = query.Where("tag = ?", opts.tag)
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return query, nil
}

func (i *Indexer) BasicOutputsWithFilters(filters ...BasicOutputFilterOption) *IndexerResult {
	opts := basicOutputFilterOptions(filters)
	query, err := i.basicOutputsQueryWithFilter(opts)
	if err != nil {
		return errorResult(err)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
