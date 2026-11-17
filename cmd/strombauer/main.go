// Command strombauer is the command line client for the Strombauer Store API.
package main

import (
	"fmt"
	"os"

	"github.com/mheers/shop-strombauer-api-golang/internal/cli"
)

func main() {
	if err := cli.NewRootCmd(cli.Options{}).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
