package main

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"
	"time"
)

// The built-in certificate list alone (without the computer's own) must reach real HTTPS hosts.
func TestBundledCerts(t *testing.T) {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certsPEM) {
		t.Fatal("certs.pem has no certificates")
	}
	hc := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}
	for _, u := range []string{"https://workers.cloudflare.com", "https://letsencrypt.org"} {
		resp, err := hc.Get(u)
		if err != nil {
			t.Fatalf("%s: %v", u, err)
		}
		resp.Body.Close()
	}
}
