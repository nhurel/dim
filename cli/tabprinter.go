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
	"io"
	"strings"
	"text/template"
)

// TabPrinter prints data in tabular format. It computes column width and support incremental data population
type TabPrinter struct {
	r         io.Reader
	w         io.Writer
	widths    []int
	rows      [][]string
	position  int
	Separator byte
	Width     int
	Wait      bool
	Template  string
}

// NewTabPrinter returns a TabPrinter with sane defaults
func NewTabPrinter(writer io.Writer, reader io.Reader, options ...TabPrinterOption) *TabPrinter {
	tp := &TabPrinter{
		w:         writer,
		r:         reader,
		Separator: '\t',
		Width:     80,
		rows:      make([][]string, 0, 20),
		Template:  rowTemplate,
	}
	for _, opt := range options {
		opt(tp)
	}
	return tp
}

// TabPrinterOption lets you customize your TabPrinter instance
type TabPrinterOption func(printer *TabPrinter)

// WithSeparator returns a TabPrinterOption to set the printer separator character
func WithSeparator(sep byte) TabPrinterOption {
	return func(printer *TabPrinter) {
		printer.Separator = sep
	}
}

// WithWidth returns a TabPrinterOption to set the printer width
func WithWidth(w int) TabPrinterOption {
	return func(printer *TabPrinter) {
		printer.Width = w
	}
}

// WithTemplate returns a TabPrinterOption to set the printer template
func WithTemplate(t string) TabPrinterOption {
	return func(printer *TabPrinter) {
		printer.Template = t
	}
}

// Append adds a row of data to the collection of data to print
func (t *TabPrinter) Append(row []string) {
	t.rows = append(t.rows, row)
}

//BulkAppend adds a serie of rows of data to the collection of data to print
func (t *TabPrinter) BulkAppend(rows [][]string) {
	t.rows = append(t.rows, rows...)
}

// PrintAll will print all the remaining data. Passing true will make the printer wait for the user to press enter between each rows
func (t *TabPrinter) PrintAll(wait bool) error {
	t.computeWidths()
	funcMap := template.FuncMap{
		"format": t.format,
		"sep":    func() string { return string(t.Separator) },
	}

	tpl, err := template.New("rowPrinter").Funcs(funcMap).Parse(rowTemplate)
	if err != nil {
		return err
	}
	for _, row := range t.rows[t.position:] {
		if wait {
			var b = make([]byte, 1, 1)
			t.r.Read(b)
		}
		t.position++

		if err := tpl.Execute(t.w, row); err != nil {
			return err
		}

		if !wait && t.position != len(t.rows) {
			t.w.Write([]byte("\n"))
		}
	}
	return nil
}

func (t *TabPrinter) format(s string, i int) string {
	width := t.widths[i]
	return format(s, width)
}

func format(s string, length int) string {
	if len(s) == length {
		return s
	}

	var b []byte

	if len(s) < length {
		b = make([]byte, 0, length)
		b = append(b, []byte(s)...)
		for i := len(b); i < length; i++ {
			b = append(b, ' ')
		}
	}
	if len(s) > length {
		b = []byte(s[0:length])
		b[length-3] = '.'
		b[length-2] = '.'
		b[length-1] = '.'
	}

	return string(b)
}

func (t *TabPrinter) computeWidths() {
	if t.widths != nil {
		return
	}
	widths := make([]int, len(t.rows[0]))
	entries := make([]int, len(t.rows[0]))
	fullwidth := 0
	for _, row := range t.rows {
		for i, s := range row {
			length := len(strings.TrimSpace(s))
			if length > 0 {
				widths[i] = (widths[i]*entries[i] + length) / (entries[i] + 1)
				entries[i]++
			}
		}
	}
	for _, width := range widths {
		fullwidth += width
	}
	if t.widths == nil {
		t.widths = make([]int, len(widths))
	}
	width := (t.Width - len(t.rows[0])) + 1
	for i := 0; i < len(widths); i++ {
		t.widths[i] = widths[i] * width / fullwidth
	}
}

const rowTemplate = `{{range $i, $r := .}}
{{- if gt $i 0}}{{sep}}{{end}}{{format $r $i}}
{{- end}}`
