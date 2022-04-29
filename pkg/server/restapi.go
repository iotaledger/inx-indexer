package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	iotago "github.com/iotaledger/iota.go/v3"
)

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

	// QueryParameterHasStorageReturnCondition is used to filter for outputs having a storage deposit return unlock condition.
	QueryParameterHasStorageReturnCondition = "hasStorageReturnCondition"

	// QueryParameterStorageReturnAddress is used to filter for outputs with a certain storage deposit return address.
	QueryParameterStorageReturnAddress = "storageReturnAddress"

	// QueryParameterHasExpirationCondition is used to filter for outputs having an expiration unlock condition.
	QueryParameterHasExpirationCondition = "hasExpirationCondition"

	// QueryParameterExpiresBefore is used to filter for outputs that expire before a certain unix time.
	QueryParameterExpiresBefore = "expiresBefore"

	// QueryParameterExpiresAfter is used to filter for outputs that expire after a certain unix time.
	QueryParameterExpiresAfter = "expiresAfter"

	// QueryParameterExpiresBeforeMilestone is used to filter for outputs that expire before a certain milestone index.
	QueryParameterExpiresBeforeMilestone = "expiresBeforeMilestone"

	// QueryParameterExpiresAfterMilestone is used to filter for outputs that expire after a certain milestone index.
	QueryParameterExpiresAfterMilestone = "expiresAfterMilestone"

	// QueryParameterExpirationReturnAddress is used to filter for outputs with a certain expiration return address.
	QueryParameterExpirationReturnAddress = "expirationReturnAddress"

	// QueryParameterHasTimelockCondition is used to filter for outputs having a timelock unlock condition.
	QueryParameterHasTimelockCondition = "hasTimelockCondition"

	// QueryParameterTimelockedBefore is used to filter for outputs that are timelocked before a certain unix time.
	QueryParameterTimelockedBefore = "timelockedBefore"

	// QueryParameterTimelockedAfter is used to filter for outputs that are timelocked after a certain unix time.
	QueryParameterTimelockedAfter = "timelockedAfter"

	// QueryParameterTimelockedBeforeMilestone is used to filter for outputs that are timelocked before a certain milestone index.
	QueryParameterTimelockedBeforeMilestone = "timelockedBeforeMilestone"

	// QueryParameterTimelockedAfterMilestone is used to filter for outputs that are timelocked after a certain milestone index.
	QueryParameterTimelockedAfterMilestone = "timelockedAfterMilestone"

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

var (
	// ErrInvalidParameter defines the invalid parameter error.
	ErrInvalidParameter = echo.NewHTTPError(http.StatusBadRequest, "invalid parameter")
)

// HTTPErrorResponse defines the error struct for the HTTPErrorResponseEnvelope.
type HTTPErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorResponseEnvelope defines the error response schema for node API responses.
type HTTPErrorResponseEnvelope struct {
	Error HTTPErrorResponse `json:"error"`
}

func ErrorHandler() func(error, echo.Context) {
	return func(err error, c echo.Context) {

		var statusCode int
		var message string

		var e *echo.HTTPError
		if errors.As(err, &e) {
			statusCode = e.Code
			message = fmt.Sprintf("%s, error: %s", e.Message, err)
		} else {
			statusCode = http.StatusInternalServerError
			message = fmt.Sprintf("internal server error. error: %s", err)
		}

		_ = c.JSON(statusCode, HTTPErrorResponseEnvelope{Error: HTTPErrorResponse{Code: strconv.Itoa(statusCode), Message: message}})
	}
}

func ParseAliasIDParam(c echo.Context) (*iotago.AliasID, error) {
	aliasIDParam := strings.ToLower(c.Param(ParameterAliasID))

	aliasIDBytes, err := iotago.DecodeHex(aliasIDParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid alias ID: %s, error: %s", aliasIDParam, err)
	}

	if len(aliasIDBytes) != iotago.AliasIDLength {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid alias ID: %s, error: %s", aliasIDParam, err)
	}

	var aliasID iotago.AliasID
	copy(aliasID[:], aliasIDBytes)
	return &aliasID, nil
}

func ParseNFTIDParam(c echo.Context) (*iotago.NFTID, error) {
	nftIDParam := strings.ToLower(c.Param(ParameterNFTID))

	nftIDBytes, err := iotago.DecodeHex(nftIDParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid NFT ID: %s, error: %s", nftIDParam, err)
	}

	if len(nftIDBytes) != iotago.NFTIDLength {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid NFT ID: %s, error: %s", nftIDParam, err)
	}

	var nftID iotago.NFTID
	copy(nftID[:], nftIDBytes)
	return &nftID, nil
}

func ParseFoundryIDParam(c echo.Context) (*iotago.FoundryID, error) {
	foundryIDParam := strings.ToLower(c.Param(ParameterFoundryID))

	foundryIDBytes, err := iotago.DecodeHex(foundryIDParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid foundry ID: %s, error: %s", foundryIDParam, err)
	}

	if len(foundryIDBytes) != iotago.FoundryIDLength {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid foundry ID: %s, error: %s", foundryIDParam, err)
	}

	var foundryID iotago.FoundryID
	copy(foundryID[:], foundryIDBytes)
	return &foundryID, nil
}

func ParseUint32QueryParam(c echo.Context, paramName string, maxValue ...uint32) (uint32, error) {
	intString := strings.ToLower(c.QueryParam(paramName))
	if intString == "" {
		return 0, errors.WithMessagef(ErrInvalidParameter, "parameter \"%s\" not specified", paramName)
	}

	value, err := strconv.ParseUint(intString, 10, 32)
	if err != nil {
		return 0, errors.WithMessagef(ErrInvalidParameter, "invalid value: %s, error: %s", intString, err)
	}

	if len(maxValue) > 0 {
		if uint32(value) > maxValue[0] {
			return 0, errors.WithMessagef(ErrInvalidParameter, "invalid value: %s, higher than the max number %d", intString, maxValue)
		}
	}
	return uint32(value), nil
}

func ParseUnixTimestampQueryParam(c echo.Context, paramName string) (time.Time, error) {
	timestamp, err := ParseUint32QueryParam(c, paramName)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(timestamp), 0), nil
}

func ParseBech32AddressQueryParam(c echo.Context, prefix iotago.NetworkPrefix, paramName string) (iotago.Address, error) {
	addressParam := strings.ToLower(c.QueryParam(paramName))

	hrp, bech32Address, err := iotago.ParseBech32(addressParam)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid address: %s, error: %s", addressParam, err)
	}

	if hrp != prefix {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid bech32 address, expected prefix: %s", prefix)
	}

	return bech32Address, nil
}

func ParseHexQueryParam(c echo.Context, paramName string, maxLen int) ([]byte, error) {
	param := c.QueryParam(paramName)

	paramBytes, err := iotago.DecodeHex(param)
	if err != nil {
		return nil, errors.WithMessagef(ErrInvalidParameter, "invalid param: %s, error: %s", paramName, err)
	}
	if len(paramBytes) > maxLen {
		return nil, errors.WithMessage(ErrInvalidParameter, fmt.Sprintf("query parameter %s too long, max. %d bytes but is %d", paramName, maxLen, len(paramBytes)))
	}
	return paramBytes, nil
}

func ParseBoolQueryParam(c echo.Context, paramName string) (bool, error) {
	return strconv.ParseBool(c.QueryParam(paramName))
}
