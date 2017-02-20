package cli

import (
	"io"
	"text/template"
)

// TemplatePrinter prints data according to a template.
type TemplatePrinter struct {
	r        io.Reader
	w        io.Writer
	rows     []interface{}
	position int
	Wait     bool
	Template string
}

// NewTemplatePrinter returns a TemplatePrinter initialised with given options
func NewTemplatePrinter(writer io.Writer, reader io.Reader, tpl string) *TemplatePrinter {
	tp := &TemplatePrinter{
		w:        writer,
		r:        reader,
		rows:     make([]interface{}, 0, 20),
		Template: tpl,
	}

	return tp
}

// Append adds a row of data to the collection of data to print
func (t *TemplatePrinter) Append(row interface{}) {
	t.rows = append(t.rows, row)
}

//BulkAppend adds a serie of rows of data to the collection of data to print
func (t *TemplatePrinter) BulkAppend(rows []interface{}) {
	t.rows = append(t.rows, rows...)
}

// PrintAll will print all the remaining data. Passing true will make the printer wait for the user to press enter between each rows
func (t *TemplatePrinter) PrintAll(wait bool) error {

	tpl, err := template.New("userTemplate").Parse(t.Template)
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
