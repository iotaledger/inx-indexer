package indexer

import (
	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type foundry struct {
	FoundryID        foundryIDBytes   `gorm:"primaryKey;notnull"`
	OutputID         outputIDBytes    `gorm:"unique;notnull"`
	NativeTokenCount uint32           `gorm:"notnull;type:integer"`
	AccountAddress   addressBytes     `gorm:"notnull;index:foundries_account_address"`
	CreatedAt        iotago.SlotIndex `gorm:"notnull;index:foundries_created_at"`
}

type FoundryFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	accountAddress      *iotago.AccountAddress
	pageSize            uint32
	cursor              *string
	createdBefore       *iotago.SlotIndex
	createdAfter        *iotago.SlotIndex
}

func FoundryHasNativeTokens(value bool) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func FoundryMinNativeTokenCount(value uint32) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func FoundryMaxNativeTokenCount(value uint32) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func FoundryWithAccountAddress(address *iotago.AccountAddress) options.Option[FoundryFilterOptions] {
	return func(args *FoundryFilterOptions) {
		args.accountAddress = address
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

func (i *Indexer) FoundryOutputsWithFilters(filters ...options.Option[FoundryFilterOptions]) *IndexerResult {
	opts := options.Apply(new(FoundryFilterOptions), filters)
	query := i.db.Model(&foundry{})

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

	if opts.accountAddress != nil {
		addr, err := addressBytesForAddress(opts.accountAddress)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("alias_address = ?", addr[:])
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
