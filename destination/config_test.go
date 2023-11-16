package destination

import (
	"testing"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
)

var exampleConfig = map[string]string{
	"aws.accessKeyId":     "access-key-123",
	"aws.secretAccessKey": "secret-key-321",
	"aws.token":           "token_token",
}

func TestParseConfig(t *testing.T) {
	is := is.New(t)
	var got Config
	err := sdk.Util.ParseConfig(exampleConfig, &got)
	want := Config{
		AWSAccessKeyID:     "access-key-123",
		AWSSecretAccessKey: "secret-key-321",
		AWSToken:           "token_token",
	}

	is.NoErr(err)
	is.Equal(want, got)
}
