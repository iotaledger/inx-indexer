package indexer

import (
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/iotaledger/hive.go/runtime/options"
	iotago "github.com/iotaledger/iota.go/v4"
)

type anchor struct {
	OutputID        []byte `gorm:"primaryKey;notnull"`
	AnchorID        []byte `gorm:"notnull"`
	Amount          iotago.BaseToken
	StateController []byte           `gorm:"notnull;index:anchors_state_controller"`
	Governor        []byte           `gorm:"notnull;index:anchors_governor"`
	Issuer          []byte           `gorm:"index:anchors_issuer"`
	Sender          []byte           `gorm:"index:anchors_sender"`
	CreatedAtSlot   iotago.SlotIndex `gorm:"notnull;index:anchors_created_at_slot"`
	DeletedAtSlot   iotago.SlotIndex `gorm:"notnull;index:anchors_deleted_at_slot"`
	Committed       bool
}

func (a *anchor) String() string {
	return fmt.Sprintf("anchor output => AnchorID: %s, OutputID: %s", hex.EncodeToString(a.AnchorID), hex.EncodeToString(a.OutputID))
}

type AnchorFilterOptions struct {
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

func AnchorUnlockableByAddress(address iotago.Address) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.unlockableByAddress = address
	}
}

func AnchorStateController(address iotago.Address) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.stateController = address
	}
}

func AnchorGovernor(address iotago.Address) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.governor = address
	}
}

func AnchorSender(address iotago.Address) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.sender = address
	}
}

func AnchorIssuer(address iotago.Address) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.issuer = address
	}
}

func AnchorPageSize(pageSize uint32) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.pageSize = pageSize
	}
}

func AnchorCursor(cursor string) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.cursor = &cursor
	}
}

func AnchorCreatedBefore(slot iotago.SlotIndex) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.createdBefore = &slot
	}
}

func AnchorCreatedAfter(slot iotago.SlotIndex) options.Option[AnchorFilterOptions] {
	return func(args *AnchorFilterOptions) {
		args.createdAfter = &slot
	}
}

func (i *Indexer) AnchorByID(anchorID iotago.AnchorID) *IndexerResult {
	query := i.db.Model(&anchor{}).
		Where("anchor_id = ?", anchorID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) anchorQueryWithFilter(opts *AnchorFilterOptions) *gorm.DB {
	query := i.db.Model(&anchor{}).Where("deleted_at_slot = 0")

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
		query = query.Where("created_at_slot < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at_slot > ?", *opts.createdAfter)
	}

	return query
}

func (i *Indexer) Anchor(filters ...options.Option[AnchorFilterOptions]) *IndexerResult {
	opts := options.Apply(&AnchorFilterOptions{
		pageSize: DefaultPageSize,
	}, filters)
	query := i.anchorQueryWithFilter(opts)

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
