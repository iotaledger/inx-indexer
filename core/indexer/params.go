package indexer

import (
	"github.com/iotaledger/hive.go/core/app"
)

type ParametersIndexer struct {
	Database struct {
		// Database engine (sqlite or postgres)
		Engine string `default:"sqlite" usage:"database engine (sqlite, postgresql)"`
		SQLite struct {
			// Path defines the path to the database folder
			Path string `default:"database" usage:"the path to the database folder"`
		} `name:"sqlite"`
		PostgreSQL struct {
			// Database name
			Database string `default:"indexer" usage:"database name"`

			// Database username
			Username string `default:"indexer" usage:"database username"`

			// Database password
			Password string `default:"" usage:"database password"`

			// Database host
			Host string `default:"localhost" usage:"database host"`

			// Database port
			Port uint `default:"5432" usage:"database port"`
		} `name:"postgresql"`
	} `name:"db"`
}

// ParametersRestAPI contains the definition of the parameters used by the Indexer HTTP server.
type ParametersRestAPI struct {
	// BindAddress defines the bind address on which the Indexer HTTP server listens.
	BindAddress string `default:"localhost:9091" usage:"the bind address on which the Indexer HTTP server listens"`

	// AdvertiseAddress defines the address of the Indexer HTTP server which is advertised to the INX Server (optional).
	AdvertiseAddress string `default:"" usage:"the address of the Indexer HTTP server which is advertised to the INX Server (optional)"`

	// MaxPageSize defines the maximum number of results that may be returned for each page
	MaxPageSize int `default:"1000" usage:"the maximum number of results that may be returned for each page"`

	// DebugRequestLoggerEnabled defines whether the debug logging for requests should be enabled
	DebugRequestLoggerEnabled bool `default:"false" usage:"whether the debug logging for requests should be enabled"`
}

var ParamsIndexer = &ParametersIndexer{}
var ParamsRestAPI = &ParametersRestAPI{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"indexer": ParamsIndexer,
		"restAPI": ParamsRestAPI,
	},
	Masked: nil,
}
