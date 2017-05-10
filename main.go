// Copyright 2016
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"

	"github.com/nhurel/dim/cmd"
	"golang.org/x/net/context"
	"github.com/nhurel/dim/lib/environment"
	"github.com/nhurel/dim/cli"
)

// Version stores current version of dim (see Makefile)
var Version string

func main() {
	c := &cli.Cli{In:os.Stdin, Out:os.Stdout, Err:os.Stderr}
	ctx := environment.Set(context.Background(), environment.VersionKey, Version)

	if len(os.Args) == 2 && (os.Args[1] == "--version") {
		if err := cmd.PrintVersion(c, ctx); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if err := cmd.NewRootCommand(c, ctx).Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
