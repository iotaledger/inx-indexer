package prometheus

import (
	"github.com/iotaledger/hive.go/core/app"
)

// ParametersPrometheus contains the definition of the parameters used by Prometheus.
type ParametersPrometheus struct {
	// Enabled defines whether the prometheus plugin is enabled.
	Enabled bool `default:"false" usage:"whether the prometheus plugin is enabled"`
	// BindAddress defines the bind address on which the Prometheus exporter listens on.
	BindAddress string `default:"localhost:9312" usage:"the bind address on which the Prometheus HTTP server listens on"`
	// GoMetrics defines whether to include go metrics.
	GoMetrics bool `default:"false" usage:"whether to include go metrics"`
	// ProcessMetrics defines whether to include process metrics.
	ProcessMetrics bool `default:"false" usage:"whether to include process metrics"`
	// RestAPIMetrics include restAPI metrics.
	RestAPIMetrics bool `default:"true" usage:"whether to include restAPI metrics"`
	// INXMetrics defines whether to include INXMetrics metrics.
	INXMetrics bool `name:"inxMetrics" default:"true" usage:"whether to include INX metrics"`
	// PromhttpMetrics defines whether to include promhttp metrics.
	PromhttpMetrics bool `default:"false" usage:"whether to include promhttp metrics"`
}

var ParamsPrometheus = &ParametersPrometheus{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"prometheus": ParamsPrometheus,
	},
	Masked: nil,
}
