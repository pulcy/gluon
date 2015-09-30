package templates

import (
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/juju/errgo"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

func Render(templateName, destinationPath string, options interface{}, destinationFileMode os.FileMode) error {
	asset, err := Asset(templateName)
	if err != nil {
		return maskAny(err)
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
		return maskAny(err)
	}
	destinationDir := filepath.Dir(destinationPath)
	if err := os.MkdirAll(destinationDir, destinationFileMode); err != nil {
		return maskAny(err)
	}
	f, err := os.Create(destinationPath)
	if err != nil {
		return maskAny(err)
	}
	// write file to host
	tmpl.Funcs(funcMap)
	err = tmpl.Execute(f, options)
	if err != nil {
		return maskAny(err)
	}
	f.Chmod(destinationFileMode)

	return nil
}

func escape(s string) string {
	s = strconv.Quote(s)
	return s[1 : len(s)-1]
}
