/*
Copyright 2020 The Linka Cloud Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"bytes"
	"text/template"
)

type Config struct {
	Forward []string
	Log     bool
	Errors  bool
	Metrics bool
	Cache   int
	Any     bool
}

func (c Config) Render() (string, error) {
	b := &bytes.Buffer{}
	if err := configTemplate.Execute(b, c); err != nil {
		return "", err
	}
	return b.String(), nil
}

var configTemplate = template.Must(template.New("corefile").Parse(`
.:53 {
	k8s_dns
{{- if .Any }}
	any
{{- end }}
{{- if .Forward }}
	forward . {{ range $val := .Forward }}{{ $val }} {{ end }}
{{- end }}
{{- if .Cache }}
	cache {{ .Cache }}
{{- end }}
{{- if .Log }}
	log
{{- end }}
{{- if .Errors }}
	errors
{{- end }}
{{- if .Metrics }}
	prometheus
{{- end }}
}
`))
