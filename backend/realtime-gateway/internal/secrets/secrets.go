// Package secrets provides utilities for loading secrets from GCP Secret Manager.
package secrets

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// LoadSecret loads the latest version of a secret from GCP Secret Manager.
// projectID is the GCP project ID (e.g., "vibecat-489105").
// secretName is the secret name (e.g., "vibecat-gemini-api-key").
func LoadSecret(ctx context.Context, projectID, secretName string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("secretmanager.NewClient: %w", err)
	}
	defer client.Close()

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("AccessSecretVersion(%s): %w", name, err)
	}

	return string(result.Payload.Data), nil
}

// LoadSecretWithVersion loads a specific version of a secret from GCP Secret Manager.
// Use "latest" for the latest version.
func LoadSecretWithVersion(ctx context.Context, projectID, secretName, version string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("secretmanager.NewClient: %w", err)
	}
	defer client.Close()

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretName, version)
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("AccessSecretVersion(%s): %w", name, err)
	}

	return string(result.Payload.Data), nil
}
