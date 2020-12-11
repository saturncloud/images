package proxy

import (
	"os"
	"text/template"
)

var haproxyConfigTemplate = `
{{- range $port, $targets := . }}
frontend dask-scheduler
	bind 0.0.0.0:{{ $port }} ssl strict-sni crt-list /etc/haproxy/config/crt-list.txt

	{{- range $hostname, $service := $targets }}
	acl {{ $hostname }} ssl_fc_sni -i {{ $hostname }}
	use_backend {{ $hostname }} if {{ $hostname }}
	{{ end }}

{{- range $hostname, $service := $targets }}
backend {{ $hostname }}
	server scheduler {{ $service }} resolvers dns
{{ end }}
{{ end }}
`

var haproxyCertListTemplate = `
{{- range $entry := . }}
{{ $entry.certBundle }} [ca-file {{ $entry.caFile }} verify required] {{ $entry.hostname }}
{{ end }}
`

/*
	Renders a template with the given input data
*/
func renderTemplate(templateString, outFile string, data interface{}) error {
	tmpl, err := template.New("tmpl").Parse(templateString)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(outFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}
