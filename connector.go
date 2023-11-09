package sqs

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/meroxa/conduit-connector-amazon-sqs/destination"
	"github.com/meroxa/conduit-connector-amazon-sqs/source"
)

var Connector = sdk.Connector{
	NewSpecification: Specification,
	NewSource:        source.NewSource,
	NewDestination:   destination.NewDestination,
}
