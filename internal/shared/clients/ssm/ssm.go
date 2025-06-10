package ssm

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type ParameterStore interface {
	GetParameter(ctx context.Context, key string) (string, error)
}

type AWSParameterStore struct {
	client *ssm.Client
}

func NewAWSParameterStore(client *ssm.Client) *AWSParameterStore {
	return &AWSParameterStore{client: client}
}

func (c *AWSParameterStore) GetParameter(ctx context.Context, key string) (string, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(true),
	}
	output, err := c.client.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error getting %s from SSM: %w", key, err)
	}
	value := output.Parameter.Value
	if value == nil {
		return "", fmt.Errorf("value of %s in SSM is nil", key)
	}
	return *value, nil
}
