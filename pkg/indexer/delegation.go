package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type delegation struct {
	OutputID      []byte `gorm:"primaryKey;notnull"`
	DelegationID  []byte `gorm:"notnull"`
	Amount        iotago.BaseToken
	Address       []byte           `gorm:"notnull;index:delegations_address"`
	Validator     []byte           `gorm:"index:delegations_validator"`
	CreatedAtSlot iotago.SlotIndex `gorm:"notnull;index:delegations_created_at_slot"`
	DeletedAtSlot iotago.SlotIndex `gorm:"notnull;index:delegations_deleted_at_slot"`
	Committed     bool
}

func (d *delegation) String() string {
	return fmt.Sprintf("delegation output => DelegationID: %s, OutputID: %s", hex.EncodeToString(d.DelegationID), hex.EncodeToString(d.OutputID))
}

type DelegationFilterOptions struct {
	address       iotago.Address
	validator     *iotago.AccountAddress
	pageSize      uint32
	cursor        *string
	createdBefore *iotago.SlotIndex
	createdAfter  *iotago.SlotIndex
}

func DelegationAddress(address iotago.Address) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.address = address
	}
}

func DelegationValidator(address *iotago.AccountAddress) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.validator = address
	}
}

func DelegationPageSize(pageSize uint32) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.pageSize = pageSize
	}
}

func DelegationCursor(cursor string) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.cursor = &cursor
	}
}

func DelegationCreatedBefore(slot iotago.SlotIndex) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.createdBefore = &slot
	}
}

func DelegationCreatedAfter(slot iotago.SlotIndex) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) DelegationByID(delegationID iotago.DelegationID) *IndexerResult {
	query := i.db.Model(&delegation{}).
		Where("delegation_id = ?", delegationID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) delegationQueryWithFilter(opts *DelegationFilterOptions) *gorm.DB {
	query := i.db.Model(&delegation{}).Where("deleted_at_slot = 0")

	if opts.address != nil {
		query = query.Where("address = ?", opts.address.ID())
	}

	if opts.validator != nil {
		query = query.Where("validator = ?", opts.validator.ID())
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at_slot < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at_slot > ?", *opts.createdAfter)
	}

	return query
}

func (i *Indexer) Delegation(filters ...options.Option[DelegationFilterOptions]) *IndexerResult {
	opts := options.Apply(&DelegationFilterOptions{
		pageSize: DefaultPageSize,
	}, filters)
	query := i.delegationQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
