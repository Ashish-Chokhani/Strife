package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"google.golang.org/grpc/credentials"
)

// SetupTLSCredentials sets up server TLS credentials
func SetupTLSCredentials(cfg TLSConfig) (credentials.TransportCredentials, error) {
	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(cfg.ServerCertPath, cfg.ServerKeyPath)
	if err != nil {
		return nil, err
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, err
	}

	// Create certificate pool and add CA certificate
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})

	return creds, nil
}