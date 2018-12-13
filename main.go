package main

import (
	"context"
	"log"
	"os"

	"github.com/david972/kubectl-login/cli"
)

var version = "2.1.0-beta.1"

func main() {
	c, err := cli.Parse(os.Args, version)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	if err := c.Run(ctx); err != nil {
		log.Fatalf("Error: %s", err)
	}
}
