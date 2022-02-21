package connectors

import (
	"fmt"
	"net/url"
)

// PostgresSSL is a type for setting connection ssl mode to PostgresDB.
type PostgresSSL uint8

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

type PostgresKeepalives int

const (
	PostgresKeepalivesOFF PostgresKeepalives = iota
	PostgresKeepalivesON
)

type PostgresSSLCompression int

const (
	PostgresSSLCompressionOFF PostgresSSLCompression = iota
	PostgresSSLCompressionON
)

type (
	// PostgresDB config for connecting to cockroachDB.
	PostgresDB struct {
		Host     string `yaml:"host" json:"host" hcl:"host"`
		HostAddr string `yaml:"hostaddr" json:"hostaddr" hcl:"hostaddr"`
		Port     int    `yaml:"port" json:"port" hcl:"port"`
		DBName   string `yaml:"dbname" json:"dbname" hcl:"dbname"`
		User     string `yaml:"user" json:"user" hcl:"user"`
		Password string `yaml:"password" json:"password" hcl:"password"`

		Parameters *PostgresDBParameters `yaml:"parameters" json:"parameters" hcl:"parameters,block"`
	}

	// PostgresDBParameters contains url parameters for connecting to database.
	PostgresDBParameters struct {
		Mode                    PostgresSSL            `yaml:"mode" json:"mode" hcl:"mode"`
		SSLCompression          PostgresSSLCompression `yaml:"sslcompression" json:"sslcompression" hcl:"sslcompression"`
		SSLCert                 string                 `yaml:"ssl_cert" json:"ssl_cert" hcl:"ssl_cert"`
		SSLKey                  string                 `yaml:"ssl_key" json:"ssl_key" hcl:"ssl_key"`
		SSLRootCert             string                 `yaml:"ssl_root_cert" json:"ssl_root_cert" hcl:"ssl_root_cert"`
		SSLCrl                  string                 `yaml:"ssl_crl" json:"ssl_crl" hcl:"ssl_crl"`
		ConnectTimeout          uint                   `yaml:"connect_timeout" json:"connect_timeout" hcl:"connect_timeout"`
		ClientEncoding          string                 `yaml:"client_encoding" json:"client_encoding" hcl:"client_encoding"`
		ApplicationName         string                 `yaml:"application_name" json:"application_name" hcl:"application_name"`
		FallbackApplicationName string                 `yaml:"fallback_application_name" json:"fallback_application_name" hcl:"fallback_application_name"`
		Keepalives              PostgresKeepalives     `yaml:"keepalives" json:"keepalives" hcl:"keepalives"`
		KeepalivesIdle          uint                   `yaml:"keepalives_idle" json:"keepalives_idle" hcl:"keepalives_idle"`
		KeepalivesInterval      uint                   `yaml:"keepalives_interval" json:"keepalives_interval" hcl:"keepalives_interval"`
		KeepalivesCount         uint                   `yaml:"keepalives_count" json:"keepalives_count" hcl:"keepalives_count"`
		RequirePeer             string                 `yaml:"requirepeer" json:"requirepeer" hcl:"requirepeer"`
		KrbSrvName              string                 `yaml:"krbsrvname" json:"krbsrvname" hcl:"krbsrvname"`
		GSSLib                  string                 `yaml:"gsslib" json:"gsslib" hcl:"gsslib"`
		Service                 string                 `yaml:"service" json:"service" hcl:"service"`

		Other map[string]string // Any other parameters from https://www.postgresql.org/docs/current/runtime-config-client.html.
	}
)

// DSN convert struct to DSN and returns connection string.
func (p PostgresDB) DSN() (string, error) {
	str := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.DBName,
	)

	uri, err := url.Parse(str)
	if err != nil {
		return "", fmt.Errorf("url.Parse: %w", err)
	}

	if p.Parameters == nil {
		return uri.String(), nil
	}

	parameters := url.Values{}

	if p.Parameters.Mode != 0 {
		parameters.Add("sslmode", p.Parameters.Mode.String())
	}

	if p.Parameters.SSLCompression != 0 {
		parameters.Add("sslcompression", fmt.Sprintf("%d", p.Parameters.SSLCompression))
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

	if p.Parameters.SSLCrl != "" {
		parameters.Add("sslcrl", p.Parameters.SSLCrl)
	}

	if p.Parameters.ConnectTimeout != 0 {
		parameters.Add("connect_timeout", fmt.Sprintf("%d", p.Parameters.ConnectTimeout))
	}

	if p.Parameters.ClientEncoding != "" {
		parameters.Add("client_encoding", p.Parameters.ClientEncoding)
	}

	if p.Parameters.ApplicationName != "" {
		parameters.Add("application_name", p.Parameters.ApplicationName)
	}

	if p.Parameters.FallbackApplicationName != "" {
		parameters.Add("fallback_application_name", p.Parameters.FallbackApplicationName)
	}

	if p.Parameters.Keepalives != 0 {
		parameters.Add("keepalives", fmt.Sprintf("%d", p.Parameters.Keepalives))
	}

	if p.Parameters.KeepalivesIdle != 0 {
		parameters.Add("keepalives_idle", fmt.Sprintf("%d", p.Parameters.KeepalivesIdle))
	}

	if p.Parameters.KeepalivesInterval != 0 {
		parameters.Add("keepalives_interval", fmt.Sprintf("%d", p.Parameters.KeepalivesInterval))
	}

	if p.Parameters.KeepalivesCount != 0 {
		parameters.Add("keepalives_count", fmt.Sprintf("%d", p.Parameters.KeepalivesCount))
	}

	if p.Parameters.RequirePeer != "" {
		parameters.Add("requirepeer", p.Parameters.RequirePeer)
	}

	if p.Parameters.KrbSrvName != "" {
		parameters.Add("krbsrvname", p.Parameters.KrbSrvName)
	}

	if p.Parameters.GSSLib != "" {
		parameters.Add("gsslib", p.Parameters.GSSLib)
	}

	if p.Parameters.Service != "" {
		parameters.Add("service", p.Parameters.Service)
	}

	if p.Parameters.Other != nil {
		for param, value := range p.Parameters.Other {
			parameters.Add(fmt.Sprintf("%s", param), value)
		}
	}

	uri.RawQuery = parameters.Encode()

	return uri.String(), nil
}
