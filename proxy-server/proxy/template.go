package proxy

import (
	"os"
	"text/template"
)

var haproxyConfigTemplate = `
{{- range $port, $targets := . }}
frontend tcp-{{ $port }}
	mode tcp
	bind 0.0.0.0:{{ $port }}

	# Wait up to 5s from the time the TCP socket opens for an SSL handshake
	tcp-request inspect-delay 5s
	tcp-request content accept if { req_ssl_hello_type 1 }

	# Filter connections by SNI. Invalid hostnames will be rejected.
	{{- range $target := $targets }}
	use_backend loopback-{{ $target.hostname }} if { req_ssl_sni -i {{ $target.hostname }} }
	{{ end }}
{{ end }}

{{- range $port, $targets := . }}
{{- range $target := $targets }}
#----------- {{ $target.hostname }} proxy -----------
backend loopback-{{ $target.hostname }}
	# Loopback to TLS listener on abstract namespace
	server loopback-for-tls abns@{{ $target.hostname }} send-proxy-v2

listen {{ $target.hostname }}
	# Internal bind with TLS configuration for the hostname
	bind abns@{{ $target.hostname }} accept-proxy ssl crt {{ $target.certBundle }} ca-file {{ $target.caFile}} strict-sni
	server {{ $target.serviceName }} {{ $target.serviceAddress }} resolvers dns
{{ end }}
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
