package connectors

import (
	"encoding"
	"encoding/json"
	"fmt"
	"net/url"

	"gopkg.in/yaml.v3"

	"github.com/sipki-tech/database"
)

var (
	_ yaml.Unmarshaler         = (*PostgresSSL)(nil)
	_ json.Unmarshaler         = (*PostgresSSL)(nil)
	_ encoding.TextUnmarshaler = (*PostgresSSL)(nil)
	_ database.Connector       = (*PostgresDB)(nil)
)

// PostgresSSL is a type for setting connection ssl mode to PostgresDB.
type PostgresSSL uint8

// UnmarshalJSON implements json.Unmarshaler.
func (i *PostgresSSL) UnmarshalJSON(b []byte) error {
	str := ""
	err := json.Unmarshal(b, &str)
	if err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	return i.UnmarshalText([]byte(str))
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (i *PostgresSSL) UnmarshalYAML(b *yaml.Node) error {
	return i.UnmarshalText([]byte(b.Value))
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *PostgresSSL) UnmarshalText(str []byte) error {
	switch string(str) {
	case PostgresSSLDisable.String():
		*i = PostgresSSLDisable
	case PostgresSSLAllow.String():
		*i = PostgresSSLAllow
	case PostgresSSLPrefer.String():
		*i = PostgresSSLPrefer
	case PostgresSSLRequire.String():
		*i = PostgresSSLRequire
	case PostgresSSLVerifyCa.String():
		*i = PostgresSSLVerifyCa
	case PostgresSSLVerifyFull.String():
		*i = PostgresSSLVerifyFull
	default:
		return fmt.Errorf("unknown mode: %s", str)
	}

	return nil
}

// Enum.
const (
	_                     PostgresSSL = iota
	PostgresSSLDisable                // disable
	PostgresSSLAllow                  // allow
	PostgresSSLPrefer                 // prefer
	PostgresSSLRequire                // require
	PostgresSSLVerifyCa               // verify-ca
	PostgresSSLVerifyFull             // verify-full
)

type (
	// PostgresDBParameters contains url parameters for connecting to database.
	PostgresDBParameters struct {
		ApplicationName string      `yaml:"application_name" json:"application_name"`
		Mode            PostgresSSL `yaml:"mode" json:"mode" hcl:"mode"`
		SSLRootCert     string      `yaml:"ssl_root_cert" json:"ssl_root_cert"`
		SSLCert         string      `yaml:"ssl_cert" json:"ssl_cert"`
		SSLKey          string      `yaml:"ssl_key" json:"ssl_key"`
	}

	// PostgresDB config for connecting to postgresDB.
	PostgresDB struct {
		User       string                `yaml:"user" json:"user"`
		Password   string                `yaml:"password" json:"password"`
		Host       string                `yaml:"host" json:"host"`
		Port       int                   `yaml:"port" json:"port"`
		Database   string                `yaml:"database" json:"database"`
		Parameters *PostgresDBParameters `yaml:"parameters" json:"parameters"`
	}
)

// DSN convert struct to DSN and returns connection string.
func (p *PostgresDB) DSN() (string, error) {
	str := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.Database,
	)

	uri, err := url.Parse(str)
	if err != nil {
		return "", fmt.Errorf("url.Parse: %w", err)
	}

	if p.Parameters == nil {
		return uri.String(), nil
	}

	parameters := url.Values{}
	if p.Parameters.ApplicationName != "" {
		parameters.Add("application_name", p.Parameters.ApplicationName)
	}

	if p.Parameters.Mode != 0 {
		parameters.Add("sslmode", p.Parameters.Mode.String())
	}

	if p.Parameters.SSLRootCert != "" {
		parameters.Add("sslrootcert", p.Parameters.SSLRootCert)
	}

	if p.Parameters.SSLCert != "" {
		parameters.Add("sslcert", p.Parameters.SSLCert)
	}

	if p.Parameters.SSLKey != "" {
		parameters.Add("sslkey", p.Parameters.SSLKey)
	}

	uri.RawQuery = parameters.Encode()
	return uri.String(), nil
}
