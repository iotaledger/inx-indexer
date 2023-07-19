package indexer

import (
	iotago "github.com/iotaledger/iota.go/v4"
)

type account struct {
	AccountID        accountIDBytes   `gorm:"primaryKey;notnull"`
	OutputID         outputIDBytes    `gorm:"unique;notnull"`
	NativeTokenCount uint32           `gorm:"notnull;type:integer"`
	StateController  addressBytes     `gorm:"notnull;index:account_state_controller"`
	Governor         addressBytes     `gorm:"notnull;index:account_governor"`
	Issuer           addressBytes     `gorm:"index:account_issuer"`
	Sender           addressBytes     `gorm:"index:account_sender"`
	CreatedAt        iotago.SlotIndex `gorm:"notnull;index:alias_created_at"`
}

type AccountFilterOptions struct {
	hasNativeTokens     *bool
	minNativeTokenCount *uint32
	maxNativeTokenCount *uint32
	stateController     *iotago.Address
	governor            *iotago.Address
	issuer              *iotago.Address
	sender              *iotago.Address
	pageSize            uint32
	cursor              *string
	createdBefore       *iotago.SlotIndex
	createdAfter        *iotago.SlotIndex
}

type AccountFilterOption func(*AccountFilterOptions)

func AccountHasNativeTokens(value bool) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.hasNativeTokens = &value
	}
}

func AccountMinNativeTokenCount(value uint32) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.minNativeTokenCount = &value
	}
}

func AccountMaxNativeTokenCount(value uint32) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.maxNativeTokenCount = &value
	}
}

func AccountStateController(address iotago.Address) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.stateController = &address
	}
}

func AccountGovernor(address iotago.Address) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.governor = &address
	}
}

func AccountSender(address iotago.Address) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.sender = &address
	}
}

func AccountIssuer(address iotago.Address) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.issuer = &address
	}
}

func AccountPageSize(pageSize uint32) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.pageSize = pageSize
	}
}

func AccountCursor(cursor string) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.cursor = &cursor
	}
}

func AccountCreatedBefore(slot iotago.SlotIndex) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.createdBefore = &slot
	}
}

func AccountCreatedAfter(slot iotago.SlotIndex) AccountFilterOption {
	return func(args *AccountFilterOptions) {
		args.createdAfter = &slot
	}
}

func accountFilterOptions(optionalOptions []AccountFilterOption) *AccountFilterOptions {
	result := &AccountFilterOptions{}

	for _, optionalOption := range optionalOptions {
		optionalOption(result)
	}

	return result
}

func (i *Indexer) AccountOutput(accountID *iotago.AccountID) *IndexerResult {
	query := i.db.Model(&account{}).
		Where("account_id = ?", accountID[:]).
		Limit(1)

	return i.combineOutputIDFilteredQuery(query, 0, nil)
}

func (i *Indexer) AccountOutputsWithFilters(filter ...AccountFilterOption) *IndexerResult {
	opts := accountFilterOptions(filter)
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

	if opts.stateController != nil {
		addr, err := addressBytesForAddress(*opts.stateController)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("state_controller = ?", addr[:])
	}

	if opts.governor != nil {
		addr, err := addressBytesForAddress(*opts.governor)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("governor = ?", addr[:])
	}

	if opts.sender != nil {
		addr, err := addressBytesForAddress(*opts.sender)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("sender = ?", addr[:])
	}

	if opts.issuer != nil {
		addr, err := addressBytesForAddress(*opts.issuer)
		if err != nil {
			return errorResult(err)
		}
		query = query.Where("issuer = ?", addr[:])
	}

	if opts.createdBefore != nil {
		query = query.Where("created_at < ?", *opts.createdBefore)
	}

	if opts.createdAfter != nil {
		query = query.Where("created_at > ?", *opts.createdAfter)
	}

	return i.combineOutputIDFilteredQuery(query, opts.pageSize, opts.cursor)
}
