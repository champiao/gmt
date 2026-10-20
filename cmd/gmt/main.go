package main

import (
	"fmt"
	"os"

	"github.com/champiao/gmt/pkg/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Erro: %v\n", err)
		os.Exit(1)
	}
}
