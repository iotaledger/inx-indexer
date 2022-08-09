package server

const (
	// ParameterFoundryID is used to identify a foundry by its ID.
	ParameterFoundryID = "foundryID"

	// ParameterAliasID is used to identify an alias by its ID.
	ParameterAliasID = "aliasID"

	// ParameterNFTID is used to identify a nft by its ID.
	ParameterNFTID = "nftID"

	// QueryParameterAddress is used to filter for a certain address.
	QueryParameterAddress = "address"

	// QueryParameterAliasAddress is used to filter for a certain alias address.
	QueryParameterAliasAddress = "aliasAddress"

	// QueryParameterIssuer is used to filter for a certain issuer.
	QueryParameterIssuer = "issuer"

	// QueryParameterSender is used to filter for a certain sender.
	QueryParameterSender = "sender"

	// QueryParameterTag is used to filter for a certain tag.
	QueryParameterTag = "tag"

	// QueryParameterHasStorageDepositReturn is used to filter for outputs having a storage deposit return unlock condition.
	QueryParameterHasStorageDepositReturn = "hasStorageDepositReturn"

	// QueryParameterStorageDepositReturnAddress is used to filter for outputs with a certain storage deposit return address.
	QueryParameterStorageDepositReturnAddress = "storageDepositReturnAddress"

	// QueryParameterHasExpiration is used to filter for outputs having an expiration unlock condition.
	QueryParameterHasExpiration = "hasExpiration"

	// QueryParameterExpiresBefore is used to filter for outputs that expire before a certain unix time.
	QueryParameterExpiresBefore = "expiresBefore"

	// QueryParameterExpiresAfter is used to filter for outputs that expire after a certain unix time.
	QueryParameterExpiresAfter = "expiresAfter"

	// QueryParameterExpirationReturnAddress is used to filter for outputs with a certain expiration return address.
	QueryParameterExpirationReturnAddress = "expirationReturnAddress"

	// QueryParameterHasTimelock is used to filter for outputs having a timelock unlock condition.
	QueryParameterHasTimelock = "hasTimelock"

	// QueryParameterTimelockedBefore is used to filter for outputs that are timelocked before a certain unix time.
	QueryParameterTimelockedBefore = "timelockedBefore"

	// QueryParameterTimelockedAfter is used to filter for outputs that are timelocked after a certain unix time.
	QueryParameterTimelockedAfter = "timelockedAfter"

	// QueryParameterStateController is used to filter for a certain state controller address.
	QueryParameterStateController = "stateController"

	// QueryParameterGovernor is used to filter for a certain governance controller address.
	QueryParameterGovernor = "governor"

	// QueryParameterPageSize is used to define the page size for the results.
	QueryParameterPageSize = "pageSize"

	// QueryParameterCursor is used to pass the offset we want to start the next results from.
	QueryParameterCursor = "cursor"

	// QueryParameterCreatedBefore is used to filter for outputs that were created before the given time.
	QueryParameterCreatedBefore = "createdBefore"

	// QueryParameterCreatedAfter is used to filter for outputs that were created after the given time.
	QueryParameterCreatedAfter = "createdAfter"

	// QueryParameterHasNativeTokens is used to filter for outputs that have native tokens.
	QueryParameterHasNativeTokens = "hasNativeTokens"

	// QueryParameterMinNativeTokenCount is used to filter for outputs that have at least an amount of native tokens.
	QueryParameterMinNativeTokenCount = "minNativeTokenCount"

	// QueryParameterMaxNativeTokenCount is used to filter for outputs that have at the most an amount of native tokens.
	QueryParameterMaxNativeTokenCount = "maxNativeTokenCount"
)
