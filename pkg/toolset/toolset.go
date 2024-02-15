package toolset

import (
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/iotaledger/hive.go/ierrors"
)

const (
	FlagToolIndexerURL = "indexerURL"
)

const (
	ToolIndexerHealth = "health"
)

// ShouldHandleTools checks if tools were requested.
func ShouldHandleTools() bool {
	args := os.Args[1:]

	for _, arg := range args {
		if strings.ToLower(arg) == "tool" || strings.ToLower(arg) == "tools" {
			return true
		}
	}

	return false
}

// HandleTools handles available tools.
func HandleTools() {

	args := os.Args[1:]
	if len(args) == 1 {
		listTools()
		os.Exit(1)
	}

	tools := map[string]func([]string) error{
		ToolIndexerHealth: checkHealth,
	}

	tool, exists := tools[strings.ToLower(args[1])]
	if !exists {
		fmt.Print("tool not found.\n\n")
		listTools()
		os.Exit(1)
	}

	if err := tool(args[2:]); err != nil {
		if ierrors.Is(err, flag.ErrHelp) {
			// help text was requested
			os.Exit(0)
		}

		fmt.Printf("\nerror: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func listTools() {
	fmt.Printf("%-20s queries the health endpoint of an indexer\n", fmt.Sprintf("%s:", ToolIndexerHealth))
}

func yesOrNo(value bool) string {
	if value {
		return "YES"
	}

	return "NO"
}

func parseFlagSet(fs *flag.FlagSet, args []string) error {

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Check if all parameters were parsed
	if fs.NArg() != 0 {
		return ierrors.New("too much arguments")
	}

	return nil
}
