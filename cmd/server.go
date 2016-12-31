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

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"

	"context"

	"bytes"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/index"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newServerCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	serverCommand := &cobra.Command{
		Use:   "server",
		Short: "Runs in server mode to provide search feature",
		Long: `Start dim in server mode. In this mode, dim indexes your private registry and provide a search endpoint.
Use the --port flag to specify the adress the server listens.
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			handleSignal()

			return runServer(c, ctx, cmd, args)
		},
	}

	serverCommand.Flags().StringVarP(&port, "port", "p", "0.0.0.0:6000", "Dim listening port")
	serverCommand.Flags().StringVar(&indexDir, "index-path", "dim.index", "Dim listening port")
	rootCommand.AddCommand(serverCommand)
}

func runServer(c *cli.Cli, ctx context.Context, cmd *cobra.Command, args []string) error {
	if registryURL == "" {
		return fmt.Errorf("No registry URL given")
	}

	realDir := path.Join(indexDir, time.Now().Format("20060102150405.000"))
	logrus.Warnf("Creating index dir at %s\n", realDir)

	var authConfig *types.AuthConfig
	if username != "" || password != "" {
		authConfig = &types.AuthConfig{Username: username, Password: password}
	}

	var idx *index.Index
	cfg := &index.Config{Directory: realDir}
	var err error

	hooks := make([]*index.Hook, 0, 10)
	if err = viper.UnmarshalKey("index.hooks", &hooks); err != nil {
		return err
	}
	cfg.Hooks = hooks
	for n, f := range hookFunctions {
		cfg.RegisterFunction(n, f)
	}

	var client dim.RegistryClient

	if client, err = registry.New(c, authConfig, registryURL); err != nil {
		return fmt.Errorf("Failed to connect to registry : %v", err)
	}

	if idx, err = index.New(cfg, client); err != nil {
		return err
	}

	indexationDone := idx.Build()

	go func() {
		_ = <-indexationDone
		logrus.Infoln("All images indexed")
	}()
	s = server.NewServer(port, idx, ctx)
	logrus.Infoln("Server listening...")
	return s.Run()
}

var (
	port     string
	indexDir string
)

var s *server.Server

func handleSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			if s != nil {
				logrus.Infoln("ShuttingDown server")
				s.BlockingClose()
			}
			os.Exit(0)
		}
	}()
}

var hookFunctions = map[string]interface{}{
	"info": func(args ...interface{}) bool {
		logrus.Infoln(args)
		return true
	},
	"warn": func(args ...interface{}) bool {
		logrus.Warnln(args)
		return true
	},
	"error": func(args ...interface{}) bool {
		logrus.Errorln(args)
		return true
	},
	"sendRequest": func(method, url, payload string) error {

		r, _ := http.NewRequest(method, url, bytes.NewBufferString(payload))
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			return err
		}
		logrus.WithFields(logrus.Fields{"url": url, "payload": payload, "method": method}).Infoln(resp.Status)
		return nil
	},
}
