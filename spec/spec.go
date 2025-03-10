package spec

import (
	_ "embed"

	sdk "github.com/conduitio/conduit-connector-sdk"
)

//go:embed connector.yaml
var specs string

var version = "(devel)"

var Spec = sdk.YAMLSpecification(specs, version)
