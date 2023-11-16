package source

//go:generate paramgen -output=paramgen.go Config

type Config struct {
	// amazon access key id
	AWSAccessKeyID string `json:"aws.accessKeyId" validate:"required"`
	// amazon secret access key
	AWSSecretAccessKey string `json:"aws.secretAccessKey" validate:"required"`
	// amazon sqs region
	AWSRegion string `json:"aws.region" validate:"required"`
	// amazon sqs queue name
	AWSQueue string `json:"aws.queue" validate:"required"`
	// visibility timeout
	AWSSQSVisibilityTimeout int32 `json:"aws.visibilityTimeout"`
}

const (
	ConfigKeyAWSAccessKeyID = "aws.accessKeyId"

	ConfigKeyAWSSecretAccessKey = "aws.secretAccessKey"

	ConfigKeyAWSRegion = "aws.region"

	ConfigKeyAWSQueue = "aws.queue"

	ConfigKeySQSVisibilityTimeout = "aws.visibilityTimeout"
)
