package utils

import (
	"fmt"
	"sort"
	"strings"

	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/howeyc/gopass"
)

// ListContains checks a list of string contains a given string
func ListContains(list []string, search string) bool {
	for _, r := range list {
		if r == search {
			return true
		}
	}
	return false
}

// MapMatchesAll checks first map contains all the second map elements with the same value
func MapMatchesAll(all, search map[string]string) bool {

	if all == nil || search == nil {
		return false
	}

	if len(all) >= len(search) {
		for k, v := range search {
			if all[k] != v {
				return false
			}
		}
		return true
	}
	return false

}

// Keys returns all the keys of the given map
func Keys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// FilterValues removes all element from m where value is s
func FilterValues(m map[string]string, s string) map[string]string {
	filtered := make(map[string]string, len(m))
	for k, v := range m {
		if v != s {
			filtered[k] = v
		}
	}
	return filtered

}

// ReadCredentials ask the user to enter his login and password and stores the value in the given registryAuth
func ReadCredentials(registryAuth *types.AuthConfig) {
	logrus.WithFields(logrus.Fields{"Login": registryAuth.Username, "Password": registryAuth.Password}).Debugln("Reading new credentials")
	askUsername(registryAuth, fmt.Scanln, os.Stdout)
	if registryAuth.Username == "" {
		return
	}

	askPassword(registryAuth, readPassword, os.Stdout)

}

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

func readPassword(a ...interface{}) (int, error) {
	pass, err := gopass.GetPasswd()
	if a, ok := a[0].(*string); ok {
		*a = string(pass)
	} else {
		err = fmt.Errorf("Only reading string is supported")
	}
	return 1, err
}

// BuildURL appends http:// or https:// to given hostname, according to the insecure parameter
func BuildURL(hostname string, insecure bool) string {
	if hostname == "" {
		return ""
	}
	if strings.HasPrefix(hostname, "http://") || strings.HasPrefix(hostname, "https://") {
		return hostname
	}

	protocol := "https"
	if insecure {
		protocol = "http"
	}
	return fmt.Sprintf("%s://%s", protocol, hostname)

}

// FlatMap returns a string representation of a map
func FlatMap(m map[string]string) string {
	if m == nil || len(m) == 0 {
		return ""
	}
	entries := make([]string, 0, len(m))
	for k, v := range m {
		entries = append(entries, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(entries)
	return strings.Join(entries, ", ")
}
