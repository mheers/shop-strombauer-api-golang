// Command specfix normalizes the vendored OpenAPI document before
// oapi-codegen runs.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mheers/shop-strombauer-api-golang/internal/specfix"
)

func main() {
	in := flag.String("in", "spec/openapi.json", "input OpenAPI document")
	out := flag.String("out", "spec/openapi.normalized.json", "normalized output document")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "specfix:", err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	src, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	normalized, err := specfix.Normalize(src)
	if err != nil {
		return err
	}
	return os.WriteFile(out, normalized, 0o644)
}
