package sqs

import (
	"github.com/conduitio-labs/conduit-connector-sqs/destination"
	"github.com/conduitio-labs/conduit-connector-sqs/source"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

var Connector = sdk.Connector{
	NewSpecification: Specification,
	NewSource:        source.NewSource,
	NewDestination:   destination.NewDestination,
}
