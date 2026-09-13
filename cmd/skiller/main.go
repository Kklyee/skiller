package main

import (
	"fmt"
	"os"

	"github.com/Kklyee/skiller/internal/cli"
)

var version = "dev"

func main() {
  if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
