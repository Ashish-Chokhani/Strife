// client/config/config.go

package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"

	"google.golang.org/grpc/credentials"
)

// Config holds application configuration
type Config struct {
	ServerAddress string
	DataDir       string
	CertPaths     CertificatePaths
}

// CertificatePaths holds paths to certificate files
type CertificatePaths struct {
	ClientCert string
	ClientKey  string
	CACert     string
}

// LoadConfig loads application configuration
func LoadConfig() (*Config, error) {
	// Create data directory if it doesn't exist
	dataDir := "data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, 0755)
	}

	return &Config{
		ServerAddress: "localhost:50053",
		DataDir:       dataDir,
		CertPaths: CertificatePaths{
			ClientCert: "certs/client.crt",
			ClientKey:  "certs/client.key",
			CACert:     "certs/ca.crt",
		},
	}, nil
}

// SetupTLSCredentials sets up TLS credentials for secure connections
func SetupTLSCredentials(certPaths CertificatePaths) (credentials.TransportCredentials, error) {
	// Load client certificate and key
	clientCert, err := tls.LoadX509KeyPair(certPaths.ClientCert, certPaths.ClientKey)
	if err != nil {
		return nil, err
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile(certPaths.CACert)
	if err != nil {
		return nil, err
	}

	// Create CA certificate pool
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, err
	}

	// Create TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{clientCert}, // Client cert for mTLS
		RootCAs:      caCertPool,                    // CA cert to verify server
		ServerName:   "localhost",                   // Must match server cert CN/SAN
	})

	return creds, nil
}