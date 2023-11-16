package main

import (
	sqs "github.com/conduitio-labs/conduit-connector-sqs"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

func main() {
	sdk.Serve(sqs.Connector)
}
