package cli

import (
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/howeyc/gopass"
)

// Cli holds reader and writer to interact wih user
type Cli struct {
	In  io.Reader
	Out io.WriteCloser
	Err io.WriteCloser
}

// ReadCredentials ask the user to enter his login and password and stores the value in the given registryAuth
func ReadCredentials(c *Cli, registryAuth *types.AuthConfig) {
	logrus.WithFields(logrus.Fields{"Login": registryAuth.Username, "Password": registryAuth.Password}).Debugln("Reading new credentials")
	askUsername(registryAuth, fmt.Scanln, c.Out)
	if registryAuth.Username == "" {
		return
	}

	askPassword(registryAuth, ReadPassword, c.Out)

}

// InputReader defines a method that can read into a string
type InputReader func(a ...interface{}) (int, error)

func askUsername(registryAuth *types.AuthConfig, read InputReader, w io.Writer) {
	var prompt string
	if registryAuth.Username != "" {
		prompt = fmt.Sprintf("Username (%s) :", registryAuth.Username)
	} else {
		prompt = fmt.Sprint("Username :")
	}
	input := ask(prompt, read, w)
	if input != "" {
		registryAuth.Username = input
	}

}

func ask(prompt string, read InputReader, w io.Writer) string {
	fmt.Fprint(w, prompt)
	var input string
	read(&input)
	return input
}

func askPassword(registryAuth *types.AuthConfig, read InputReader, w io.Writer) {
	input := ask("Password :", read, w)
	if input != "" {
		registryAuth.Password = input
	}
}

// ReadPassword reads from the terminal without echoing input
func ReadPassword(a ...interface{}) (int, error) {
	pass, err := gopass.GetPasswd()
	if a, ok := a[0].(*string); ok {
		*a = string(pass)
	} else {
		err = fmt.Errorf("Only reading string is supported")
	}
	return 1, err
}

// Printer defines common method to print rows of results
type Printer interface {
	PrintAll(wait bool) error
}
