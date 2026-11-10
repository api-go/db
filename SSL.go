package db

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/api-go/log"
)

func BuildTLSConfig(ca, cert, key []byte, insecure bool) *tls.Config {
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(ca) {
		log.DefaultLogger.Error("ca error for db")
		return nil
	}

	certs, err := tls.X509KeyPair(cert, key)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{certs},
		RootCAs:            caPool,
		InsecureSkipVerify: insecure,
	}
}
