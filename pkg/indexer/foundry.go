package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type foundry struct {
	FoundryID         []byte `gorm:"primaryKey;notnull"`
	OutputID          []byte `gorm:"unique;notnull"`
	Amount            iotago.BaseToken
	NativeTokenAmount *string
	AccountAddress    []byte           `gorm:"notnull;index:foundries_account_address"`
	CreatedAt         iotago.SlotIndex `gorm:"notnull;index:foundries_created_at"`
	DeletedAt         iotago.SlotIndex
	Committed         bool
}

func (o *foundry) String() string {
	return fmt.Sprintf("foundry output => FoundryID: %s, OutputID: %s", hex.EncodeToString(o.FoundryID), hex.EncodeToString(o.OutputID))
}

type FoundryFilterOptions struct {
	hasNativeToken *bool
	nativeToken    *iotago.NativeTokenID
	account        *iotago.AccountAddress
	pageSize       uint32
	cursor         *string
	createdBefore  *iotago.SlotIndex
	createdAfter   *iotago.SlotIndex
}

func FoundryHasNativeToken(value bool) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.hasNativeToken = &value
	}
}

func FoundryNativeToken(tokenID iotago.NativeTokenID) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.nativeToken = &tokenID
	}
}

func FoundryWithAccountAddress(address *iotago.AccountAddress) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.account = address
	}
}

func FoundryPageSize(pageSize uint32) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.pageSize = pageSize
	}
}

func FoundryCursor(cursor string) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.cursor = &cursor
	}
}

func FoundryCreatedBefore(slot iotago.SlotIndex) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.createdBefore = &slot
	}
}

func FoundryCreatedAfter(slot iotago.SlotIndex) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) FoundryByID(foundryID iotago.FoundryID) *IndexerResult {
	query := i.db.Model(&foundry{}).
		Where("foundry_id = ?", foundryID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) foundryOutputsQueryWithFilter(opts *FoundryFilterOptions) *gorm.DB {
	query := i.db.Model(&foundry{}).Where("deleted_at == 0")

	if opts.hasNativeToken != nil {
		if *opts.hasNativeToken {
			query = query.Where("native_token_amount IS NOT NULL")
		} else {
			query = query.Where("native_token_amount IS NULL")
		}
	}

	// Since the foundry can only hold its own native token, we can filter out by foundry_id here.
	if opts.nativeToken != nil {
		query = query.Where("foundry_id = ?", opts.nativeToken[:])
	}

	if opts.account != nil {
		query = query.Where("account_address = ?", opts.account.ID())
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return query
}

func (i *Indexer) Foundry(filters ...options.Option[FoundryFilterOptions]) *IndexerResult {
	opts := options.Apply(new(FoundryFilterOptions), filters)
	query := i.foundryOutputsQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
