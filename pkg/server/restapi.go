package server

const (
	// QueryParameterUnlockableByAddress is used to filter for all unlock conditions regarding a certain address.
	QueryParameterUnlockableByAddress = "unlockableByAddress"

	// QueryParameterAddress is used to filter for a certain address.
	QueryParameterAddress = "address"

	// QueryParameterAccount is used to filter for a certain account.
	QueryParameterAccount = "account"

	// QueryParameterIssuer is used to filter for a certain issuer.
	QueryParameterIssuer = "issuer"

	// QueryParameterSender is used to filter for a certain sender.
	QueryParameterSender = "sender"

	// QueryParameterTag is used to filter for a certain tag.
	QueryParameterTag = "tag"

	// QueryParameterValidator is used to filter for a certain validator.
	QueryParameterValidator = "validator"

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

	// QueryParameterTimelockedBefore is used to filter for outputs that are timelocked before a certain slot.
	QueryParameterTimelockedBefore = "timelockedBefore"

	// QueryParameterTimelockedAfter is used to filter for outputs that are timelocked after a certain slot.
	QueryParameterTimelockedAfter = "timelockedAfter"

	// QueryParameterStateController is used to filter for a certain state controller address.
	QueryParameterStateController = "stateController"

	// QueryParameterGovernor is used to filter for a certain governance controller address.
	QueryParameterGovernor = "governor"

	// QueryParameterPageSize is used to define the page size for the results.
	QueryParameterPageSize = "pageSize"

	// QueryParameterCursor is used to pass the offset we want to start the next results from.
	QueryParameterCursor = "cursor"

	// QueryParameterCreatedBefore is used to filter for outputs that were created before the given slot.
	QueryParameterCreatedBefore = "createdBefore"

	// QueryParameterCreatedAfter is used to filter for outputs that were created after the given slot.
	QueryParameterCreatedAfter = "createdAfter"

	// QueryParameterHasNativeToken is used to filter for outputs that have native tokens.
	QueryParameterHasNativeToken = "hasNativeToken"

	// QueryParameterNativeToken is used to filter for outputs that have a certain native token.
	QueryParameterNativeToken = "nativeToken"
)
