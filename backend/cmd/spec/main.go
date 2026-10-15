// Command spec updates the embedded OpenAPI contract from Go types.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/api"
)

func main() {
	output := flag.String("output", "internal/api/openapi.json", "OpenAPI output file")
	flag.Parse()
	data, err := api.Specification()
	if err == nil {
		err = os.WriteFile(*output, data, 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
