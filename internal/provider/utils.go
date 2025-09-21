package provider

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/proxy"
)

// Convert an index type declared in terraform as string into a type and value expected by Mongo's client.
func convertToMongoIndexType(indexType string) interface{} {
	switch indexType {
	case "asc":
		return 1
	case "desc":
		return -1
	default:
		return indexType
	}
}

// Convert an index type returned by Mongo's client into a string understood by terraform.
func convertToTfIndexType(indexType interface{}) (string, error) {
	intValue, isInt := indexType.(int32)
	if isInt {
		switch intValue {
		case 1:
			return "asc", nil
		case -1:
			return "desc", nil
		default:
			return "", errors.New("if typeIndex is int, it MUST have value 1 ou -1")
		}
	}

	strValue, isStr := indexType.(string)
	if isStr {
		return strValue, nil
	}

	return "", errors.New("typeIndex MUST be int32 or string")
}

type indexId struct {
	database   string
	collection string
	indexName  string
}

func parseIndexId(path string) (*indexId, error) {
	splitPath := strings.Split(path, ".")
	if len(splitPath) != 3 {
		return nil, errors.New("Index id's format must be <database>.<collection>.<index_name>")
	}

	return &indexId{database: splitPath[0], collection: splitPath[1], indexName: splitPath[2]}, nil
}

func (co *collation) toMongoCollation() *options.Collation {
	if co == nil {
		return nil
	}

	res := options.Collation{}

	res.Locale = co.Locale
	if co.CaseLevel != nil {
		res.CaseLevel = *co.CaseLevel
	}
	if co.CaseFirst != nil {
		res.CaseFirst = *co.CaseFirst
	}
	if co.Strength != nil {
		res.Strength = *co.Strength
	}
	if co.NumericOrdering != nil {
		res.NumericOrdering = *co.NumericOrdering
	}
	if co.Alternate != nil {
		res.Alternate = *co.Alternate
	}
	if co.MaxVariable != nil {
		res.MaxVariable = *co.MaxVariable
	}
	if co.Normalization != nil {
		res.Normalization = *co.Normalization
	}
	if co.Backwards != nil {
		res.Backwards = *co.Backwards
	}
	return &res
}

func addArgs(arguments string, newArg string) string {
	if arguments != "" {
		return arguments + "&" + newArg
	} else {
		return "/?" + newArg
	}

}

// caPEM   – optional CA certificate(s) in PEM format
// certPEM – optional client certificate in PEM format
// keyPEM  – optional client private key in PEM format
// insecureSkipVerify – true disables server name verification
func getTLSConfigWithAllServerCertificates(
	caPEM, certPEM, keyPEM []byte,
	insecureSkipVerify bool,
) (*tls.Config, error) {

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	// --- Handle CA certificates (optional) ---
	if len(caPEM) > 0 {
		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(caPEM); !ok {
			return nil, errors.New("failed parsing CA PEM file")
		}
		tlsConfig.RootCAs = rootCAs
	}

	// --- Handle client certificate (optional) ---
	if len(certPEM) > 0 && len(keyPEM) > 0 {
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func proxyDialer(proxyUrlFromProvider string) (options.ContextDialer, error) {
	if proxyUrlFromProvider != "" {
		proxyURL, err := url.Parse(proxyUrlFromProvider)
		if err != nil {
			return nil, err
		}
		proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return proxyDialer.(options.ContextDialer), nil
	}

	return nil, nil
}
