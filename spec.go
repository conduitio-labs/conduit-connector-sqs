package sqs

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
)

// version is set during the build process (i.e. the Makefile).
// It follows Go's convention for module version, where the version
// starts with the letter v, followed by a semantic version.
var version = "v0.1.0-dev"

// Specification returns the Plugin's Specification.
func Specification() sdk.Specification {
	return sdk.Specification{
		Name:    "amazon_sqs",
		Summary: "The Amazon SQS plugin for Conduit, written in Go.",
		Description: "The Amazon SQS connector is one of Conduit plugins." +
			"It provides the source and destination connector.",
		Version: version,
		Author:  "Meroxa, Inc.",
	}
}
