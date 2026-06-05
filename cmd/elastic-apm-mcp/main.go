package main

import (
	"fmt"
	"os"

	"github.com/skynet2/elastic-apm-mcp/cmd/elastic-apm-mcp/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
