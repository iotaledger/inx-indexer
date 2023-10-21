package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/nodeclient/apimodels"
)

const (
	// RouteOutputs is the route for getting basic, foundry, account, delegation and nft outputs filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "hasNativeToken", "nativeToken", "unlockableByAddress", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputs = "/outputs"

	// RouteOutputsBasic is the route for getting basic outputs filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "hasNativeToken", "nativeToken", "address", "unlockableByAddress", "hasStorageDepositReturn", "storageDepositReturnAddress",
	// 					 "hasExpiration", "expiresBefore", "expiresAfter", "expirationReturnAddress",
	//					 "hasTimelock", "timelockedBefore", "timelockedAfter", "sender", "tag",
	//					 "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsBasic = "/outputs/basic"

	// RouteOutputsAccounts is the route for getting accounts filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "unlockableByAddress", "stateController", "governor", "issuer", "sender",
	//					 "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsAccounts = "/outputs/account"

	// RouteOutputsAccountByID is the route for getting accounts by their accountID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsAccountByID = "/outputs/account/:" + ParameterAccountID

	// RouteOutputsNFTs is the route for getting NFT filtered by the given parameters.
	// Query parameters: "address", "unlockableByAddress", "hasStorageDepositReturn", "storageDepositReturnAddress",
	// 					 "hasExpiration", "expiresBefore", "expiresAfter", "expirationReturnAddress",
	//					 "hasTimelock", "timelockedBefore", "timelockedAfter", "issuer", "sender", "tag",
	//					 "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsNFTs = "/outputs/nft"

	// RouteOutputsNFTByID is the route for getting NFT by their nftID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsNFTByID = "/outputs/nft/:" + ParameterNFTID

	// RouteOutputsFoundries is the route for getting foundries filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "hasNativeToken", "nativeToken", "account", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsFoundries = "/outputs/foundry"

	// RouteOutputsFoundryByID is the route for getting foundries by their foundryID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsFoundryByID = "/outputs/foundry/:" + ParameterFoundryID

	// RouteOutputsDelegations is the route for getting delegations filtered by the given parameters.
	// GET with query parameter returns all outputIDs that fit these filter criteria.
	// Query parameters: "address", "validator", "createdBefore", "createdAfter"
	// Returns an empty list if no results are found.
	RouteOutputsDelegations = "/outputs/delegation"

	// RouteOutputsDelegationByID is the route for getting delegations by their delegationID.
	// GET returns the outputIDs or 404 if no record is found.
	RouteOutputsDelegationByID = "/outputs/delegation/:" + ParameterDelegationID

	RouteMultiAddressByAddress = "/multiaddress/:" + ParameterAddress
)

const (
	MaxTagLength = 64
)

func (s *IndexerServer) configureRoutes(routeGroup *echo.Group) {

	routeGroup.GET(RouteOutputs, func(c echo.Context) error {
		resp, err := s.combinedOutputsWithFilter(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsBasic, func(c echo.Context) error {
		resp, err := s.basicOutputsWithFilter(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsAccounts, func(c echo.Context) error {
		resp, err := s.accountsWithFilter(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsAccountByID, func(c echo.Context) error {
		resp, err := s.accountByID(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsNFTs, func(c echo.Context) error {
		resp, err := s.nftsWithFilter(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsNFTByID, func(c echo.Context) error {
		resp, err := s.nftByID(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsFoundries, func(c echo.Context) error {
		resp, err := s.foundriesWithFilter(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsFoundryByID, func(c echo.Context) error {
		resp, err := s.foundryByID(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsDelegations, func(c echo.Context) error {
		resp, err := s.delegationsWithFilter(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteOutputsDelegationByID, func(c echo.Context) error {
		resp, err := s.delegationByID(c)
		if err != nil {
			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), resp)
	})

	routeGroup.GET(RouteMultiAddressByAddress, s.multiAddressByAddress)
}

func (s *IndexerServer) combinedOutputsWithFilter(c echo.Context) (*apimodels.IndexerResponse, error) {
	filters := []options.Option[indexer.CombinedFilterOptions]{indexer.CombinedPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeToken)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasNativeToken)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.CombinedHasNativeToken(value))
	}

	if len(c.QueryParam(QueryParameterNativeToken)) > 0 {
		value, err := httpserver.ParseHexQueryParam(c, QueryParameterNativeToken, iotago.NativeTokenIDLength)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.CombinedNativeToken(iotago.NativeTokenID(value)))
	}

	if len(c.QueryParam(QueryParameterUnlockableByAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterUnlockableByAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.CombinedUnlockableByAddress(addr))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.CombinedCursor(cursor), indexer.CombinedPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.CombinedCreatedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.CombinedCreatedAfter(slot))
	}

	return indexerResponseFromResult(s.Indexer.Combined(filters...))
}

func (s *IndexerServer) basicOutputsWithFilter(c echo.Context) (*apimodels.IndexerResponse, error) {
	filters := []options.Option[indexer.BasicFilterOptions]{indexer.BasicPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeToken)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasNativeToken)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicHasNativeToken(value))
	}

	if len(c.QueryParam(QueryParameterNativeToken)) > 0 {
		value, err := httpserver.ParseHexQueryParam(c, QueryParameterNativeToken, iotago.NativeTokenIDLength)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicNativeToken(iotago.NativeTokenID(value)))
	}

	if len(c.QueryParam(QueryParameterUnlockableByAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterUnlockableByAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicUnlockableByAddress(addr))
	}

	if len(c.QueryParam(QueryParameterAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicUnlockAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasStorageDepositReturn)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasStorageDepositReturn)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicHasStorageDepositReturnCondition(value))
	}

	if len(c.QueryParam(QueryParameterStorageDepositReturnAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterStorageDepositReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicStorageDepositReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasExpiration)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasExpiration)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicHasExpirationCondition(value))
	}

	if len(c.QueryParam(QueryParameterExpirationReturnAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterExpirationReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicExpirationReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterExpiresBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterExpiresBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicExpiresBefore(slot))
	}

	if len(c.QueryParam(QueryParameterExpiresAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterExpiresAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicExpiresAfter(slot))
	}

	if len(c.QueryParam(QueryParameterHasTimelock)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasTimelock)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicHasTimelockCondition(value))
	}

	if len(c.QueryParam(QueryParameterTimelockedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterTimelockedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicTimelockedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterTimelockedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterTimelockedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicTimelockedAfter(slot))
	}

	if len(c.QueryParam(QueryParameterSender)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterSender)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicSender(addr))
	}

	if len(c.QueryParam(QueryParameterTag)) > 0 {
		tagBytes, err := httpserver.ParseHexQueryParam(c, QueryParameterTag, MaxTagLength)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicTag(tagBytes))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicCursor(cursor), indexer.BasicPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicCreatedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.BasicCreatedAfter(slot))
	}

	return indexerResponseFromResult(s.Indexer.Basic(filters...))
}

func (s *IndexerServer) accountByID(c echo.Context) (*apimodels.IndexerResponse, error) {
	accountID, err := httpserver.ParseAccountIDParam(c, ParameterAccountID)
	if err != nil {
		return nil, err
	}

	return singleOutputResponseFromResult(s.Indexer.AccountByID(accountID))
}

func (s *IndexerServer) accountsWithFilter(c echo.Context) (*apimodels.IndexerResponse, error) {
	filters := []options.Option[indexer.AccountFilterOptions]{indexer.AccountPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterUnlockableByAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterUnlockableByAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountUnlockableByAddress(addr))
	}

	if len(c.QueryParam(QueryParameterStateController)) > 0 {
		stateController, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterStateController)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountStateController(stateController))
	}

	if len(c.QueryParam(QueryParameterGovernor)) > 0 {
		governor, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterGovernor)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountGovernor(governor))
	}

	if len(c.QueryParam(QueryParameterIssuer)) > 0 {
		issuer, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterIssuer)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountIssuer(issuer))
	}

	if len(c.QueryParam(QueryParameterSender)) > 0 {
		sender, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterSender)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountSender(sender))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountCursor(cursor), indexer.AccountPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountCreatedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.AccountCreatedAfter(slot))
	}

	return indexerResponseFromResult(s.Indexer.Account(filters...))
}

func (s *IndexerServer) nftByID(c echo.Context) (*apimodels.IndexerResponse, error) {
	nftID, err := httpserver.ParseNFTIDParam(c, ParameterNFTID)
	if err != nil {
		return nil, err
	}

	return singleOutputResponseFromResult(s.Indexer.NFTByID(nftID))
}

func (s *IndexerServer) nftsWithFilter(c echo.Context) (*apimodels.IndexerResponse, error) {
	filters := []options.Option[indexer.NFTFilterOptions]{indexer.NFTPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterUnlockableByAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterUnlockableByAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTUnlockableByAddress(addr))
	}

	if len(c.QueryParam(QueryParameterAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTUnlockAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasStorageDepositReturn)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasStorageDepositReturn)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasStorageDepositReturnCondition(value))
	}

	if len(c.QueryParam(QueryParameterStorageDepositReturnAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterStorageDepositReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTStorageDepositReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterHasExpiration)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasExpiration)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasExpirationCondition(value))
	}

	if len(c.QueryParam(QueryParameterExpirationReturnAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterExpirationReturnAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpirationReturnAddress(addr))
	}

	if len(c.QueryParam(QueryParameterExpiresBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterExpiresBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpiresBefore(slot))
	}

	if len(c.QueryParam(QueryParameterExpiresAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterExpiresAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTExpiresAfter(slot))
	}

	if len(c.QueryParam(QueryParameterHasTimelock)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasTimelock)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTHasTimelockCondition(value))
	}

	if len(c.QueryParam(QueryParameterTimelockedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterTimelockedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTimelockedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterTimelockedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterTimelockedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTTimelockedAfter(slot))
	}

	if len(c.QueryParam(QueryParameterIssuer)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterIssuer)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTIssuer(addr))
	}

	if len(c.QueryParam(QueryParameterSender)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterSender)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTSender(addr))
	}

	if len(c.QueryParam(QueryParameterTag)) > 0 {
		tagBytes, err := httpserver.ParseHexQueryParam(c, QueryParameterTag, MaxTagLength)
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
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTCreatedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.NFTCreatedAfter(slot))
	}

	return indexerResponseFromResult(s.Indexer.NFT(filters...))
}

func (s *IndexerServer) foundryByID(c echo.Context) (*apimodels.IndexerResponse, error) {
	foundryID, err := httpserver.ParseFoundryIDParam(c, ParameterFoundryID)
	if err != nil {
		return nil, err
	}

	return singleOutputResponseFromResult(s.Indexer.FoundryByID(foundryID))
}

func (s *IndexerServer) foundriesWithFilter(c echo.Context) (*apimodels.IndexerResponse, error) {
	filters := []options.Option[indexer.FoundryFilterOptions]{indexer.FoundryPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterHasNativeToken)) > 0 {
		value, err := httpserver.ParseBoolQueryParam(c, QueryParameterHasNativeToken)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryHasNativeToken(value))
	}

	if len(c.QueryParam(QueryParameterNativeToken)) > 0 {
		value, err := httpserver.ParseHexQueryParam(c, QueryParameterNativeToken, iotago.NativeTokenIDLength)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryNativeToken(iotago.NativeTokenID(value)))
	}

	if len(c.QueryParam(QueryParameterAccount)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAccount)
		if err != nil {
			return nil, err
		}
		if addr.Type() != iotago.AddressAccount {
			return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid address: %s, not an account address", addr.Bech32(s.Bech32HRP))
		}

		//nolint:forcetypeassert // we already checked the type
		filters = append(filters, indexer.FoundryWithAccountAddress(addr.(*iotago.AccountAddress)))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryCursor(cursor), indexer.FoundryPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryCreatedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.FoundryCreatedAfter(slot))
	}

	return indexerResponseFromResult(s.Indexer.Foundry(filters...))
}

func (s *IndexerServer) delegationByID(c echo.Context) (*apimodels.IndexerResponse, error) {
	delegationID, err := httpserver.ParseDelegationIDParam(c, ParameterDelegationID)
	if err != nil {
		return nil, err
	}

	return singleOutputResponseFromResult(s.Indexer.DelegationByID(delegationID))
}

func (s *IndexerServer) delegationsWithFilter(c echo.Context) (*apimodels.IndexerResponse, error) {
	filters := []options.Option[indexer.DelegationFilterOptions]{indexer.DelegationPageSize(s.pageSizeFromContext(c))}

	if len(c.QueryParam(QueryParameterAddress)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterAddress)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.DelegationAddress(addr))
	}

	if len(c.QueryParam(QueryParameterValidator)) > 0 {
		addr, err := httpserver.ParseBech32AddressQueryParam(c, s.Bech32HRP, QueryParameterValidator)
		if err != nil {
			return nil, err
		}
		if addr.Type() != iotago.AddressAccount {
			return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid address: %s, not an account address", addr.Bech32(s.Bech32HRP))
		}

		//nolint:forcetypeassert // we already checked the type
		filters = append(filters, indexer.DelegationValidator(addr.(*iotago.AccountAddress)))
	}

	if len(c.QueryParam(QueryParameterCursor)) > 0 {
		cursor, pageSize, err := s.parseCursorQueryParameter(c)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.DelegationCursor(cursor), indexer.DelegationPageSize(pageSize))
	}

	if len(c.QueryParam(QueryParameterCreatedBefore)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedBefore)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.DelegationCreatedBefore(slot))
	}

	if len(c.QueryParam(QueryParameterCreatedAfter)) > 0 {
		slot, err := httpserver.ParseSlotQueryParam(c, QueryParameterCreatedAfter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, indexer.DelegationCreatedAfter(slot))
	}

	return indexerResponseFromResult(s.Indexer.Delegation(filters...))
}

func singleOutputResponseFromResult(result *indexer.IndexerResult) (*apimodels.IndexerResponse, error) {
	if result.Error != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading outputIDs failed: %s", result.Error)
	}
	if len(result.OutputIDs) == 0 {
		return nil, errors.WithMessage(echo.ErrNotFound, "record not found")
	}

	return indexerResponseFromResult(result)
}

func indexerResponseFromResult(result *indexer.IndexerResult) (*apimodels.IndexerResponse, error) {
	if result.Error != nil {
		return nil, errors.WithMessagef(echo.ErrInternalServerError, "reading outputIDs failed: %s", result.Error)
	}

	var cursor string
	if result.Cursor != nil {
		// Add the pageSize to the cursor we expose in the API
		cursor = fmt.Sprintf("%s.%d", *result.Cursor, result.PageSize)
	}

	return &apimodels.IndexerResponse{
		CommittedSlot: result.CommittedSlot,
		PageSize:      result.PageSize,
		Cursor:        cursor,
		Items:         result.OutputIDs.ToHex(),
	}, nil
}

func (s *IndexerServer) multiAddressByAddress(c echo.Context) error {
	address, err := httpserver.ParseBech32AddressParam(c, s.Bech32HRP, ParameterAddress)
	if err != nil {
		return err
	}

	if multiAddressRef, isMultiRef := address.(*iotago.MultiAddressReference); isMultiRef {
		multiAddress, err := s.Indexer.MultiAddressForReference(multiAddressRef)
		if err != nil {
			if ierrors.Is(err, indexer.ErrMultiAddressNotFound) {
				return echo.ErrNotFound
			}

			return err
		}

		return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), multiAddress)
	}

	if restrictedAddress, isRestricted := address.(*iotago.RestrictedAddress); isRestricted {
		if innerMultiAddressRef, isMultiRef := restrictedAddress.Address.(*iotago.MultiAddressReference); isMultiRef {
			multiAddress, err := s.Indexer.MultiAddressForReference(innerMultiAddressRef)
			if err != nil {
				if ierrors.Is(err, indexer.ErrMultiAddressNotFound) {
					return echo.ErrNotFound
				}

				return err
			}

			return httpserver.SendResponseByHeader(c, s.APIProvider.CommittedAPI(), &iotago.RestrictedAddress{
				Address:             multiAddress,
				AllowedCapabilities: restrictedAddress.AllowedCapabilities,
			})
		}
	}

	return echo.ErrNotFound
}

func (s *IndexerServer) parseCursorQueryParameter(c echo.Context) (string, uint32, error) {
	cursorWithPageSize := c.QueryParam(QueryParameterCursor)

	components := strings.Split(cursorWithPageSize, ".")
	if len(components) != 2 {
		return "", 0, errors.WithMessage(httpserver.ErrInvalidParameter, fmt.Sprintf("query parameter %s has wrong format", QueryParameterCursor))
	}

	if len(components[0]) != indexer.CursorLength {
		return "", 0, errors.WithMessage(httpserver.ErrInvalidParameter, fmt.Sprintf("query parameter %s has wrong format", QueryParameterCursor))
	}

	size, err := strconv.ParseUint(components[1], 10, 32)
	if err != nil {
		return "", 0, errors.WithMessage(httpserver.ErrInvalidParameter, fmt.Sprintf("query parameter %s has wrong format", QueryParameterCursor))
	}

	pageSize := uint32(size)
	if pageSize > uint32(s.RestAPILimitsMaxResults) {
		pageSize = uint32(s.RestAPILimitsMaxResults)
	}

	return components[0], pageSize, nil
}

func (s *IndexerServer) pageSizeFromContext(c echo.Context) uint32 {
	maxPageSize := uint32(s.RestAPILimitsMaxResults)
	if len(c.QueryParam(QueryParameterPageSize)) > 0 {
		pageSizeQueryParam, err := httpserver.ParseUint32QueryParam(c, QueryParameterPageSize, maxPageSize)
		if err != nil {
			return maxPageSize
		}

		if pageSizeQueryParam > 0 && pageSizeQueryParam < maxPageSize {
			// use the smaller page size given by the request
			maxPageSize = pageSizeQueryParam
		}
	}

	return maxPageSize
}
