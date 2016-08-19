package main

import (
	"fmt"
	"github.com/nhurel/dim/cmd"
	"os"
)

func main() {
	if err := cmd.RootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
