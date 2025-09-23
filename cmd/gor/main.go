package main

import (
	"fmt"
	"os"

	"github.com/cuemby/gor/internal/cli"
)

const version = "1.0.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	app := cli.NewApp(version)
	return app.Run(os.Args)
}
