package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/gohornet/inx-indexer/indexer"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (

	// RouteOutputsBasic is the route for getting basic outputs filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "address", "hasStorageReturnCondition", "storageReturnAddress", "hasExpirationCondition",
	//					 "expiresBefore", "expiresAfter", "expiresBeforeMilestone", "expiresAfterMilestone",
	//					 "hasTimelockCondition", "timelockedBefore", "timelockedAfter", "timelockedBeforeMilestone",
	//					 "timelockedAfterMilestone", "sender", "tag", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsBasic = "/outputs/basic"

	// RouteOutputsAliases is the route for getting aliases filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "stateController", "governor", "issuer", "sender", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsAliases = "/outputs/alias"

	// RouteOutputsAliasByID is the route for getting aliases by their aliasID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsAliasByID = "/outputs/alias/:" + ParameterAliasID

	// RouteOutputsNFTs is the route for getting NFT filtered by the given parameters.
	// Query parameters: "address", "hasStorageReturnCondition", "storageReturnAddress", "hasExpirationCondition",
	//					 "expiresBefore", "expiresAfter", "expiresBeforeMilestone", "expiresAfterMilestone",
	//					 "hasTimelockCondition", "timelockedBefore", "timelockedAfter", "timelockedBeforeMilestone",
	//					 "timelockedAfterMilestone", "issuer", "sender", "tag", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsNFTs = "/outputs/nft"

	// RouteOutputsNFTByID is the route for getting NFT by their nftID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsNFTByID = "/outputs/nft/:" + ParameterNFTID

	// RouteOutputsFoundries is the route for getting foundries filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "aliasAddress", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsFoundries = "/outputs/foundry"

	// RouteOutputsFoundryByID is the route for getting foundries by their foundryID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsFoundryByID = "/outputs/foundry/:" + ParameterFoundryID
)

func (s *IndexerServer) configureRoutes(routeGroup *echo.Group) {

	routeGroup.GET(RouteOutputsBasic, func(c echo.Context) error {
		resp, err := s.basicOutputsWithFilter(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutputsAliases, func(c echo.Context) error {
		resp, err := s.aliasesWithFilter(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutputsAliasByID, func(c echo.Context) error {
		resp, err := s.aliasByID(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutputsNFTs, func(c echo.Context) error {
		resp, err := s.nftsWithFilter(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutputsNFTByID, func(c echo.Context) error {
		resp, err := s.nftByID(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutputsFoundries, func(c echo.Context) error {
		resp, err := s.foundriesWithFilter(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})

	routeGroup.GET(RouteOutputsFoundryByID, func(c echo.Context) error {
		resp, err := s.foundryByID(c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	})
}

func (s *IndexerServer) basicOutputsWithFilter(c echo.Context) (*outputsResponse, error) {
	filters := []indexer.BasicOutputFilterOption{indexer.BasicOutputPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeTokens)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasNativeTokens)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputHasNativeTokens(value))
	}

	if len(c.QueryParam(QueryParameterMinNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMinNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputMinNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterMaxNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMaxNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputMaxNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterAddress)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputUnlockableByAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasStorageReturnCondition)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasStorageReturnCondition)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputHasStorageDepositReturnCondition(value))
	}

	if len(c.QueryParam(QueryParameterStorageReturnAddress)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterStorageReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputStorageDepositReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasExpirationCondition)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasExpirationCondition)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputHasExpirationCondition(value))
	}

	if len(c.QueryParam(QueryParameterExpirationReturnAddress)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterExpirationReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputExpirationReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterExpiresBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterExpiresBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputExpiresBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterExpiresAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterExpiresAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputExpiresAfter(timestamp))
	}

	if len(c.QueryParam(QueryParameterExpiresBeforeMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterExpiresBeforeMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputExpiresBeforeMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterExpiresAfterMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterExpiresAfterMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputExpiresAfterMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterHasTimelockCondition)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasTimelockCondition)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputHasTimelockCondition(value))
	}

	if len(c.QueryParam(QueryParameterTimelockedBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterTimelockedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputTimelockedBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterTimelockedAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterTimelockedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputTimelockedAfter(timestamp))
	}

	if len(c.QueryParam(QueryParameterTimelockedBeforeMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterTimelockedBeforeMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputTimelockedBeforeMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterTimelockedAfterMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterTimelockedAfterMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputTimelockedAfterMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterSender)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterSender)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputSender(addr))
	}

	if len(c.QueryParam(QueryParameterTag)) > 0 {
		tagBytes, err := ParseHexQueryParam(c, QueryParameterTag, iotago.MaxTagLength)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputTag(tagBytes))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputCursor(cursor), indexer.BasicOutputPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputCreatedBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicOutputCreatedAfter(timestamp))
	}

	return outputsResponseFromResult(s.Indexer.BasicOutputsWithFilters(filters...))
}

func (s *IndexerServer) aliasByID(c echo.Context) (*outputsResponse, error) {
	aliasID, err := ParseAliasIDParam(c)
	if err != nil {
		return nil, err
	}
	return singleOutputResponseFromResult(s.Indexer.AliasOutput(aliasID))
}

func (s *IndexerServer) aliasesWithFilter(c echo.Context) (*outputsResponse, error) {
	filters := []indexer.AliasFilterOption{indexer.AliasPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeTokens)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasNativeTokens)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasHasNativeTokens(value))
	}

	if len(c.QueryParam(QueryParameterMinNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMinNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasMinNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterMaxNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMaxNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasMaxNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterStateController)) > 0 {
		stateController, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterStateController)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasStateController(stateController))
	}

	if len(c.QueryParam(QueryParameterGovernor)) > 0 {
		governor, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterGovernor)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasGovernor(governor))
	}

	if len(c.QueryParam(QueryParameterIssuer)) > 0 {
		issuer, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterIssuer)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasIssuer(issuer))
	}

	if len(c.QueryParam(QueryParameterSender)) > 0 {
		sender, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterSender)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasSender(sender))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasCursor(cursor), indexer.AliasPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasCreatedBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AliasCreatedAfter(timestamp))
	}

	return outputsResponseFromResult(s.Indexer.AliasOutputsWithFilters(filters...))
}

func (s *IndexerServer) nftByID(c echo.Context) (*outputsResponse, error) {
	nftID, err := ParseNFTIDParam(c)
	if err != nil {
		return nil, err
	}
	return singleOutputResponseFromResult(s.Indexer.NFTOutput(nftID))
}

func (s *IndexerServer) nftsWithFilter(c echo.Context) (*outputsResponse, error) {
	filters := []indexer.NFTFilterOption{indexer.NFTPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeTokens)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasNativeTokens)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasNativeTokens(value))
	}

	if len(c.QueryParam(QueryParameterMinNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMinNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTMinNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterMaxNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMaxNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTMaxNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterAddress)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTUnlockableByAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasStorageReturnCondition)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasStorageReturnCondition)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasStorageDepositReturnCondition(value))
	}

	if len(c.QueryParam(QueryParameterStorageReturnAddress)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterStorageReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTStorageDepositReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasExpirationCondition)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasExpirationCondition)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasExpirationCondition(value))
	}

	if len(c.QueryParam(QueryParameterExpirationReturnAddress)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterExpirationReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpirationReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterExpiresBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterExpiresBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpiresBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterExpiresAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterExpiresAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpiresAfter(timestamp))
	}

	if len(c.QueryParam(QueryParameterExpiresBeforeMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterExpiresBeforeMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpiresBeforeMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterExpiresAfterMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterExpiresAfterMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpiresAfterMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterHasTimelockCondition)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasTimelockCondition)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasTimelockCondition(value))
	}

	if len(c.QueryParam(QueryParameterTimelockedBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterTimelockedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTimelockedBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterTimelockedAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterTimelockedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTimelockedAfter(timestamp))
	}

	if len(c.QueryParam(QueryParameterTimelockedBeforeMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterTimelockedBeforeMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTimelockedBeforeMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterTimelockedAfterMilestone)) > 0 {
		msIndex, err := ParseUint32QueryParam(c, QueryParameterTimelockedAfterMilestone)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTimelockedAfterMilestone(msIndex))
	}

	if len(c.QueryParam(QueryParameterIssuer)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterIssuer)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTIssuer(addr))
	}

	if len(c.QueryParam(QueryParameterSender)) > 0 {
		addr, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterSender)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTSender(addr))
	}

	if len(c.QueryParam(QueryParameterTag)) > 0 {
		tagBytes, err := ParseHexQueryParam(c, QueryParameterTag, iotago.MaxTagLength)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTag(tagBytes))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTCursor(cursor), indexer.NFTPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTCreatedBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTCreatedAfter(timestamp))
	}

	return outputsResponseFromResult(s.Indexer.NFTOutputsWithFilters(filters...))
}

func (s *IndexerServer) foundryByID(c echo.Context) (*outputsResponse, error) {
	foundryID, err := ParseFoundryIDParam(c)
	if err != nil {
		return nil, err
	}
	return singleOutputResponseFromResult(s.Indexer.FoundryOutput(foundryID))
}

func (s *IndexerServer) foundriesWithFilter(c echo.Context) (*outputsResponse, error) {
	filters := []indexer.FoundryFilterOption{indexer.FoundryPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeTokens)) > 0 {
		value, err := ParseBoolQueryParam(c, QueryParameterHasNativeTokens)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryHasNativeTokens(value))
	}

	if len(c.QueryParam(QueryParameterMinNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMinNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryMinNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterMaxNativeTokenCount)) > 0 {
		value, err := ParseUint32QueryParam(c, QueryParameterMaxNativeTokenCount, iotago.MaxNativeTokenCountPerOutput)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryMaxNativeTokenCount(value))
	}

	if len(c.QueryParam(QueryParameterAliasAddress)) > 0 {
		address, err := ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAliasAddress)
		if err != nil {
			return nil, err
		}
		if address.Type() != iotago.AddressAlias {
			return nil, errors.WithMessagef(ErrInvalidParameter, "invalid address: %s, not an alias address", address.Bech32(s.Bech32HRP))
		}
		filters = append(filters, indexer.FoundryWithAliasAddress(address.(*iotago.AliasAddress)))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryCursor(cursor), indexer.FoundryPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryCreatedBefore(timestamp))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		timestamp, err := ParseUnixTimestampQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryCreatedAfter(timestamp))
	}

	return outputsResponseFromResult(s.Indexer.FoundryOutputsWithFilters(filters...))
}

func singleOutputResponseFromResult(result *indexer.IndexerResult) (*outputsResponse, error) {
	if result.Error != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading outputIDs failed: %s", result.Error)
	}
	if len(result.OutputIDs) == 0 {
		return nil, errors.WithMessage(echo.ErrNotFound, "record not found")
	}
	return outputsResponseFromResult(result)
}

func outputsResponseFromResult(result *indexer.IndexerResult) (*outputsResponse, error) {
	if result.Error != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading outputIDs failed: %s", result.Error)
	}

	var cursor *string
	if result.Cursor != nil {
		// Add the pageSize to the cursor we expose in the API
		cursorWithPageSize := fmt.Sprintf("%s.%d", *result.Cursor, result.PageSize)
		cursor = &cursorWithPageSize
	}

	return &outputsResponse{
		LedgerIndex: result.LedgerIndex,
		PageSize:    uint32(result.PageSize),
		Cursor:      cursor,
		Items:       result.OutputIDs.ToHex(),
	}, nil
}

func (s *IndexerServer) parseCursorQueryParameter(c echo.Context) (string, uint32, error) {
	cursorWithPageSize := c.QueryParam(QueryParameterCursor)

	components := strings.Split(cursorWithPageSize, ".")
	if len(components) != 2 {
		return "", 0, errors.WithMessage(ErrInvalidParameter, fmt.Sprintf("query parameter %s has wrong format", QueryParameterCursor))
	}

	if len(components[0]) != indexer.CursorLength {
		return "", 0, errors.WithMessage(ErrInvalidParameter, fmt.Sprintf("query parameter %s has wrong format", QueryParameterCursor))
	}

	size, err := strconv.ParseUint(components[1], 10, 32)
	if err != nil {
		return "", 0, errors.WithMessage(ErrInvalidParameter, fmt.Sprintf("query parameter %s has wrong format", QueryParameterCursor))
	}

	pageSize := uint32(size)
	if pageSize > uint32(s.RestAPILimitsMaxResults) {
		pageSize = uint32(s.RestAPILimitsMaxResults)
	}

	return components[0], pageSize, nil
}

func (s *IndexerServer) pageSizeFromContext(c echo.Context) uint32 {
	pageSize := uint32(s.RestAPILimitsMaxResults)
	if len(c.QueryParam(QueryParameterPageSize)) > 0 {
		i, err := ParseUint32QueryParam(c, QueryParameterPageSize, pageSize)
		if err != nil {
			return pageSize
		}
		pageSize = i
	}
	return pageSize
}
