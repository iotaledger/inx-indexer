package server

import (
	"github.com/labstack/echo/v4"

	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	APIRoute = "/api/indexer/v2"
)

type IndexerServer struct {
	Indexer                 *indexer.Indexer
	APIProvider             iotago.APIProvider
	Bech32HRP               iotago.NetworkPrefix
	RestAPILimitsMaxResults int
}

func NewIndexerServer(indexer *indexer.Indexer, echo *echo.Echo, apiProvider iotago.APIProvider, maxPageSize int) *IndexerServer {
	s := &IndexerServer{
		Indexer:                 indexer,
		APIProvider:             apiProvider,
		Bech32HRP:               apiProvider.CommittedAPI().ProtocolParameters().Bech32HRP(),
		RestAPILimitsMaxResults: maxPageSize,
	}
	s.configureRoutes(echo.Group(APIRoute))

	return s
}
