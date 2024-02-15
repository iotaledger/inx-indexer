package server

import (
	"github.com/labstack/echo/v4"

	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-indexer/pkg/indexer"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	APIRoute = "/api/indexer/v2"
)

type IndexerServer struct {
	Indexer                 *indexer.Indexer
	NodeBridge              nodebridge.NodeBridge
	RestAPILimitsMaxResults int

	APIProvider iotago.APIProvider
	Bech32HRP   iotago.NetworkPrefix
}

func NewIndexerServer(indexer *indexer.Indexer, echo *echo.Echo, nodeBridge nodebridge.NodeBridge, maxPageSize int) *IndexerServer {
	s := &IndexerServer{
		Indexer:                 indexer,
		NodeBridge:              nodeBridge,
		RestAPILimitsMaxResults: maxPageSize,
		APIProvider:             nodeBridge.APIProvider(),
		Bech32HRP:               nodeBridge.APIProvider().CommittedAPI().ProtocolParameters().Bech32HRP(),
	}
	s.configureRoutes(echo.Group(APIRoute))

	return s
}
