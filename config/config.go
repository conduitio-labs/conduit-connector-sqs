package config

type Config struct {
	AWSAccessKeyID     string `json:"aws.accessKeyId" validate:"required"`
	AWSSecretAccessKey string `json:"aws.secretAccessKey" validate:"required"`
	AWSToken           string `json:"aws.token"`
	AWSQueue           string `json:"aws.queue" validate:"required"`
}

const (
	ConfigKeyAWSAccessKeyID = "aws.accessKeyId"

	ConfigKeyAWSSecretAccessKey = "aws.secretAccessKey"

	ConfigKeyAWSToken = "aws.token"

	ConfigKeyAWSQueue = "aws.queue"
)

func ParseConfig(cfg map[string]string) (Config, error) {
	var parsed Config
	key, ok := cfg[ConfigKeyAWSAccessKeyID]
	if ok {
		parsed.AWSAccessKeyID = key
	}

	secret, ok := cfg[ConfigKeyAWSSecretAccessKey]
	if ok {
		parsed.AWSSecretAccessKey = secret
	}

	token, ok := cfg[ConfigKeyAWSToken]
	if ok {
		parsed.AWSToken = token
	}

	queue, ok := cfg[ConfigKeyAWSQueue]
	if ok {
		parsed.AWSQueue = queue
	}

	return parsed, nil
}
