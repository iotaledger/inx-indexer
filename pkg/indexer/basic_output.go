package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type basic struct {
	OutputID                    []byte `gorm:"primaryKey;notnull"`
	Amount                      iotago.BaseToken
	NativeToken                 []byte
	NativeTokenAmount           *string
	Sender                      []byte `gorm:"index:basic_sender_tag"`
	Tag                         []byte `gorm:"index:basic_sender_tag"`
	Address                     []byte `gorm:"notnull;index:basic_address"`
	StorageDepositReturn        *iotago.BaseToken
	StorageDepositReturnAddress []byte `gorm:"index:basic_storage_deposit_return_address"`
	TimelockSlot                *iotago.SlotIndex
	ExpirationSlot              *iotago.SlotIndex
	ExpirationReturnAddress     []byte           `gorm:"index:basic_expiration_return_address"`
	CreatedAtSlot               iotago.SlotIndex `gorm:"notnull;index:basic_created_at_slot"`
	DeletedAtSlot               iotago.SlotIndex
	Committed                   bool
}

func (o *basic) String() string {
	return fmt.Sprintf("basic output => OutputID: %s", hex.EncodeToString(o.OutputID))
}

type BasicFilterOptions struct {
	hasNativeToken                   *bool
	nativeToken                      *iotago.NativeTokenID
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

func BasicHasNativeToken(value bool) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.hasNativeToken = &value
	}
}

func BasicNativeToken(tokenID iotago.NativeTokenID) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.nativeToken = &tokenID
	}
}

func BasicUnlockableByAddress(address iotago.Address) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.unlockableByAddress = address
	}
}

func BasicUnlockAddress(address iotago.Address) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.address = address
	}
}

func BasicHasStorageDepositReturnCondition(value bool) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.hasStorageDepositReturnCondition = &value
	}
}

func BasicStorageDepositReturnAddress(address iotago.Address) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.storageDepositReturnAddress = address
	}
}

func BasicHasExpirationCondition(value bool) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.hasExpirationCondition = &value
	}
}

func BasicExpiresBefore(slot iotago.SlotIndex) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.expiresBefore = &slot
	}
}

func BasicExpiresAfter(slot iotago.SlotIndex) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.expiresAfter = &slot
	}
}

func BasicHasTimelockCondition(value bool) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.hasTimelockCondition = &value
	}
}

func BasicTimelockedBefore(slot iotago.SlotIndex) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.timelockedBefore = &slot
	}
}

func BasicTimelockedAfter(slot iotago.SlotIndex) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.timelockedAfter = &slot
	}
}

func BasicExpirationReturnAddress(address iotago.Address) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.expirationReturnAddress = address
	}
}

func BasicSender(address iotago.Address) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.sender = address
	}
}

func BasicTag(tag []byte) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.tag = tag
	}
}

func BasicPageSize(pageSize uint32) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.pageSize = pageSize
	}
}

func BasicCursor(cursor string) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.cursor = &cursor
	}
}

func BasicCreatedBefore(slot iotago.SlotIndex) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.createdBefore = &slot
	}
}

func BasicCreatedAfter(slot iotago.SlotIndex) options.Option[BasicFilterOptions] {
	return func(args *BasicFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) basicQueryWithFilter(opts *BasicFilterOptions) *gorm.DB {
	query := i.db.Model(&basic{}).Where("deleted_at_slot == 0")

	if opts.hasNativeToken != nil {
		if *opts.hasNativeToken {
			query = query.Where("native_token_amount IS NOT NULL")
		} else {
			query = query.Where("native_token_amount IS NULL")
		}
	}

	if opts.nativeToken != nil {
		query = query.Where("native_token = ?", opts.nativeToken[:])
	}

	if opts.unlockableByAddress != nil {
		addrID := opts.unlockableByAddress.ID()
		query = query.Where("(address = ? OR expiration_return_address = ? OR storage_deposit_return_address = ?)", addrID, addrID, addrID)
	}

	if opts.address != nil {
		query = query.Where("address = ?", opts.address.ID())
	}

	if opts.hasStorageDepositReturnCondition != nil {
		if *opts.hasStorageDepositReturnCondition {
			query = query.Where("storage_deposit_return IS NOT NULL")
		} else {
			query = query.Where("storage_deposit_return IS NULL")
		}
	}

	if opts.storageDepositReturnAddress != nil {
		query = query.Where("storage_deposit_return_address = ?", opts.storageDepositReturnAddress.ID())
	}

	if opts.hasExpirationCondition != nil {
		if *opts.hasExpirationCondition {
			query = query.Where("expiration_return_address IS NOT NULL")
		} else {
			query = query.Where("expiration_return_address IS NULL")
		}
	}

	if opts.expirationReturnAddress != nil {
		query = query.Where("expiration_return_address = ?", opts.expirationReturnAddress.ID())
	}

	if opts.expiresBefore != nil {
		query = query.Where("expiration_slot < ?", *opts.expiresBefore)
	}

	if opts.expiresAfter != nil {
		query = query.Where("expiration_slot > ?", *opts.expiresAfter)
	}

	if opts.hasTimelockCondition != nil {
		if *opts.hasTimelockCondition {
			query = query.Where("timelock_slot IS NOT NULL")
		} else {
			query = query.Where("timelock_slot IS NULL")
		}
	}

	if opts.timelockedBefore != nil {
		query = query.Where("timelock_slot < ?", *opts.timelockedBefore)
	}

	if opts.timelockedAfter != nil {
		query = query.Where("timelock_slot > ?", *opts.timelockedAfter)
	}

	if opts.sender != nil {
		query = query.Where("sender = ?", opts.sender.ID())
	}

	if opts.tag != nil && len(opts.tag) > 0 {
		query = query.Where("tag = ?", opts.tag)
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at_slot < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at_slot > ?", *opts.createdAfter)
	}

	return query
}

func (i *Indexer) Basic(filters ...options.Option[BasicFilterOptions]) *IndexerResult {
	opts := options.Apply(new(BasicFilterOptions), filters)
	query := i.basicQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
