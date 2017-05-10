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

package utils

import (
	"fmt"
	"sort"
	"strings"

	"crypto/sha256"
	"encoding/hex"
	"time"
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

// ParseDuration returns a string representation of a duration
func ParseDuration(since time.Duration) string {
	if since.Hours() > 24 {
		return parseHours(since.Hours())
	}
	if since.Minutes() >= 90 {
		return fmt.Sprintf("%0.f hours ago", since.Hours())
	}
	if since.Minutes() < 90 && since.Minutes() > 60 {
		return "1 hour ago"
	}
	if since.Seconds() > 90 {
		return fmt.Sprintf("%0.f minutes ago", since.Minutes())
	}
	if since.Seconds() > 60 {
		return "1 minute ago"
	}
	if since.Seconds() < 1.5 {
		return "1 second ago"
	}
	return fmt.Sprintf("%0.f seconds ago", since.Seconds())
}

func parseHours(hours float64) string {
	if hours < 48 {
		return fmt.Sprintf("%0.f hours ago", hours)
	}
	if hours < 24*7*2 {
		return fmt.Sprintf("%0.f days ago", hours/(24))
	}
	if hours < 24*7*4*2 {
		return fmt.Sprintf("%0.f weeks ago", hours/(24*7))
	}

	return fmt.Sprintf("%0.f months ago", hours/(24*7*4))
}

// Sha256 returns the string reprensation of the given password encoded using sha256
func Sha256(passwd string) string {
	h := sha256.New()
	h.Write([]byte(passwd))
	return hex.EncodeToString(h.Sum(nil))
}
