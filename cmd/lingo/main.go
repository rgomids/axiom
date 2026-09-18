package main

import (
	"context"
	"os"

	"github.com/rgomids/axiom/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.UnavailableService{}, os.Stdout))
}
