package indexer

import (
	"github.com/iotaledger/hive.go/core/app"
)

type ParametersIndexer struct {
	Database struct {
		// Path defines the path to the database folder
		Path string `default:"database" usage:"the path to the database folder"`
	} `name:"db"`

	// BindAddress defines the bind address on which the Indexer HTTP server listens.
	BindAddress string `default:"localhost:9091" usage:"the bind address on which the Indexer HTTP server listens"`

	// MaxPageSize defines the maximum number of results that may be returned for each page
	MaxPageSize int `default:"1000" usage:"the maximum number of results that may be returned for each page"`

	// DebugRequestLoggerEnabled defines whether the debug logging for requests should be enabled
	DebugRequestLoggerEnabled bool `default:"false" usage:"whether the debug logging for requests should be enabled"`
}

var ParamsIndexer = &ParametersIndexer{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"indexer": ParamsIndexer,
	},
	Masked: nil,
}
