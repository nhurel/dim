package cmd

import (
	"context"
	"fmt"
	"net/http"

	"strings"

	"github.com/docker/docker/reference"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/index"
	"github.com/nhurel/dim/lib/registry"
	"github.com/spf13/cobra"
)

func newHooktestCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	hooktestCommand := &cobra.Command{
		Use:   "hooktest IMAGE[:TAG]",
		Short: "Tests index hooks against an image",
		Long: `Tests index hook against the image IMAGE.
If no TAG is specified, latest will be used.
If flag -r is given the image is read from the remote registry.
testhooks will parse and execute templates given in dim configuration with mock functions so user can check the hooks behaves as expected for a given image`,
		Example: `dim hooktest ubuntu
dim hooktest -r ubuntu:xenial
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHooktest(c, args)
		},
	}

	hooktestCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Reads the image used to test hooks from the remote registry")
	rootCommand.AddCommand(hooktestCommand)
}

func runHooktest(c *cli.Cli, args []string) error {
	var err error
	if len(args) == 0 {
		return fmt.Errorf("image name missing")
	}

	image := args[0]

	var cfg *index.Config
	if cfg, err = readConfigHooks(mockHookFunctions(c)); err != nil {
		return err
	}

	var hookedImage *dim.IndexImage

	if remoteFlag {
		var client dim.RegistryClient
		var parsedName reference.Named
		var repo dim.Repository

		if client, parsedName, err = connectRegistry(c, image); err != nil {
			return err
		}
		name, _ := reference.ParseNamed(parsedName.Name()[strings.Index(parsedName.Name(), "/")+1:])
		if repo, err = client.NewRepository(name); err != nil {
			return err
		}

		tag := registry.ParseTag(parsedName)
		var image *dim.RegistryImage
		if image, err = repo.Image(tag); err != nil {
			return err
		}

		hookedImage = index.Parse(parsedName.Name(), image)

	} else {
		if hookedImage, err = Dim.AsIndexImage(image); err != nil {
			return err
		}
	}

	for n, hook := range cfg.Hooks {
		fmt.Fprintf(c.Out, "Hook #%d would produce :\n", n)
		if err := hook.Eval(hookedImage); err != nil {
			fmt.Fprintf(c.Err, "Failed to evaluate hook #%d", n)
		}
	}

	return nil
}

func mockHookFunctions(c *cli.Cli) map[string]interface{} {
	return map[string]interface{}{
		"info": func(args ...interface{}) bool {
			switch len(args) {
			case 1:
				fmt.Fprintf(c.Out, "Would have logged at INFO level : %s\n", args[0].(string))
			default:
				fmt.Fprintf(c.Out, "Would have logged at INFO level : %s\n", fmt.Sprintf("%s %v", args[0].(string), args[1:]))

			}
			return true
		},
		"warn": func(args ...interface{}) bool {
			switch len(args) {
			case 1:
				fmt.Fprintf(c.Out, "Would have logged at WARN level : %s\n", args[0].(string))
			default:
				fmt.Fprintf(c.Out, "Would have logged at WARN level : %s\n", fmt.Sprintf("%s %v", args[0].(string), args[1:]))

			}
			return true
		},
		"error": func(args ...interface{}) bool {
			switch len(args) {
			case 1:
				fmt.Fprintf(c.Out, "Would have logged at ERROR level : %s\n", args[0].(string))
			default:
				fmt.Fprintf(c.Out, "Would have logged at ERROR level : %s\n", fmt.Sprintf("%s %v", args[0].(string), args[1:]))

			}
			return true
		},
		"sendRequest": func(url string, options ...notificationRequestOption) error {
			request := defaultNotificationRequest
			request.headers = make(http.Header)

			for _, opts := range options {
				opts(&request)
			}

			if request.payload == "" {
				fmt.Fprintf(c.Out, "Would have sent request with method %s to %s with headers %v\n", request.method, url, request.headers)
			} else {
				fmt.Fprintf(c.Out, "Would have sent payload %s with method %s to %s with headers %v\n", request.payload, request.method, url, request.headers)
			}

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
}
