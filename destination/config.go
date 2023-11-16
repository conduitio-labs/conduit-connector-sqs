package destination

//go:generate paramgen -output=paramgen.go Config

type Config struct {
	// amazon access key id
	AWSAccessKeyID string `json:"aws.accessKeyId" validate:"required"`
	// amazon secret access key
	AWSSecretAccessKey string `json:"aws.secretAccessKey" validate:"required"`
	// amazon access token
	AWSToken string `json:"aws.token"`
	// amazon sqs queue name
	AWSQueue string `json:"aws.queue" validate:"required"`
}

const (
	ConfigKeyAWSAccessKeyID = "aws.accessKeyId"

	ConfigKeyAWSSecretAccessKey = "aws.secretAccessKey"

	ConfigKeyAWSToken = "aws.token"

	ConfigKeyAWSQueue = "aws.queue"
)
