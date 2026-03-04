package secrets

import (
	"context"
	"os"
	"testing"
)

// skipIfNoGCPCredentials skips the test if no GCP credentials are available.
func skipIfNoGCPCredentials(t *testing.T) {
	// Check for GCP credentials environment variables
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" && os.Getenv("GCP_PROJECT") == "" {
		t.Skip("Skipping test: no GCP credentials available (set GOOGLE_APPLICATION_CREDENTIALS or GCP_PROJECT)")
	}
}

// TestLoadSecret_Integration tests loading a secret from GCP Secret Manager.
// This test requires GCP credentials to be configured.
func TestLoadSecret_Integration(t *testing.T) {
	skipIfNoGCPCredentials(t)

	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		projectID = "vibecat-489105" // default project
	}

	// This test assumes vibecat-gateway-auth-secret exists and has at least one version
	secretName := "vibecat-gateway-auth-secret"

	_, err := LoadSecret(ctx, projectID, secretName)
	if err != nil {
		t.Fatalf("LoadSecret failed: %v", err)
	}
}

// TestLoadSecretWithVersion_Integration tests loading a specific version of a secret.
// This test requires GCP credentials to be configured.
func TestLoadSecretWithVersion_Integration(t *testing.T) {
	skipIfNoGCPCredentials(t)

	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		projectID = "vibecat-489105" // default project
	}

	// This test assumes vibecat-gateway-auth-secret exists and has at least one version
	secretName := "vibecat-gateway-auth-secret"
	version := "latest"

	_, err := LoadSecretWithVersion(ctx, projectID, secretName, version)
	if err != nil {
		t.Fatalf("LoadSecretWithVersion failed: %v", err)
	}
}

// TestLoadSecret_InvalidProject tests error handling for invalid project.
func TestLoadSecret_InvalidProject(t *testing.T) {
	skipIfNoGCPCredentials(t)

	ctx := context.Background()
	// Use a non-existent project ID
	projectID := "non-existent-project-12345"
	secretName := "test-secret"

	_, err := LoadSecret(ctx, projectID, secretName)
	if err == nil {
		t.Fatal("Expected error for invalid project, got nil")
	}
}

// TestLoadSecret_InvalidSecret tests error handling for non-existent secret.
func TestLoadSecret_InvalidSecret(t *testing.T) {
	skipIfNoGCPCredentials(t)

	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		projectID = "vibecat-489105" // default project
	}

	// Use a non-existent secret name
	secretName := "non-existent-secret-xyz123"

	_, err := LoadSecret(ctx, projectID, secretName)
	if err == nil {
		t.Fatal("Expected error for non-existent secret, got nil")
	}
}
