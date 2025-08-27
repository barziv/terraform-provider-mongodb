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

func getTLSConfigWithAllServerCertificates(ca []byte, verify bool) (*tls.Config, error) {
	/* As of version 1.2.1, the MongoDB Go Driver will only use the first CA server certificate found in sslcertificateauthorityfile.
	   The code below addresses this limitation by manually appending all server certificates found in sslcertificateauthorityfile
	   to a custom TLS configuration used during client creation. */

	tlsConfig := new(tls.Config)

	tlsConfig.InsecureSkipVerify = verify
	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(ca)

	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}

	return tlsConfig, nil
}

func proxyDialer(proxyUrlFromProvider string) (options.ContextDialer, error) {
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
