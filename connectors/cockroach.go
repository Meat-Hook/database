package connectors

import (
	"encoding"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Meat-Hook/database"
)

var (
	_ yaml.Unmarshaler         = (*CockroachSSL)(nil)
	_ json.Unmarshaler         = (*CockroachSSL)(nil)
	_ encoding.TextUnmarshaler = (*CockroachSSL)(nil)
	_ database.Connector       = (*CockroachDB)(nil)
)

// CockroachSSL is a type for setting connection ssl mode to CockroachDB.
type CockroachSSL uint8

// UnmarshalJSON implements json.Unmarshaler.
func (i *CockroachSSL) UnmarshalJSON(b []byte) error {
	str := ""
	err := json.Unmarshal(b, &str)
	if err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	return i.UnmarshalText([]byte(str))
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (i *CockroachSSL) UnmarshalYAML(b *yaml.Node) error {
	return i.UnmarshalText([]byte(b.Value))
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *CockroachSSL) UnmarshalText(str []byte) error {
	switch string(str) {
	case CockroachSSLDisable.String():
		*i = CockroachSSLDisable
	case CockroachSSLAllow.String():
		*i = CockroachSSLAllow
	case CockroachSSLPrefer.String():
		*i = CockroachSSLPrefer
	case CockroachSSLRequire.String():
		*i = CockroachSSLRequire
	case CockroachSSLVerifyCa.String():
		*i = CockroachSSLVerifyCa
	case CockroachSSLVerifyFull.String():
		*i = CockroachSSLVerifyFull
	default:
		return fmt.Errorf("unknown mode: %s", str)
	}

	return nil
}

// Enum.
const (
	_                      CockroachSSL = iota
	CockroachSSLDisable                 // disable
	CockroachSSLAllow                   // allow
	CockroachSSLPrefer                  // prefer
	CockroachSSLRequire                 // require
	CockroachSSLVerifyCa                // verify-ca
	CockroachSSLVerifyFull              // verify-full
)

type (
	// CockroachDBVariable sets variable for connections.
	CockroachDBVariable struct {
		Name  string `yaml:"name" json:"name"`
		Value string `yaml:"value" json:"value"`
	}

	// CockroachDBOptions contains options for setting variables and cluster ID.
	CockroachDBOptions struct {
		Cluster  string              `yaml:"cluster" json:"cluster"`
		Variable CockroachDBVariable `yaml:"variable" json:"variable"`
	}

	// CockroachDBParameters contains url parameters for connecting to database.
	CockroachDBParameters struct {
		ApplicationName string       `yaml:"application_name" json:"application_name"`
		Mode            CockroachSSL `yaml:"mode" json:"mode" hcl:"mode"`
		SSLRootCert     string       `yaml:"ssl_root_cert" json:"ssl_root_cert"`
		SSLCert         string       `yaml:"ssl_cert" json:"ssl_cert"`
		SSLKey          string       `yaml:"ssl_key" json:"ssl_key"`

		// It isn't recommended, so it's disable. You must use CockroachDB.Password instead of it.
		// Password        string

		Options *CockroachDBOptions `yaml:"options" json:"options"`
	}

	// CockroachDB config for connecting to cockroachDB.
	CockroachDB struct {
		User       string                 `yaml:"user" json:"user"`
		Password   string                 `yaml:"password" json:"password"`
		Host       string                 `yaml:"host" json:"host"`
		Port       int                    `yaml:"port" json:"port"`
		Database   string                 `yaml:"database" json:"database"`
		Parameters *CockroachDBParameters `yaml:"parameters" json:"parameters"`

		// We don't have support for UNIX domain socket.
		// DirectoryPath string `yaml:"directory_path" json:"directory_path"`
	}
)

// DSN convert struct to DSN and returns connection string.
func (c CockroachDB) DSN() (string, error) {
	str := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)

	uri, err := url.Parse(str)
	if err != nil {
		return "", fmt.Errorf("url.Parse: %w", err)
	}

	if c.Parameters == nil {
		return uri.String(), nil
	}

	parameters := url.Values{}
	if c.Parameters.ApplicationName != "" {
		parameters.Add("application_name", c.Parameters.ApplicationName)
	}

	if c.Parameters.Mode != 0 {
		parameters.Add("sslmode", c.Parameters.Mode.String())
	}

	if c.Parameters.SSLRootCert != "" {
		parameters.Add("sslrootcert", c.Parameters.SSLRootCert)
	}

	if c.Parameters.SSLCert != "" {
		parameters.Add("sslcert", c.Parameters.SSLCert)
	}

	if c.Parameters.SSLKey != "" {
		parameters.Add("sslkey", c.Parameters.SSLKey)
	}

	uri.RawQuery = parameters.Encode()
	if c.Parameters.Options == nil {
		return uri.String(), nil
	}

	var options []string
	if c.Parameters.Options.Cluster != "" {
		options = append(options, fmt.Sprintf("--cluster=%s", c.Parameters.Options.Cluster))
	}

	if c.Parameters.Options.Variable.Name != "" {
		options = append(options, fmt.Sprintf("-c %s=%s", c.Parameters.Options.Variable.Name, c.Parameters.Options.Variable.Value))
	}

	parameters.Add("options", strings.Join(options, " "))
	uri.RawQuery = parameters.Encode()

	return uri.String(), nil
}
