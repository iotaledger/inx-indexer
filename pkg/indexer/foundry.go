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
	NativeTokenAmount string
	AccountAddress    []byte           `gorm:"notnull;index:foundries_account_address"`
	CreatedAt         iotago.SlotIndex `gorm:"notnull;index:foundries_created_at"`
}

func (o *foundry) String() string {
	return fmt.Sprintf("foundry output => FoundryID: %s, OutputID: %s", hex.EncodeToString(o.FoundryID), hex.EncodeToString(o.OutputID))
}

type FoundryFilterOptions struct {
	hasNativeTokens *bool
	account         *iotago.AccountAddress
	pageSize        uint32
	cursor          *string
	createdBefore   *iotago.SlotIndex
	createdAfter    *iotago.SlotIndex
}

func FoundryHasNativeTokens(value bool) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.hasNativeTokens = &value
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

func (i *Indexer) FoundryOutput(foundryID iotago.FoundryID) *IndexerResult {
	query := i.db.Model(&foundry{}).
		Where("foundry_id = ?", foundryID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) foundryOutputsQueryWithFilter(opts *FoundryFilterOptions) *gorm.DB {
	query := i.db.Model(&foundry{})

	if opts.hasNativeTokens != nil {
		if *opts.hasNativeTokens {
			query = query.Where("native_token_amount != null")
		} else {
			query = query.Where("native_token_amount == null")
		}
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

func (i *Indexer) FoundryOutputsWithFilters(filters ...options.Option[FoundryFilterOptions]) *IndexerResult {
	opts := options.Apply(new(FoundryFilterOptions), filters)
	query := i.foundryOutputsQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
