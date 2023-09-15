package indexer

import (
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	iotago "github.com/iotaledger/iota.go/v3"
)

type alias struct {
	AliasID          []byte    `gorm:"primaryKey;notnull"`
	OutputID         []byte    `gorm:"unique;notnull"`
	NativeTokenCount uint32    `gorm:"notnull;type:integer"`
	StateController  []byte    `gorm:"notnull;index:alias_state_controller"`
	Governor         []byte    `gorm:"notnull;index:alias_governor"`
	Issuer           []byte    `gorm:"index:alias_issuer"`
	Sender           []byte    `gorm:"index:alias_sender"`
	CreatedAt        time.Time `gorm:"notnull;index:alias_created_at"`
}

func (o *alias) String() string {
	return fmt.Sprintf("alias output => AliasID: %s outputID: %s", hex.EncodeToString(o.AliasID), hex.EncodeToString(o.OutputID))
}

type AliasFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	unlockableByAddress *iotago.Address
	stateController     *iotago.Address
	governor            *iotago.Address
	issuer              *iotago.Address
	sender              *iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *time.Time
	createdAfter        *time.Time
}

type AliasFilterOption func(*AliasFilterOptions)

func AliasHasNativeTokens(value bool) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func AliasMinNativeTokenCount(value uint32) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func AliasMaxNativeTokenCount(value uint32) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func AliasUnlockableByAddress(address iotago.Address) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.unlockableByAddress = &address
	}
}

func AliasStateController(address iotago.Address) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.stateController = &address
	}
}

func AliasGovernor(address iotago.Address) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.governor = &address
	}
}

func AliasSender(address iotago.Address) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.sender = &address
	}
}

func AliasIssuer(address iotago.Address) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.issuer = &address
	}
}

func AliasPageSize(pageSize uint32) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.pageSize = pageSize
	}
}

func AliasCursor(cursor string) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.cursor = &cursor
	}
}

func AliasCreatedBefore(time time.Time) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.createdBefore = &time
	}
}

func AliasCreatedAfter(time time.Time) AliasFilterOption {
	return func(args *AliasFilterOptions) {
		args.createdAfter = &time
	}
}

func aliasFilterOptions(optionalOptions []AliasFilterOption) *AliasFilterOptions {
	result := &AliasFilterOptions{}

	for _, optionalOption := range optionalOptions {
		optionalOption(result)
	}

	return result
}

func (i *Indexer) AliasOutput(aliasID *iotago.AliasID) *IndexerResult {
	query := i.db.Model(&alias{}).
		Where("alias_id = ?", aliasID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) aliasQueryWithFilter(opts *AliasFilterOptions) (*gorm.DB, error) {
	query := i.db.Model(&alias{})

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
		query = query.Where("(state_controller = ? OR governor = ?)", addr[:], addr[:])
	}

	if opts.stateController != nil {
		addr, err := addressBytesForAddress(*opts.stateController)
		if err != nil {
			return nil, err
		}
		query = query.Where("state_controller = ?", addr[:])
	}

	if opts.governor != nil {
		addr, err := addressBytesForAddress(*opts.governor)
		if err != nil {
			return nil, err
		}
		query = query.Where("governor = ?", addr[:])
	}

	if opts.sender != nil {
		addr, err := addressBytesForAddress(*opts.sender)
		if err != nil {
			return nil, err
		}
		query = query.Where("sender = ?", addr[:])
	}

	if opts.issuer != nil {
		addr, err := addressBytesForAddress(*opts.issuer)
		if err != nil {
			return nil, err
		}
		query = query.Where("issuer = ?", addr[:])
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return query, nil
}

func (i *Indexer) AliasOutputsWithFilters(filter ...AliasFilterOption) *IndexerResult {
	opts := aliasFilterOptions(filter)
	query, err := i.aliasQueryWithFilter(opts)
	if err != nil {
		return errorResult(err)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
