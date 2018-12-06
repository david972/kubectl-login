package main

import (
	"context"
	"log"
	"os"

	"github.com/david972/kubelogin/cli"
)

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
