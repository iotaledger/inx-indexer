package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type account struct {
	AccountID       []byte `gorm:"primaryKey;notnull"`
	OutputID        []byte `gorm:"unique;notnull"`
	Amount          iotago.BaseToken
	StateController []byte           `gorm:"notnull;index:account_state_controller"`
	Governor        []byte           `gorm:"notnull;index:account_governor"`
	Issuer          []byte           `gorm:"index:account_issuer"`
	Sender          []byte           `gorm:"index:account_sender"`
	CreatedAt       iotago.SlotIndex `gorm:"notnull;index:account_created_at"`
}

func (a *account) String() string {
	return fmt.Sprintf("account output => AccountID: %s, OutputID: %s", hex.EncodeToString(a.AccountID), hex.EncodeToString(a.OutputID))
}

type AccountFilterOptions struct {
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

func (i *Indexer) AccountByID(accountID iotago.AccountID) *IndexerResult {
	query := i.db.Model(&account{}).
		Where("account_id = ?", accountID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) accountQueryWithFilter(opts *AccountFilterOptions) *gorm.DB {
	query := i.db.Model(&account{})

	if opts.unlockableByAddress != nil {
		addrID := opts.unlockableByAddress.ID()
		query = query.Where("(state_controller = ? OR governor = ?)", addrID, addrID)
	}

	if opts.stateController != nil {
		query = query.Where("state_controller = ?", opts.stateController.ID())
	}

	if opts.governor != nil {
		query = query.Where("governor = ?", opts.governor.ID())
	}

	if opts.sender != nil {
		query = query.Where("sender = ?", opts.sender.ID())
	}

	if opts.issuer != nil {
		query = query.Where("issuer = ?", opts.issuer.ID())
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return query
}

func (i *Indexer) Account(filters ...options.Option[AccountFilterOptions]) *IndexerResult {
	opts := options.Apply(new(AccountFilterOptions), filters)
	query := i.accountQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
