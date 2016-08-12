package main

import (
	"fmt"
	"github.com/nhurel/dim/cmd"
	"os"
)

var GenerateCompletion string

func main() {
	if GenerateCompletion == "true" {
		complete()
		return
	}
	if err := cmd.RootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)

}

func complete() {
	cmd.RootCommand.GenBashCompletionFile("dim_compl")
}
