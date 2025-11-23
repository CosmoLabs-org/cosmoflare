/*
R2Go2 - A production-ready CLI tool for managing Cloudflare R2 buckets

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT (https://opensource.org/licenses/MIT)

For usage instructions, see: https://github.com/CosmoLabs-org/CosmoDev-R2Go2
*/

package main

import (
	"fmt"
	"os"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd"
)

func main() {
	// Build info is set in cmd package via ldflags

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}