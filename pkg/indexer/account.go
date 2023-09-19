package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type account struct {
	AccountID        []byte           `gorm:"primaryKey;notnull"`
	OutputID         []byte           `gorm:"unique;notnull"`
	NativeTokenCount uint32           `gorm:"notnull;type:integer"`
	StateController  []byte           `gorm:"notnull;index:account_state_controller"`
	Governor         []byte           `gorm:"notnull;index:account_governor"`
	Issuer           []byte           `gorm:"index:account_issuer"`
	Sender           []byte           `gorm:"index:account_sender"`
	CreatedAt        iotago.SlotIndex `gorm:"notnull;index:account_created_at"`
}

func (a *account) String() string {
	return fmt.Sprintf("account output => AccountID: %s, OutputID: %s", hex.EncodeToString(a.AccountID), hex.EncodeToString(a.OutputID))
}

type AccountFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	unlockableByAddress iotago.Address
	stateController     iotago.Address
	governor            iotago.Address
	issuer              iotago.Address
	sender              iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *iotago.SlotIndex
	createdAfter        *iotago.SlotIndex
}

func AccountHasNativeTokens(value bool) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func AccountMinNativeTokenCount(value uint32) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func AccountMaxNativeTokenCount(value uint32) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func AccountUnlockableByAddress(address iotago.Address) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.unlockableByAddress = address
	}
}

func AccountStateController(address iotago.Address) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.stateController = address
	}
}

func AccountGovernor(address iotago.Address) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.governor = address
	}
}

func AccountSender(address iotago.Address) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.sender = address
	}
}

func AccountIssuer(address iotago.Address) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.issuer = address
	}
}

func AccountPageSize(pageSize uint32) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.pageSize = pageSize
	}
}

func AccountCursor(cursor string) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.cursor = &cursor
	}
}

func AccountCreatedBefore(slot iotago.SlotIndex) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.createdBefore = &slot
	}
}

func AccountCreatedAfter(slot iotago.SlotIndex) options.Option[AccountFilterOptions] {
	return func(args *AccountFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) AccountOutput(accountID iotago.AccountID) *IndexerResult {
	query := i.db.Model(&account{}).
		Where("account_id = ?", accountID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) accountQueryWithFilter(opts *AccountFilterOptions) (*gorm.DB, error) {
	query := i.db.Model(&account{})

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
		addr, err := addressBytesForAddress(opts.unlockableByAddress)
		if err != nil {
			return nil, err
		}
		query = query.Where("(state_controller = ? OR governor = ?)", addr, addr)
	}

	if opts.stateController != nil {
		addr, err := addressBytesForAddress(opts.stateController)
		if err != nil {
			return nil, err
		}
		query = query.Where("state_controller = ?", addr)
	}

	if opts.governor != nil {
		addr, err := addressBytesForAddress(opts.governor)
		if err != nil {
			return nil, err
		}
		query = query.Where("governor = ?", addr)
	}

	if opts.sender != nil {
		addr, err := addressBytesForAddress(opts.sender)
		if err != nil {
			return nil, err
		}
		query = query.Where("sender = ?", addr)
	}

	if opts.issuer != nil {
		addr, err := addressBytesForAddress(opts.issuer)
		if err != nil {
			return nil, err
		}
		query = query.Where("issuer = ?", addr)
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return query, nil
}

func (i *Indexer) AccountOutputsWithFilters(filters ...options.Option[AccountFilterOptions]) *IndexerResult {
	opts := options.Apply(new(AccountFilterOptions), filters)
	query, err := i.accountQueryWithFilter(opts)
	if err != nil {
		return errorResult(err)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
