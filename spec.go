package sqs

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
)

// version is set during the build process with ldflags (see Makefile).
// Default version matches default from runtime/debug.
var version = "(devel)"

// Specification returns the Plugin's Specification.
func Specification() sdk.Specification {
	return sdk.Specification{
		Name:    "sqs",
		Summary: "The Amazon SQS plugin for Conduit, written in Go.",
		Description: "The Amazon SQS connector is one of Conduit plugins." +
			"It provides the source and destination connector.",
		Version: version,
		Author:  "Meroxa, Inc.",
	}
}
