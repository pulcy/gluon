// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates

import (
	"bytes"
	"os"
	"strconv"
	"text/template"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

type TemplateConfigurator func(*template.Template)

// Render updates the given destinationPath according to the given template and options.
// Returns true if the file was created or changed, false if nothing has changed.
func Render(templateName, destinationPath string, options interface{}, destinationFileMode os.FileMode, config ...TemplateConfigurator) (bool, error) {
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
	for _, c := range config {
		c(tmpl)
	}
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
