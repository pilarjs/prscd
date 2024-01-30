package prscd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"
)

func loadTLS(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	// check if TLS cert is expired
	// Parse the X.509 certificate
	parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}

	// Get the expiration date
	expirationDate := parsedCert.NotAfter
	log.Debug("check TLS cert expiration date", "date", expirationDate)

	// determine if the certificate is expired
	if time.Now().After(expirationDate) {
		return nil, fmt.Errorf("tls cert is expired")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"http/1.1", "h2", "h3", "http/0.9", "http/1.0", "spdy/1", "spdy/2", "spdy/3"},
	}, nil
}
