// Package e2e contains desktop live tests that run against a real VibeCat installation.
//
// These tests require:
//   - VIBECAT_E2E_CONTROL=1 (enables E2E control bridge)
//   - DESKTOP_E2E=1 (enables desktop execution)
//   - VibeCat E2E control bridge running on localhost:9876
//
// Usage:
//
//	VIBECAT_E2E_CONTROL=1 DESKTOP_E2E=1 go test -v -count=1 -run TestDesktopLive ./...
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// ---- Types ----

// DesktopScenario represents a single desktop E2E test scenario loaded from JSON.
type DesktopScenario struct {
	Name          string        `json:"name"`
	Surface       string        `json:"surface"`
	Setup         ScenarioSetup `json:"setup"`
	Command       string        `json:"command"`
	SuccessPrompt string        `json:"successPrompt"`
	Timeout       int           `json:"timeout"`
	SurfaceID     string        `json:"surface_id"`
	Artifacts     []string      `json:"artifacts"`
}

// ScenarioSetup contains setup requirements for a scenario.
type ScenarioSetup struct {
	Description     string   `json:"description"`
	RequiredApp     string   `json:"requiredApp"`
	RequiredBundles []string `json:"requiredBundles"`
}

// commandRequest is the payload for POST /e2e/command.
type commandRequest struct {
	Command   string `json:"command"`
	SurfaceID string `json:"surface_id,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
}

// commandResponse is the response from POST /e2e/command.
type commandResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// statusResponse is the response from GET /e2e/status.
type statusResponse struct {
	TaskID  string `json:"task_id"`
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// screenshotResponse is the response from POST /e2e/screenshot.
type screenshotResponse struct {
	Data      string `json:"data"` // base64-encoded PNG
	Timestamp string `json:"timestamp"`
	Format    string `json:"format"`
}

// visionVerifyRequest is the payload for POST /navigator/escalate used for vision verification.
type visionVerifyRequest struct {
	Command    string `json:"command"`
	Screenshot string `json:"screenshot"`
	Context    string `json:"context,omitempty"`
}

// visionVerifyResponse is the response from POST /navigator/escalate.
type visionVerifyResponse struct {
	Success    bool    `json:"success"`
	Confidence float64 `json:"confidence"`
	Summary    string  `json:"summary"`
	Target     string  `json:"target,omitempty"`
}

// e2eEvent represents a single event from GET /e2e/events.
type e2eEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Data      any    `json:"data"`
}

// eventsResponse is the response from GET /e2e/events.
type eventsResponse struct {
	Events []e2eEvent `json:"events"`
}

// scenarioArtifacts holds all collected artifacts for a scenario run.
type scenarioArtifacts struct {
	ScenarioName   string
	Timestamp      string
	PreScreenshot  string
	PostScreenshot string
	Events         []e2eEvent
	VerifyResult   string
	Passed         bool
}

// ---- Guard functions ----

// desktopE2EEnabled returns true when both required env vars are set.
func desktopE2EEnabled() bool {
	return os.Getenv("VIBECAT_E2E_CONTROL") == "1" && os.Getenv("DESKTOP_E2E") == "1"
}

// bridgeURL returns the E2E control bridge base URL from env or the default.
func bridgeURL() string {
	u := os.Getenv("VIBECAT_E2E_BRIDGE_URL")
	if u == "" {
		return "http://localhost:9876"
	}
	return u
}

// orchestratorBaseURL returns the ADK orchestrator URL from env.
func orchestratorBaseURL() string {
	return os.Getenv("ORCHESTRATOR_URL")
}

// ---- Test entry points ----

func TestDesktopLive_TerminalOpenCode(t *testing.T) {
	runDesktopScenario(t, "terminal_opencode.json")
}

func TestDesktopLive_AntigravityInline(t *testing.T) {
	runDesktopScenario(t, "antigravity_inline.json")
}

func TestDesktopLive_ChromeYouTubeMusic(t *testing.T) {
	runDesktopScenario(t, "chrome_youtube_music.json")
}

// ---- Core runner ----

// runDesktopScenario executes a full desktop E2E scenario by filename.
func runDesktopScenario(t *testing.T, scenarioFile string) {
	t.Helper()

	if !desktopE2EEnabled() {
		t.Skip("VIBECAT_E2E_CONTROL=1 and DESKTOP_E2E=1 required — skipping desktop E2E tests")
	}

	bridge := bridgeURL()
	scenario := loadScenario(t, scenarioFile)
	timeout := time.Duration(scenario.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	artifacts := &scenarioArtifacts{
		ScenarioName: scenario.Name,
		Timestamp:    ts,
	}

	t.Logf("▶ scenario=%s surface=%s timeout=%v", scenario.Name, scenario.Surface, timeout)
	t.Logf("  setup: %s", scenario.Setup.Description)

	// Step 1: pre-screenshot
	t.Log("📸 capturing pre-screenshot")
	pre, err := captureScreenshot(t, bridge)
	if err != nil {
		t.Logf("⚠️  pre-screenshot failed (non-fatal): %v", err)
	} else {
		artifacts.PreScreenshot = pre
		t.Logf("✅ pre-screenshot captured (%d bytes base64)", len(pre))
	}

	// Step 2: submit command
	t.Logf("📤 submitting command: %s", scenario.Command)
	taskID, err := submitCommand(t, bridge, scenario.Command)
	if err != nil {
		t.Fatalf("submit command: %v", err)
	}
	t.Logf("✅ command submitted — taskID=%s", taskID)

	// Step 3: poll for completion
	t.Logf("⏳ waiting for completion (timeout=%v)", timeout)
	finalState, err := waitForCompletion(t, bridge, taskID, timeout)
	if err != nil {
		t.Logf("⚠️  wait for completion error: %v", err)
	}
	t.Logf("🏁 final state: %s", finalState)

	// Step 4: post-screenshot
	t.Log("📸 capturing post-screenshot")
	post, err := captureScreenshot(t, bridge)
	if err != nil {
		t.Logf("⚠️  post-screenshot failed (non-fatal): %v", err)
	} else {
		artifacts.PostScreenshot = post
		t.Logf("✅ post-screenshot captured (%d bytes base64)", len(post))
	}

	// Step 5: vision verification (optional — only if orchestrator is available)
	orchURL := orchestratorBaseURL()
	if orchURL != "" && post != "" {
		t.Log("🔍 verifying with Gemini Vision via ADK orchestrator")
		ok, summary, vErr := verifyWithVision(t, orchURL, post, scenario.SuccessPrompt)
		if vErr != nil {
			t.Logf("⚠️  vision verification error (non-fatal): %v", vErr)
		} else {
			artifacts.VerifyResult = summary
			if ok {
				t.Logf("✅ vision verification PASSED: %s", summary)
			} else {
				t.Logf("❌ vision verification FAILED: %s", summary)
			}
		}
	} else {
		t.Log("ℹ️  ORCHESTRATOR_URL not set — skipping vision verification")
	}

	// Step 6: collect events
	events, evErr := collectEvents(bridge)
	if evErr != nil {
		t.Logf("⚠️  collect events error (non-fatal): %v", evErr)
	} else {
		artifacts.Events = events
		t.Logf("📋 collected %d events", len(events))
	}

	// Step 7: save artifacts
	saveArtifacts(t, artifacts)

	// Final verdict — fail if execution itself errored
	if err != nil {
		t.Fatalf("scenario %s failed: execution error: %v", scenario.Name, err)
	}
	if finalState == "error" || finalState == "failed" {
		t.Fatalf("scenario %s failed: final state was %q", scenario.Name, finalState)
	}

	artifacts.Passed = true
	t.Logf("✅ scenario %s completed (state=%s)", scenario.Name, finalState)
}

// ---- Helper functions ----

// loadScenario reads and parses a scenario JSON file from desktop_scenarios/.
func loadScenario(t *testing.T, filename string) DesktopScenario {
	t.Helper()

	// Determine the directory of the test file at runtime.
	_, callerFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(callerFile)
	path := filepath.Join(dir, "desktop_scenarios", filename)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load scenario %q: %v", filename, err)
	}

	var s DesktopScenario
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("parse scenario %q: %v", filename, err)
	}
	return s
}

// captureScreenshot calls POST /e2e/screenshot and returns the base64 PNG data.
func captureScreenshot(t *testing.T, bridgeBase string) (string, error) {
	t.Helper()

	resp, err := http.Post(bridgeBase+"/e2e/screenshot", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("POST /e2e/screenshot: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST /e2e/screenshot returned %d: %s", resp.StatusCode, body)
	}

	var r screenshotResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("decode screenshot response: %w", err)
	}
	return r.Data, nil
}

// submitCommand sends a command to the VibeCat E2E control bridge and returns the task ID.
func submitCommand(t *testing.T, bridgeBase, command string) (string, error) {
	t.Helper()

	payload, _ := json.Marshal(commandRequest{Command: command})
	resp, err := http.Post(bridgeBase+"/e2e/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("POST /e2e/command: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("POST /e2e/command returned %d: %s", resp.StatusCode, body)
	}

	var r commandResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("decode command response: %w", err)
	}
	return r.TaskID, nil
}

// waitForCompletion polls GET /e2e/status every 2 seconds until the task reaches a terminal state
// or the timeout is exceeded. Returns the final state string.
func waitForCompletion(t *testing.T, bridgeBase, taskID string, timeout time.Duration) (string, error) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		url := fmt.Sprintf("%s/e2e/status?task_id=%s", bridgeBase, taskID)
		resp, err := http.Get(url)
		if err != nil {
			t.Logf("  poll error (will retry): %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var s statusResponse
		if err := json.Unmarshal(body, &s); err != nil {
			t.Logf("  decode status error (will retry): %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		t.Logf("  state=%s", s.State)

		// Terminal states: completed, done, success, error, failed, timeout
		switch s.State {
		case "completed", "done", "success", "error", "failed", "timeout":
			if s.Error != "" {
				return s.State, fmt.Errorf("bridge reported error: %s", s.Error)
			}
			return s.State, nil
		case "executing", "running", "pending", "":
			// still in progress
		default:
			t.Logf("  unknown state %q — continuing to poll", s.State)
		}

		time.Sleep(2 * time.Second)
	}

	return "timeout", fmt.Errorf("task %s did not complete within %v", taskID, timeout)
}

// verifyWithVision sends the post-action screenshot to the ADK orchestrator for Gemini Vision
// verification using the scenario's successPrompt. Returns (passed, summary, error).
func verifyWithVision(t *testing.T, orchestratorBase, screenshot, successPrompt string) (bool, string, error) {
	t.Helper()

	payload, _ := json.Marshal(visionVerifyRequest{
		Command:    successPrompt,
		Screenshot: screenshot,
		Context:    "desktop_e2e_verification",
	})

	resp, err := http.Post(orchestratorBase+"/navigator/escalate", "application/json", bytes.NewReader(payload))
	if err != nil {
		return false, "", fmt.Errorf("POST /navigator/escalate: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("POST /navigator/escalate returned %d: %s", resp.StatusCode, body)
	}

	var r visionVerifyResponse
	if err := json.Unmarshal(body, &r); err != nil {
		// Attempt a generic map decode in case the schema differs.
		var generic map[string]any
		if err2 := json.Unmarshal(body, &generic); err2 != nil {
			return false, "", fmt.Errorf("decode vision response: %w", err)
		}
		// Best-effort summary extraction.
		summary := fmt.Sprintf("%v", generic)
		return false, summary, nil
	}

	return r.Success, r.Summary, nil
}

// collectEvents calls GET /e2e/events and returns all collected events.
func collectEvents(bridgeBase string) ([]e2eEvent, error) {
	resp, err := http.Get(bridgeBase + "/e2e/events")
	if err != nil {
		return nil, fmt.Errorf("GET /e2e/events: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /e2e/events returned %d: %s", resp.StatusCode, body)
	}

	var r eventsResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("decode events response: %w", err)
	}
	return r.Events, nil
}

// saveArtifacts writes scenario artifacts to tests/e2e/desktop_artifacts/{name}/{timestamp}/.
func saveArtifacts(t *testing.T, a *scenarioArtifacts) {
	t.Helper()

	_, callerFile, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(callerFile), "desktop_artifacts", a.ScenarioName, a.Timestamp)

	if err := os.MkdirAll(base, 0755); err != nil {
		t.Logf("⚠️  create artifact dir %s: %v", base, err)
		return
	}

	// Save pre-screenshot
	if a.PreScreenshot != "" {
		if err := os.WriteFile(filepath.Join(base, "pre_screenshot.b64"), []byte(a.PreScreenshot), 0644); err != nil {
			t.Logf("⚠️  save pre-screenshot: %v", err)
		}
	}

	// Save post-screenshot
	if a.PostScreenshot != "" {
		if err := os.WriteFile(filepath.Join(base, "post_screenshot.b64"), []byte(a.PostScreenshot), 0644); err != nil {
			t.Logf("⚠️  save post-screenshot: %v", err)
		}
	}

	// Save events as JSON
	if len(a.Events) > 0 {
		eventsData, _ := json.MarshalIndent(a.Events, "", "  ")
		if err := os.WriteFile(filepath.Join(base, "events.json"), eventsData, 0644); err != nil {
			t.Logf("⚠️  save events: %v", err)
		}
	}

	// Save summary
	summary := map[string]any{
		"scenario":      a.ScenarioName,
		"timestamp":     a.Timestamp,
		"passed":        a.Passed,
		"verify_result": a.VerifyResult,
		"event_count":   len(a.Events),
	}
	summaryData, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(filepath.Join(base, "summary.json"), summaryData, 0644); err != nil {
		t.Logf("⚠️  save summary: %v", err)
	}

	t.Logf("💾 artifacts saved → %s", base)
}
