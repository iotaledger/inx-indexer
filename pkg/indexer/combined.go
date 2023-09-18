package indexer

import (
	"time"

	"gorm.io/gorm"

	iotago "github.com/iotaledger/iota.go/v3"
)

type CombinedFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	unlockableByAddress *iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *time.Time
	createdAfter        *time.Time
}

type CombinedFilterOption func(*CombinedFilterOptions)

func CombinedHasNativeTokens(value bool) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func CombinedMinNativeTokenCount(value uint32) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func CombinedMaxNativeTokenCount(value uint32) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func CombinedUnlockableByAddress(address iotago.Address) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.unlockableByAddress = &address
	}
}

func CombinedPageSize(pageSize uint32) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.pageSize = pageSize
	}
}

func CombinedCursor(cursor string) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.cursor = &cursor
	}
}

func CombinedCreatedBefore(time time.Time) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.createdBefore = &time
	}
}

func CombinedCreatedAfter(time time.Time) CombinedFilterOption {
	return func(args *CombinedFilterOptions) {
		args.createdAfter = &time
	}
}

func combinedFilterOptions(optionalOptions []CombinedFilterOption) *CombinedFilterOptions {
	result := &CombinedFilterOptions{}

	for _, optionalOption := range optionalOptions {
		optionalOption(result)
	}

	return result
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

func (i *Indexer) CombinedOutputsWithFilters(filters ...CombinedFilterOption) *IndexerResult {
	opts := combinedFilterOptions(filters)

	basicQuery, err := i.basicOutputsQueryWithFilter(opts.BasicFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	aliasQuery, err := i.aliasQueryWithFilter(opts.AliasFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	nftQuery, err := i.nftOutputsQueryWithFilter(opts.NFTFilterOptions())
	if err != nil {
		return errorResult(err)
	}

	queries := []*gorm.DB{basicQuery, aliasQuery, nftQuery}

	return i.combineOutputIDFilteredQueries(queries, opts.pageSize, opts.cursor)
}
