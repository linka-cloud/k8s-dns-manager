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
	k8s_crds
{{- if .Forward }}
	forward . {{ range $val := .Forward }}{{ $val }} {{ end }}
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
