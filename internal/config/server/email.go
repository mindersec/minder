package server

// EmailConfig is the configuration for the email sending service
type EmailConfig struct {
	// AWSSES is the AWS SES configuration
	AWSSES AWSSES `mapstructure:"aws_ses"`
}

// AWSSES is the AWS SES configuration
type AWSSES struct {
	// Sender is the email address of the sender
	Sender string `mapstructure:"sender"`
	// Region is the AWS region to use for AWS SES
	Region string `mapstructure:"region"`
}
