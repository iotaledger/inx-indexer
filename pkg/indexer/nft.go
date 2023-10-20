package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type nft struct {
	NFTID                       []byte `gorm:"primaryKey;notnull"`
	OutputID                    []byte `gorm:"unique;notnull"`
	Amount                      iotago.BaseToken
	Issuer                      []byte `gorm:"index:nfts_issuer"`
	Sender                      []byte `gorm:"index:nfts_sender_tag"`
	Tag                         []byte `gorm:"index:nfts_sender_tag"`
	Address                     []byte `gorm:"notnull;index:nfts_address"`
	StorageDepositReturn        *uint64
	StorageDepositReturnAddress []byte `gorm:"index:nfts_storage_deposit_return_address"`
	TimelockSlot                *iotago.SlotIndex
	ExpirationSlot              *iotago.SlotIndex
	ExpirationReturnAddress     []byte           `gorm:"index:nfts_expiration_return_address"`
	CreatedAt                   iotago.SlotIndex `gorm:"notnull;index:nfts_created_at"`
	DeletedAt                   iotago.SlotIndex
	Committed                   bool
}

func (o *nft) String() string {
	return fmt.Sprintf("nft output => NFTID: %s, OutputID: %s", hex.EncodeToString(o.NFTID), hex.EncodeToString(o.OutputID))
}

type NFTFilterOptions struct {
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
	issuer                           iotago.Address
	sender                           iotago.Address
	tag                              []byte
	pageSize                         uint32
	cursor                           *string
	createdBefore                    *iotago.SlotIndex
	createdAfter                     *iotago.SlotIndex
}

func NFTUnlockableByAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.unlockableByAddress = address
	}
}

func NFTUnlockAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.address = address
	}
}

func NFTHasStorageDepositReturnCondition(value bool) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.hasStorageDepositReturnCondition = &value
	}
}

func NFTStorageDepositReturnAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.storageDepositReturnAddress = address
	}
}

func NFTExpirationReturnAddress(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.expirationReturnAddress = address
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
		args.issuer = address
	}
}

func NFTSender(address iotago.Address) options.Option[NFTFilterOptions] {
	return func(args *NFTFilterOptions) {
		args.sender = address
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

func (i *Indexer) NFTByID(nftID iotago.NFTID) *IndexerResult {
	query := i.db.Model(&nft{}).
		Where("nft_id = ?", nftID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) nftQueryWithFilter(opts *NFTFilterOptions) *gorm.DB {
	query := i.db.Model(&nft{}).Where("deleted_at == 0")

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

	if opts.issuer != nil {
		query = query.Where("issuer = ?", opts.issuer.ID())
	}

	if opts.sender != nil {
		query = query.Where("sender = ?", opts.sender.ID())
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

	return query
}

func (i *Indexer) NFT(filters ...options.Option[NFTFilterOptions]) *IndexerResult {
	opts := options.Apply(new(NFTFilterOptions), filters)
	query := i.nftQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
