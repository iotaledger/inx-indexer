package app

import (
	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/components/profiling"
	"github.com/iotaledger/hive.go/app/components/shutdown"
	"github.com/iotaledger/inx-app/components/inx"
	"github.com/iotaledger/inx-indexer/components/indexer"
	"github.com/iotaledger/inx-indexer/components/prometheus"
	"github.com/iotaledger/inx-indexer/pkg/toolset"
)

var (
	// Name of the app.
	Name = "inx-indexer"

	// Version of the app.
	Version = "v2.0.0-develop"
)

func App() *app.App {
	return app.New(Name, Version,
		app.WithInitComponent(InitComponent),
		app.WithComponents(
			inx.Component,
			indexer.Component,
			shutdown.Component,
			profiling.Component,
			prometheus.Component,
		),
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
		Init: initialize,
	}
}

func initialize(_ *app.App) error {
	if toolset.ShouldHandleTools() {
		toolset.HandleTools()
		// HandleTools will call os.Exit
	}

	return nil
}
