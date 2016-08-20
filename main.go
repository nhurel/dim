package main

import (
	"fmt"
	"github.com/nhurel/dim/cmd"
	"os"
)

// Version stores current version of dim (see Makefile)
var Version string

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("dim version : %s\n", Version)
		os.Exit(0)
	}
	if err := cmd.RootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
