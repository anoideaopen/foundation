package telemetry

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

func getTLSConfig(caCertsBase64 string) (*tls.Config, error) {
	caCertsBytes, err := base64.StdEncoding.DecodeString(caCertsBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode TLS configuration: %s", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCertsBytes)
	if !ok {
		return nil, fmt.Errorf("failed to add CA certificates to CA cert pool")
	}

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	return tlsConfig, nil
}
