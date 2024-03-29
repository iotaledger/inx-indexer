package indexer

import (
	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type CombinedFilterOptions struct {
	hasNativeToken      *bool
	nativeToken         *iotago.NativeTokenID
	unlockableByAddress iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *iotago.SlotIndex
	createdAfter        *iotago.SlotIndex
}

func CombinedHasNativeToken(value bool) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.hasNativeToken = &value
	}
}

func CombinedNativeToken(tokenID iotago.NativeTokenID) options.Option[CombinedFilterOptions] {
	return func(args *CombinedFilterOptions) {
		args.nativeToken = &tokenID
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
		hasNativeToken:      o.hasNativeToken,
		nativeToken:         o.nativeToken,
		unlockableByAddress: o.unlockableByAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) FoundryFilterOptions() *FoundryFilterOptions {
	var accountAddress *iotago.AccountAddress
	if o.unlockableByAddress != nil {
		var ok bool
		accountAddress, ok = o.unlockableByAddress.(*iotago.AccountAddress)
		if !ok {
			return nil
		}
	}

	return &FoundryFilterOptions{
		hasNativeToken: o.hasNativeToken,
		nativeToken:    o.nativeToken,
		account:        accountAddress,
		pageSize:       o.pageSize,
		cursor:         o.cursor,
		createdBefore:  o.createdBefore,
		createdAfter:   o.createdAfter,
	}
}

func (o *CombinedFilterOptions) AccountFilterOptions() *AccountFilterOptions {
	if o.hasNativeToken != nil && *o.hasNativeToken {
		// Do not support native tokens
		return nil
	}

	return &AccountFilterOptions{
		address:       o.unlockableByAddress,
		pageSize:      o.pageSize,
		cursor:        o.cursor,
		createdBefore: o.createdBefore,
		createdAfter:  o.createdAfter,
	}
}

func (o *CombinedFilterOptions) AnchorFilterOptions() *AnchorFilterOptions {
	if o.hasNativeToken != nil && *o.hasNativeToken {
		// Do not support native tokens
		return nil
	}

	return &AnchorFilterOptions{
		unlockableByAddress: o.unlockableByAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) NFTFilterOptions() *NFTFilterOptions {
	if o.hasNativeToken != nil && *o.hasNativeToken {
		// Do not support native tokens
		return nil
	}

	return &NFTFilterOptions{
		unlockableByAddress: o.unlockableByAddress,
		pageSize:            o.pageSize,
		cursor:              o.cursor,
		createdBefore:       o.createdBefore,
		createdAfter:        o.createdAfter,
	}
}

func (o *CombinedFilterOptions) DelegationFilterOptions() *DelegationFilterOptions {
	if o.hasNativeToken != nil && *o.hasNativeToken {
		// Do not support native tokens
		return nil
	}

	return &DelegationFilterOptions{
		address:       o.unlockableByAddress,
		pageSize:      o.pageSize,
		cursor:        o.cursor,
		createdBefore: o.createdBefore,
		createdAfter:  o.createdAfter,
	}
}

func (i *Indexer) Combined(filters ...options.Option[CombinedFilterOptions]) *IndexerResult {
	opts := options.Apply(&CombinedFilterOptions{
		pageSize: DefaultPageSize,
	}, filters)

	var queries []*gorm.DB

	if filter := opts.BasicFilterOptions(); filter != nil {
		queries = append(queries, i.basicQueryWithFilter(filter))
	}

	if filter := opts.AccountFilterOptions(); filter != nil {
		queries = append(queries, i.accountQueryWithFilter(filter))
	}

	if filter := opts.AnchorFilterOptions(); filter != nil {
		queries = append(queries, i.anchorQueryWithFilter(filter))
	}

	if filter := opts.NFTFilterOptions(); filter != nil {
		queries = append(queries, i.nftQueryWithFilter(filter))
	}

	if filter := opts.FoundryFilterOptions(); filter != nil {
		queries = append(queries, i.foundryOutputsQueryWithFilter(filter))
	}

	if filter := opts.DelegationFilterOptions(); filter != nil {
		queries = append(queries, i.delegationQueryWithFilter(filter))
	}

	return i.combineOutputIDFilteredQueries(queries, opts.pageSize, opts.cursor)
}
