package connectors_test

import "github.com/sipki-tech/database/connectors"

var (
	fullRawConfig = connectors.Raw{
		Query: "postgres://user:password@127.0.0.1:26257/defaultdb?application_name=application_name",
	}
)
