package cli

import (
	"io"
	"os"
	"strings"
	"text/template"
)

// TabPrinter prints data in tabular format. It computes column width and support incremental data population
type TabPrinter struct {
	w         io.Writer
	widths    []int
	rows      [][]string
	position  int
	Separator byte
	Width     int
	Wait      bool
}

// NewTabPrinter returns a TabPrinter with sane defaults
func NewTabPrinter(writer io.Writer) *TabPrinter {
	return &TabPrinter{w: writer, Separator: '\t', Width: 80, rows: make([][]string, 0, 20)}
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

	tpl, _ := template.New("rowPrinter").Funcs(funcMap).Parse(rowTemplate)
	for _, row := range t.rows[t.position:] {
		if wait {
			var b = make([]byte, 1, 1)
			os.Stdin.Read(b)
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
