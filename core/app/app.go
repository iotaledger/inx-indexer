package app

import (
	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/core/shutdown"
	"github.com/iotaledger/hive.go/app/plugins/profiling"

	"github.com/iotaledger/inx-app/inx"
	"github.com/iotaledger/inx-indexer/core/indexer"
	"github.com/iotaledger/inx-indexer/plugins/prometheus"
)

var (
	// Name of the app.
	Name = "inx-indexer"

	// Version of the app.
	Version = "1.0.0-beta.5"
)

func App() *app.App {
	return app.New(Name, Version,
		app.WithInitComponent(InitComponent),
		app.WithCoreComponents([]*app.CoreComponent{
			inx.CoreComponent,
			indexer.CoreComponent,
			shutdown.CoreComponent,
		}...),
		app.WithPlugins([]*app.Plugin{
			profiling.Plugin,
			prometheus.Plugin,
		}...),
	)
}

var (
	InitComponent *app.InitComponent
)

func init() {
	InitComponent = &app.InitComponent{
		Component: &app.Component{
			Name: "App",
		},
		NonHiddenFlags: []string{
			"config",
			"help",
			"version",
		},
	}
}
