package main

import (
	"os"

	"github.com/MakazhanAlpamys/claudeshield/cmd/claudeshield/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
