package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestVoiceInputTranscriptionRoundTrip(t *testing.T) {
	if os.Getenv("E2E_VOICE_INPUT") == "" {
		t.Skip("E2E_VOICE_INPUT not set — skipping synthesized voice input test")
	}
	if _, err := exec.LookPath("say"); err != nil {
		t.Skip("macOS 'say' command not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("'ffmpeg' command not available")
	}

	base := gatewayURL(t)
	token := registerToken(t, base)
	conn := dialWS(t, base, token)
	defer conn.Close()

	sendSetup(t, conn)
	waitForSetupComplete(t, conn)

	events := make(chan wsEvent, 2048)
	readErrs := make(chan error, 1)
	go readWSEvents(conn, events, readErrs)

	pcmPath := synthesizeSpeechPCM(t, "이 터미널에 A부터 Z까지 다시 입력해 줘")
	pcm, err := os.ReadFile(pcmPath)
	if err != nil {
		t.Fatalf("read synthesized pcm: %v", err)
	}

	if err := streamPCM(conn, pcm, 80*time.Millisecond); err != nil {
		t.Fatalf("stream synthesized speech: %v", err)
	}
	if err := streamPCM(conn, make([]byte, 3200), 80*time.Millisecond, 12); err != nil {
		t.Fatalf("stream trailing silence: %v", err)
	}

	transcript, err := waitForFinishedInputTranscription(events, readErrs, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("✅ synthesized inputTranscription: %s", transcript)
}

func waitForFinishedInputTranscription(events <-chan wsEvent, errs <-chan error, timeout time.Duration) (string, error) {
	deadline := time.After(timeout)
	for {
		select {
		case err := <-errs:
			if err != nil {
				return "", fmt.Errorf("websocket read failed during voice input test: %w", err)
			}
		case event, ok := <-events:
			if !ok {
				return "", fmt.Errorf("event stream closed before inputTranscription finished")
			}
			if event.msgType != websocket.TextMessage {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal(event.data, &msg); err != nil {
				continue
			}
			if msg["type"] != "inputTranscription" {
				continue
			}
			finished, _ := msg["finished"].(bool)
			if !finished {
				continue
			}
			text, _ := msg["text"].(string)
			if text == "" {
				return "", fmt.Errorf("finished inputTranscription was empty")
			}
			return text, nil
		case <-deadline:
			return "", fmt.Errorf("timed out waiting for finished inputTranscription")
		}
	}
}
