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

	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
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
		PreRun: func(cmd *cobra.Command, args []string) {
			sslCertFile = viper.GetString("ssl-cert-file")
			sslKeyFile = viper.GetString("ssl-key-file")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			handleSignal()

			return runServer(c, ctx, cmd, args)
		},
	}

	serverCommand.Flags().StringVarP(&port, "port", "p", "0.0.0.0:6000", "Dim listening port")
	serverCommand.Flags().StringVar(&indexDir, "index-path", "dim.index", "Dim listening port")
	serverCommand.Flags().String("ssl-cert-file", "", "SSL certificate file for https connections")
	serverCommand.Flags().String("ssl-key-file", "", "SSL key file for https connections")

	viper.BindPFlag("ssl-cert-file", serverCommand.Flags().Lookup("ssl-cert-file"))
	viper.BindPFlag("ssl-key-file", serverCommand.Flags().Lookup("ssl-key-file"))

	rootCommand.AddCommand(serverCommand)
}

func runServer(c *cli.Cli, ctx context.Context, cmd *cobra.Command, args []string) error {
	if registryURL == "" {
		return fmt.Errorf("No registry URL given")
	}

	if (sslCertFile == "") != (sslKeyFile == "") {
		return fmt.Errorf("ssl-cert-file and ssl-key-file cannot be defined separately")
	}

	realDir := path.Join(indexDir, time.Now().Format("20060102150405.000"))
	logrus.Warnf("Creating index dir at %s\n", realDir)

	var authConfig *types.AuthConfig
	var u, p string
	if username != "" || password != "" {
		authConfig = &types.AuthConfig{Username: username, Password: password}
		u, p = authConfig.Username, authConfig.Password
	}

	var idx *index.Index
	var err error

	var cfg *index.Config
	if cfg, err = readConfigHooks(hookFunctions); err != nil {
		return err
	}
	cfg.Directory = realDir

	var client dim.RegistryClient

	var url *url.URL
	if url, err = url.Parse(registryURL); err != nil {
		return err
	}

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

	proxy := server.NewRegistryProxy(url, u, p)

	var sCfg *server.Config
	if sCfg, err = readServerConfig(); err != nil {
		return err
	}
	s = server.NewServer(sCfg, idx, ctx, proxy)

	logrus.WithField("port", port).Infoln("Server listening...")

	if sslCertFile != "" {
		logrus.WithFields(logrus.Fields{"cert": sslCertFile, "key": sslKeyFile}).Debugln("Starting https server")
		return s.RunSecure(sslCertFile, sslKeyFile)
	}

	return s.Run()
}

var (
	port        string
	indexDir    string
	sslCertFile string
	sslKeyFile  string
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

var defaultNotificationRequest = notificationRequest{
	method: "GET",
}

type notificationRequest struct {
	method  string
	payload string
	headers http.Header
}

type notificationRequestOption func(*notificationRequest)

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
	"sendRequest": func(url string, options ...notificationRequestOption) error {
		request := defaultNotificationRequest
		request.headers = make(http.Header)

		for _, opts := range options {
			opts(&request)
		}

		r, _ := http.NewRequest(request.method, url, bytes.NewBufferString(request.payload))
		r.Header = request.headers
		logrus.WithFields(logrus.Fields{"url": url, "payload": request.payload, "method": request.method}).Infoln("Send request")

		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			return err
		}
		logrus.WithFields(logrus.Fields{"url": url, "payload": request.payload, "method": request.method}).Infoln(resp.Status)
		return nil
	},
	"withPayload": func(payload string) notificationRequestOption {
		return func(req *notificationRequest) {
			req.payload = payload
		}
	},
	"withMethod": func(method string) notificationRequestOption {
		return func(req *notificationRequest) {
			req.method = method
		}
	},
	"withHeader": func(key, value string) notificationRequestOption {
		return func(req *notificationRequest) {
			req.headers.Add(key, value)
		}
	},
}
