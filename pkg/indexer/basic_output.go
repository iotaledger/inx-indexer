package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type basicOutput struct {
	OutputID                    []byte `gorm:"primaryKey;notnull"`
	NativeTokenCount            uint32 `gorm:"notnull;type:integer"`
	Sender                      []byte `gorm:"index:basic_outputs_sender_tag"`
	Tag                         []byte `gorm:"index:basic_outputs_sender_tag"`
	Address                     []byte `gorm:"notnull;index:basic_outputs_address"`
	StorageDepositReturn        *iotago.BaseToken
	StorageDepositReturnAddress []byte `gorm:"index:basic_outputs_storage_deposit_return_address"`
	TimelockSlot                *iotago.SlotIndex
	ExpirationSlot              *iotago.SlotIndex
	ExpirationReturnAddress     []byte           `gorm:"index:basic_outputs_expiration_return_address"`
	CreatedAt                   iotago.SlotIndex `gorm:"notnull;index:basic_outputs_created_at"`
}

func (o *basicOutput) String() string {
	return fmt.Sprintf("basic output => OutputID: %s", hex.EncodeToString(o.OutputID))
}

type BasicOutputFilterOptions struct {
	hasNativeTokens                  *bool
	minNativeTokenCount              *uint32
	maxNativeTokenCount              *uint32
	unlockableByAddress              iotago.Address
	address                          iotago.Address
	hasStorageDepositReturnCondition *bool
	storageDepositReturnAddress      iotago.Address
	hasExpirationCondition           *bool
	expirationReturnAddress          iotago.Address
	expiresBefore                    *iotago.SlotIndex
	expiresAfter                     *iotago.SlotIndex
	hasTimelockCondition             *bool
	timelockedBefore                 *iotago.SlotIndex
	timelockedAfter                  *iotago.SlotIndex
	sender                           iotago.Address
	tag                              []byte
	pageSize                         uint32
	cursor                           *string
	createdBefore                    *iotago.SlotIndex
	createdAfter                     *iotago.SlotIndex
}

func BasicOutputHasNativeTokens(value bool) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func BasicOutputMinNativeTokenCount(value uint32) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func BasicOutputMaxNativeTokenCount(value uint32) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func BasicOutputUnlockableByAddress(address iotago.Address) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.unlockableByAddress = address
	}
}

func BasicOutputUnlockAddress(address iotago.Address) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.address = address
	}
}

func BasicOutputHasStorageDepositReturnCondition(value bool) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.hasStorageDepositReturnCondition = &value
	}
}

func BasicOutputStorageDepositReturnAddress(address iotago.Address) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.storageDepositReturnAddress = address
	}
}

func BasicOutputHasExpirationCondition(value bool) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.hasExpirationCondition = &value
	}
}

func BasicOutputExpiresBefore(slot iotago.SlotIndex) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.expiresBefore = &slot
	}
}

func BasicOutputExpiresAfter(slot iotago.SlotIndex) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.expiresAfter = &slot
	}
}

func BasicOutputHasTimelockCondition(value bool) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.hasTimelockCondition = &value
	}
}

func BasicOutputTimelockedBefore(slot iotago.SlotIndex) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.timelockedBefore = &slot
	}
}

func BasicOutputTimelockedAfter(slot iotago.SlotIndex) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.timelockedAfter = &slot
	}
}

func BasicOutputExpirationReturnAddress(address iotago.Address) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.expirationReturnAddress = address
	}
}

func BasicOutputSender(address iotago.Address) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.sender = address
	}
}

func BasicOutputTag(tag []byte) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.tag = tag
	}
}

func BasicOutputPageSize(pageSize uint32) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.pageSize = pageSize
	}
}

func BasicOutputCursor(cursor string) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.cursor = &cursor
	}
}

func BasicOutputCreatedBefore(slot iotago.SlotIndex) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.createdBefore = &slot
	}
}

func BasicOutputCreatedAfter(slot iotago.SlotIndex) options.Option[BasicOutputFilterOptions] {
	return func(args *BasicOutputFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) basicQueryWithFilter(opts *BasicOutputFilterOptions) (*gorm.DB, error) {
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
		addr, err := addressBytesForAddress(opts.unlockableByAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("(address = ? OR expiration_return_address = ? OR storage_deposit_return_address = ?)", addr, addr, addr)
	}

	if opts.address != nil {
		addr, err := addressBytesForAddress(opts.address)
		if err != nil {
			return nil, err
		}
		query = query.Where("address = ?", addr)
	}

	if opts.hasStorageDepositReturnCondition != nil {
		if *opts.hasStorageDepositReturnCondition {
			query = query.Where("storage_deposit_return IS NOT NULL")
		} else {
			query = query.Where("storage_deposit_return IS NULL")
		}
	}

	if opts.storageDepositReturnAddress != nil {
		addr, err := addressBytesForAddress(opts.storageDepositReturnAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("storage_deposit_return_address = ?", addr)
	}

	if opts.hasExpirationCondition != nil {
		if *opts.hasExpirationCondition {
			query = query.Where("expiration_return_address IS NOT NULL")
		} else {
			query = query.Where("expiration_return_address IS NULL")
		}
	}

	if opts.expirationReturnAddress != nil {
		addr, err := addressBytesForAddress(opts.expirationReturnAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("expiration_return_address = ?", addr)
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
		addr, err := addressBytesForAddress(opts.sender)
		if err != nil {
			return nil, err
		}
		query = query.Where("sender = ?", addr)
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

func (i *Indexer) BasicOutputsWithFilters(filters ...options.Option[BasicOutputFilterOptions]) *IndexerResult {
	opts := options.Apply(new(BasicOutputFilterOptions), filters)
	query, err := i.basicQueryWithFilter(opts)
	if err != nil {
		return errorResult(err)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
