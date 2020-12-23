package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

type CredentialConfig struct {
	Region      string
	AccessKey   string
	SecretKey   string
	RoleARN     string
	Profile     string
	Filename    string
	Token       string
	EndpointURL string
}

func (c *CredentialConfig) Credentials() (client.ConfigProvider, error) {
	if c.RoleARN != "" {
		return c.assumeCredentials()
	} else {
		return c.rootCredentials()
	}
}

func (c *CredentialConfig) rootCredentials() (client.ConfigProvider, error) {
	config := &aws.Config{
		Region: aws.String(c.Region),
	}
	if c.EndpointURL != "" {
		config.Endpoint = &c.EndpointURL
	}
	if c.AccessKey != "" || c.SecretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, c.Token)
	} else if c.Profile != "" || c.Filename != "" {
		config.Credentials = credentials.NewSharedCredentials(c.Filename, c.Profile)
	}

	return session.NewSession(config)
}

func (c *CredentialConfig) assumeCredentials() (client.ConfigProvider, error) {
	rootCredentials, err := c.rootCredentials()
	if err != nil {
		return nil, err
	}
	config := &aws.Config{
		Region:   aws.String(c.Region),
		Endpoint: &c.EndpointURL,
	}
	config.Credentials = stscreds.NewCredentials(rootCredentials, c.RoleARN)
	return session.NewSession(config)
}
