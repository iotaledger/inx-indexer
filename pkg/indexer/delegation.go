package indexer

import (
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type delegation struct {
	DelegationID delegationIDBytes `gorm:"primaryKey;notnull"`
	OutputID     outputIDBytes     `gorm:"unique;notnull"`
	Address      addressBytes      `gorm:"notnull;index:delegation_outputs_address"`
	Validator    addressBytes      `gorm:"index:delegation_outputs_validator"`
	CreatedAt    iotago.SlotIndex  `gorm:"notnull;index:delegation_outputs_created_at"`
}

func (d *delegation) String() string {
	return fmt.Sprintf("delegation output => DelegationID: %s, OutputID: %s", hex.EncodeToString(d.DelegationID), hex.EncodeToString(d.OutputID))
}

type DelegationFilterOptions struct {
	unlockableByAddress *iotago.Address
	validator           *iotago.AccountAddress
	pageSize            uint32
	cursor              *string
	createdBefore       *iotago.SlotIndex
	createdAfter        *iotago.SlotIndex
}

func DelegationUnlockableByAddress(address iotago.Address) options.Option[DelegationFilterOptions] {
	return func(args *DelegationFilterOptions) {
		args.unlockableByAddress = &address
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

func (i *Indexer) DelegationOutput(delegationID iotago.DelegationID) *IndexerResult {
	query := i.db.Model(&delegation{}).
		Where("delegation_id = ?", delegationID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) DelegationsWithFilters(filters ...options.Option[DelegationFilterOptions]) *IndexerResult {
	opts := options.Apply(new(DelegationFilterOptions), filters)
	query := i.db.Model(&delegation{})

	if opts.unlockableByAddress != nil {
		addr, err := addressBytesForAddress(*opts.unlockableByAddress)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("address = ?", addr[:])
	}

	if opts.validator != nil {
		addr, err := addressBytesForAddress(opts.validator)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("validator = ?", addr[:])
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
