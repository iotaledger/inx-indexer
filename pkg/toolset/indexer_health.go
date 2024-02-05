package toolset

import (
	"context"
	"fmt"
	"net/http"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/iotaledger/hive.go/app/configuration"
	"github.com/iotaledger/iota.go/v4/api"
)

func checkHealth(args []string) error {
	fs := configuration.NewUnsortedFlagSet("", flag.ContinueOnError)
	nodeURLFlag := fs.String(FlagToolIndexerURL, "http://localhost:9091", "URL of the indexer (optional)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", ToolIndexerHealth)
		fs.PrintDefaults()
		println(fmt.Sprintf("\nexample: %s --%s %s",
			ToolIndexerHealth,
			FlagToolIndexerURL,
			"http://192.168.1.221:9091",
		))
	}

	if err := parseFlagSet(fs, args); err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s", *nodeURLFlag, api.RouteHealth)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("IsHealthy: %s\n", yesOrNo(resp.StatusCode == http.StatusOK))

	return nil
}
