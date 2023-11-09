package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
	sqs "github.com/meroxa/conduit-connector-amazon-sqs"
)

func main() {
	sdk.Serve(sqs.Connector)
}
