package test

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/stretchr/testify/require"
)

const DefaultMinIOPort = "9000"

// DefaultMinIOHost is set to 127.0.0.1 instead of localhost because I had trouble
// running tests locally after connecting and disconnecting from VPN. DNS failed for
// localhost
const DefaultMinIOHost = "127.0.0.1"

// defaultMinIOS3Client can be shared. Get with DefaultMinIOS3Client
var defaultMinIOS3Client *s3.Client

func DefaultMinIOS3Client(ctx context.Context, t require.TestingT) *s3.Client {
	if defaultMinIOS3Client == nil {
		defaultMinIOS3Client = DockerMinIOSettings.Load(t).NewS3Client(ctx, t)
	}
	return defaultMinIOS3Client
}

type MinIOConfig struct {
	User     string
	Password string
	Host     string
	Port     int
}

func (c MinIOConfig) NewS3Client(ctx context.Context, t require.TestingT) *s3.Client {
	// Putting the credentials provider at the config level rather than service client level in case
	// it ever gets refactored out, so we don't accidentally create a config that uses a dev's real credentials.
	testAWSConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.User, c.Password, "")),
		awsConfig.WithRegion("us-east-1"),
	)
	require.NoError(t, err)
	return s3.NewFromConfig(testAWSConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(fmt.Sprintf("http://%s:%d", c.Host, c.Port))
		// TODO: figure out how to use virtual-host style when host is not localhost, i.e.,
		// use virtual-host style when tests are running in Docker on CI. Couldn't get
		// Docker network to work with requests going to http://random-bucket-name.minio:9000.
		if c.Host != DefaultMinIOHost {
			options.UsePathStyle = true
		}
	})
}

type MinIOSettings struct {
	User     sharedconfig.EnvironmentSetting
	Password sharedconfig.EnvironmentSetting
	Host     sharedconfig.EnvironmentSetting
	Port     sharedconfig.EnvironmentSetting
}

func (s MinIOSettings) Load(t require.TestingT) MinIOConfig {
	user, err := s.User.Get()
	require.NoError(t, err)
	password, err := s.Password.Get()
	require.NoError(t, err)
	host, err := s.Host.Get()
	require.NoError(t, err)
	port, err := s.Port.GetInt()
	require.NoError(t, err)
	return MinIOConfig{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
	}
}

var DockerMinIOSettings = MinIOSettings{
	// User and Password match what it passed to minio container in docker-compose.test.yml
	User:     sharedconfig.NewEnvironmentSettingWithDefault("MINIO_ROOT_USER", "TestAWSKey"),
	Password: sharedconfig.NewEnvironmentSettingWithDefault("MINIO_ROOT_PASSWORD", "TestAWSSecret"),
	// Host will be set in the test image created by docker-compose.test.yml when running in CI. But when run
	// locally it is not set, so the default will be used
	Host: sharedconfig.NewEnvironmentSettingWithDefault("MINIO_HOST", DefaultMinIOHost),
	// Port is never set as an env var. Just uses default
	Port: sharedconfig.NewEnvironmentSettingWithDefault("MINIO_PORT", DefaultMinIOPort),
}
