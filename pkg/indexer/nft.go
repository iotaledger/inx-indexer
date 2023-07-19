package indexer

import (
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type nft struct {
	NFTID                       []byte `gorm:"primaryKey;notnull"`
	OutputID                    []byte `gorm:"unique;notnull"`
	NativeTokenCount            uint32 `gorm:"notnull;type:integer"`
	Issuer                      []byte `gorm:"index:nfts_issuer"`
	Sender                      []byte `gorm:"index:nfts_sender_tag"`
	Tag                         []byte `gorm:"index:nfts_sender_tag"`
	Address                     []byte `gorm:"notnull;index:nfts_address"`
	StorageDepositReturn        *uint64
	StorageDepositReturnAddress []byte `gorm:"index:nfts_storage_deposit_return_address"`
	TimelockTime                *iotago.SlotIndex
	ExpirationTime              *iotago.SlotIndex
	ExpirationReturnAddress     []byte    `gorm:"index:nfts_expiration_return_address"`
	CreatedAt                   iotago.SlotIndex `gorm:"notnull;index:nfts_created_at"`
}

func (o *nft) String() string {
	return fmt.Sprintf("nft output => NFTID: %s outputID: %s", hex.EncodeToString(o.NFTID), hex.EncodeToString(o.OutputID))
}

type NFTFilterOptions struct {
	hasNativeTokens                  *bool
	minNativeTokenCount              *uint32
	maxNativeTokenCount              *uint32
	unlockableByAddress              *iotago.Address
	address                          *iotago.Address
	hasStorageDepositReturnCondition *bool
	storageDepositReturnAddress      *iotago.Address
	hasExpirationCondition           *bool
	expirationReturnAddress          *iotago.Address
	expiresBefore                    *iotago.SlotIndex
	expiresAfter                     *iotago.SlotIndex
	hasTimelockCondition             *bool
	timelockedBefore                 *iotago.SlotIndex
	timelockedAfter                  *iotago.SlotIndex
	issuer                           *iotago.Address
	sender                           *iotago.Address
	tag                              []byte
	pageSize                         uint32
	cursor                           *string
	createdBefore                    *iotago.SlotIndex
	createdAfter                     *iotago.SlotIndex
}

func NFTHasNativeTokens(value bool) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func NFTMinNativeTokenCount(value uint32) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func NFTMaxNativeTokenCount(value uint32) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func NFTUnlockableByAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.unlockableByAddress = &address
	}
}

func NFTUnlockAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.address = &address
	}
}

func NFTHasStorageDepositReturnCondition(value bool) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.hasStorageDepositReturnCondition = &value
	}
}

func NFTStorageDepositReturnAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.storageDepositReturnAddress = &address
	}
}

func NFTExpirationReturnAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.expirationReturnAddress = &address
	}
}

func NFTHasExpirationCondition(value bool) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.hasExpirationCondition = &value
	}
}

func NFTExpiresBefore(slot iotago.SlotIndex) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.expiresBefore = &slot
	}
}

func NFTExpiresAfter(slot iotago.SlotIndex) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.expiresAfter = &slot
	}
}

func NFTHasTimelockCondition(value bool) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.hasTimelockCondition = &value
	}
}

func NFTTimelockedBefore(slot iotago.SlotIndex) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.timelockedBefore = &slot
	}
}

func NFTTimelockedAfter(slot iotago.SlotIndex) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.timelockedAfter = &slot
	}
}

func NFTIssuer(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.issuer = &address
	}
}

func NFTSender(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.sender = &address
	}
}

func NFTTag(tag []byte) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.tag = tag
	}
}

func NFTPageSize(pageSize uint32) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.pageSize = pageSize
	}
}

func NFTCursor(cursor string) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.cursor = &cursor
	}
}

func NFTCreatedBefore(slot iotago.SlotIndex) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.createdBefore = &slot
	}
}

func NFTCreatedAfter(slot iotago.SlotIndex) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) NFTOutput(nftID *iotago.NFTID) *IndexerResult {
	query := i.db.Model(&nft{}).
		Where("nft_id = ?", nftID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) nftOutputsQueryWithFilter(opts *NFTFilterOptions) (*gorm.DB, error) {
	query := i.db.Model(&nft{})

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
		query = query.Where("(address = ? OR expiration_return_address = ? OR storage_deposit_return_address = ?)", addr, addr, addr)
	}

	if opts.address != nil {
		addr, err := addressBytesForAddress(*opts.address)
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
		addr, err := addressBytesForAddress(*opts.storageDepositReturnAddress)
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
		addr, err := addressBytesForAddress(*opts.expirationReturnAddress)
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

	if opts.issuer != nil {
		addr, err := addressBytesForAddress(*opts.issuer)
		if err != nil {
			return nil, err
		}
		query = query.Where("issuer = ?", addr)
	}

	if opts.sender != nil {
		addr, err := addressBytesForAddress(*opts.sender)
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

func (i *Indexer) NFTOutputsWithFilters(filters ...options.Option[NFTFilterOptions]) *IndexerResult {
	opts := options.Apply(new(NFTFilterOptions), filters)
	query, err := i.nftOutputsQueryWithFilter(opts)
	if err != nil {
		return errorResult(err)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
