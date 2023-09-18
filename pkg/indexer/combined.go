package indexer

import (
	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type CombinedFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	unlockableByAddress *iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *iotago.SlotIndex
	createdAfter        *iotago.SlotIndex
}

func CombinedHasNativeTokens(value bool) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func CombinedMinNativeTokenCount(value uint32) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func CombinedMaxNativeTokenCount(value uint32) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func CombinedUnlockableByAddress(address iotago.Address) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.unlockableByAddress = &address
	}
}

func CombinedPageSize(pageSize uint32) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.pageSize = pageSize
	}
}

func CombinedCursor(cursor string) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.cursor = &cursor
	}
}

func CombinedCreatedBefore(slot iotago.SlotIndex) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.createdBefore = &slot
	}
}

func CombinedCreatedAfter(slot iotago.SlotIndex) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.createdAfter = &slot
	}
}

func (o *CombinedFilterOptions) BasicFilterOptions() *BasicFilterOptions {
	return &BasicFilterOptions{
		hasNativeTokens:     o.hasNativeTokens,
		minNativeTokenCount: o.minNativeTokenCount,
		maxNativeTokenCount: o.maxNativeTokenCount,
		unlockableByAddress: o.unlockableByAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) AccountFilterOptions() *AccountFilterOptions {
	return &AccountFilterOptions{
		hasNativeTokens:     o.hasNativeTokens,
		minNativeTokenCount: o.minNativeTokenCount,
		maxNativeTokenCount: o.maxNativeTokenCount,
		unlockableByAddress: o.unlockableByAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) NFTFilterOptions() *NFTFilterOptions {
	return &NFTFilterOptions{
		hasNativeTokens:     o.hasNativeTokens,
		minNativeTokenCount: o.minNativeTokenCount,
		maxNativeTokenCount: o.maxNativeTokenCount,
		unlockableByAddress: o.unlockableByAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) DelegationFilterOptions() *DelegationFilterOptions {
	return &DelegationFilterOptions{
		address:       o.unlockableByAddress,
		pageSize:      o.pageSize,
		cursor:        o.cursor,
		createdBefore: o.createdBefore,
		createdAfter:  o.createdAfter,
	}
}

func (i *Indexer) CombinedOutputsWithFilters(filter ...options.Option[CombinedFilterOptions]) *IndexerResult {
	opts := options.Apply(new(CombinedFilterOptions), filter)

	basicQuery, err := i.basicQueryWithFilter(opts.BasicFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	accountQuery, err := i.accountQueryWithFilter(opts.AccountFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	nftQuery, err := i.nftQueryWithFilter(opts.NFTFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	delegationQuery, err := i.delegationQueryWithFilter(opts.DelegationFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	queries := []*gorm.DB{basicQuery, accountQuery, nftQuery, delegationQuery}

	return i.combineOutputIDFilteredQueries(queries, opts.pageSize, opts.cursor)
}
