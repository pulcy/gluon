package templates

import (
	"bytes"
	"os"
	"strconv"
	"text/template"

	"github.com/juju/errgo"

	"github.com/pulcy/yard/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

// Render updates the given destinationPath according to the given template and options.
// Returns true if the file was created or changed, false if nothing has changed.
func Render(templateName, destinationPath string, options interface{}, destinationFileMode os.FileMode) (bool, error) {
	asset, err := Asset(templateName)
	if err != nil {
		return false, maskAny(err)
	}

	// parse template
	var tmpl *template.Template
	tmpl = template.New(templateName)
	funcMap := template.FuncMap{
		"escape": escape,
		"quote":  strconv.Quote,
	}
	tmpl.Funcs(funcMap)
	_, err = tmpl.Parse(string(asset))
	if err != nil {
		return false, maskAny(err)
	}
	// execute template to buffer
	buf := &bytes.Buffer{}
	tmpl.Funcs(funcMap)
	err = tmpl.Execute(buf, options)
	if err != nil {
		return false, maskAny(err)
	}

	// Update file
	changed, err := util.UpdateFile(destinationPath, buf.Bytes(), destinationFileMode)
	return changed, maskAny(err)
}

func escape(s string) string {
	s = strconv.Quote(s)
	return s[1 : len(s)-1]
}
