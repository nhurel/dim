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

package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestPrintAll(t *testing.T) {
	writer := bytes.NewBufferString("")
	printer := NewTabPrinter(writer, os.Stdin, WithWidth(30), WithSeparator('+'))
	printer.Append([]string{"ABCDE", "ABCDE", "ABCDE", "ABCDE", "ABCDE"})

	printer.PrintAll(false)

	if writer.String() != "ABCDE+ABCDE+ABCDE+ABCDE+ABCDE" {
		t.Errorf("PrintAll returned %s instead of %s", writer.String(), "ABCDE+ABCDE+ABCDE+ABCDE+ABCDE")
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		original string
		length   int
		expected string
	}{
		{"short", 10, "short     "},
		{"stringtoolong", 10, "stringt..."},
		{"goodstring", 10, "goodstring"},
	}

	for _, test := range tests {
		got := format(test.original, test.length)
		if len(got) != test.length {
			t.Errorf("format returned a string of %d chars instead of %d", len(got), test.length)
		}
		if got != test.expected {
			t.Errorf("format returned '%s' instead of '%s'", got, test.expected)
		}
	}
}

func TestComputeWidths(t *testing.T) {
	printer := NewTabPrinter(os.Stdout, os.Stdin, WithWidth(80), WithSeparator('|'))
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})

	printer.computeWidths()

	expected := []int{17, 8, 38, 12}
	for i := 0; i < 4; i++ {
		if printer.widths[i] != expected[i] {
			t.Errorf("printer computed wrong widths : Got %v instead of %v", printer.widths, expected)
			break
		}
	}

	// Results should be the same even when some values are missing :
	printer = NewTabPrinter(os.Stdout, os.Stdin, WithWidth(80), WithSeparator('|'))
	printer.Append([]string{"ABCDEFGH", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"", "ABCD", "", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "", "ABCDEFGHIJKLMNOPQR", ""})
	printer.Append([]string{"ABCDEFGH", "ABCD", "", "ABCDEF"})
	printer.Append([]string{"", "ABCD", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})
	printer.Append([]string{"ABCDEFGH", "", "ABCDEFGHIJKLMNOPQR", "ABCDEF"})

	printer.computeWidths()

	for i := 0; i < 4; i++ {
		if printer.widths[i] != expected[i] {
			t.Errorf("printer computed wrong widths : Got %v instead of %v", printer.widths, expected)
			break
		}
	}

}
