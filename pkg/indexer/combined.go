package indexer

import (
	"time"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v3"
)

type CombinedFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	unlockableByAddress iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *time.Time
	createdAfter        *time.Time
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
		args.unlockableByAddress = address
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

func CombinedCreatedBefore(time time.Time) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.createdBefore = &time
	}
}

func CombinedCreatedAfter(time time.Time) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.createdAfter = &time
	}
}

func (o *CombinedFilterOptions) BasicFilterOptions() *BasicOutputFilterOptions {
	return &BasicOutputFilterOptions{
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

func (o *CombinedFilterOptions) FoundryFilterOptions() *FoundryFilterOptions {
	var aliasAddress *iotago.AliasAddress
	if o.unlockableByAddress != nil {
		var ok bool
		aliasAddress, ok = o.unlockableByAddress.(*iotago.AliasAddress)
		if !ok {
			return nil
		}
	}

	return &FoundryFilterOptions{
		hasNativeTokens:     o.hasNativeTokens,
		minNativeTokenCount: o.minNativeTokenCount,
		maxNativeTokenCount: o.maxNativeTokenCount,
		aliasAddress:        aliasAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) AliasFilterOptions() *AliasFilterOptions {
	return &AliasFilterOptions{
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

func (i *Indexer) CombinedOutputsWithFilters(filters ...options.Option[CombinedFilterOptions]) *IndexerResult {
	opts := options.Apply(new(CombinedFilterOptions), filters)

	var queries []*gorm.DB

	if filter := opts.BasicFilterOptions(); filter != nil {
		query, err := i.basicOutputsQueryWithFilter(filter)
		if err != nil {
			return errorResult(err)
		}
		queries = append(queries, query)
	}

	if filter := opts.AliasFilterOptions(); filter != nil {
		query, err := i.aliasQueryWithFilter(filter)
		if err != nil {
			return errorResult(err)
		}
		queries = append(queries, query)
	}

	if filter := opts.NFTFilterOptions(); filter != nil {
		query, err := i.nftOutputsQueryWithFilter(filter)
		if err != nil {
			return errorResult(err)
		}
		queries = append(queries, query)
	}

	if filter := opts.FoundryFilterOptions(); filter != nil {
		query, err := i.foundryOutputsQueryWithFilter(filter)
		if err != nil {
			return errorResult(err)
		}
		queries = append(queries, query)
	}

	return i.combineOutputIDFilteredQueries(queries, opts.pageSize, opts.cursor)
}
