package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestVoiceBargeInRoundTrip(t *testing.T) {
	if os.Getenv("E2E_VOICE_BARGE_IN") == "" {
		t.Skip("E2E_VOICE_BARGE_IN not set — skipping voice barge-in smoke test")
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

	if err := conn.WriteJSON(map[string]any{
		"type": "clientContent",
		"clientContent": map[string]any{
			"turnComplete": true,
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]string{
						{"text": "하나부터 스무까지 천천히 세어 줘."},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send prompt: %v", err)
	}

	if err := waitForAssistantSpeechStart(events, readErrs, 15*time.Second); err != nil {
		t.Fatal(err)
	}

	pcmPath := synthesizeSpeechPCM(t, "stop and answer me")
	pcm, err := os.ReadFile(pcmPath)
	if err != nil {
		t.Fatalf("read synthesized pcm: %v", err)
	}

	if err := streamPCM(conn, pcm, 100*time.Millisecond); err != nil {
		t.Fatalf("stream synthesized speech: %v", err)
	}
	if err := streamPCM(conn, bytes.Repeat([]byte{0}, 3200), 100*time.Millisecond, 12); err != nil {
		t.Fatalf("stream trailing silence: %v", err)
	}

	if err := waitForBargeInRecovery(events, readErrs, 20*time.Second); err != nil {
		t.Fatal(err)
	}
}

type wsEvent struct {
	msgType int
	data    []byte
}

func sendSetup(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	setup := map[string]any{
		"type": "setup",
		"config": map[string]any{
			"voice":    "Zephyr",
			"language": "ko",
		},
	}
	if err := conn.WriteJSON(setup); err != nil {
		t.Fatalf("send setup: %v", err)
	}
}

func waitForSetupComplete(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read setup response: %v", err)
		}
		if msgType != websocket.TextMessage {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		if msg["type"] == "setupComplete" {
			return
		}
		if msg["type"] == "error" {
			t.Fatalf("setup returned error: %s", payload)
		}
	}
}

func readWSEvents(conn *websocket.Conn, events chan<- wsEvent, errs chan<- error) {
	defer close(events)
	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			select {
			case errs <- err:
			default:
			}
			return
		}
		events <- wsEvent{msgType: msgType, data: payload}
	}
}

func waitForAssistantSpeechStart(events <-chan wsEvent, errs <-chan error, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		select {
		case err := <-errs:
			if err != nil {
				return fmt.Errorf("websocket read failed before assistant speech: %w", err)
			}
		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("event stream closed before assistant speech")
			}
			if event.msgType == websocket.BinaryMessage && len(event.data) > 0 {
				return nil
			}
			if event.msgType != websocket.TextMessage {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal(event.data, &msg); err != nil {
				continue
			}
			if msg["type"] == "turnState" && msg["state"] == "speaking" {
				return nil
			}
		case <-deadline:
			return fmt.Errorf("timed out waiting for assistant speech start")
		}
	}
}

func waitForBargeInRecovery(events <-chan wsEvent, errs <-chan error, timeout time.Duration) error {
	deadline := time.After(timeout)
	sawInterrupted := false
	sawInputFinished := false

	for {
		select {
		case err := <-errs:
			if err != nil {
				return fmt.Errorf("websocket read failed during barge-in recovery: %w", err)
			}
		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("event stream closed during barge-in recovery")
			}
			if event.msgType == websocket.BinaryMessage {
				if sawInputFinished && len(event.data) > 0 {
					return nil
				}
				continue
			}
			if event.msgType != websocket.TextMessage {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal(event.data, &msg); err != nil {
				continue
			}
			switch msg["type"] {
			case "interrupted":
				sawInterrupted = true
			case "inputTranscription":
				finished, _ := msg["finished"].(bool)
				if sawInterrupted && finished {
					sawInputFinished = true
				}
			case "turnState":
				if sawInputFinished && msg["state"] == "speaking" {
					return nil
				}
			}
		case <-deadline:
			return fmt.Errorf("timed out waiting for barge-in recovery (interrupted=%t inputFinished=%t)", sawInterrupted, sawInputFinished)
		}
	}
}

func synthesizeSpeechPCM(t *testing.T, phrase string) string {
	t.Helper()
	dir := t.TempDir()
	aiffPath := filepath.Join(dir, "utterance.aiff")
	pcmPath := filepath.Join(dir, "utterance.pcm")

	cmd := exec.Command("say", "-o", aiffPath, phrase)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate speech with say: %v (%s)", err, output)
	}

	ffmpeg := exec.Command(
		"ffmpeg",
		"-y",
		"-i", aiffPath,
		"-ac", "1",
		"-ar", "16000",
		"-f", "s16le",
		pcmPath,
	)
	if output, err := ffmpeg.CombinedOutput(); err != nil {
		t.Fatalf("convert synthesized speech to pcm: %v (%s)", err, output)
	}

	return pcmPath
}

func streamPCM(conn *websocket.Conn, pcm []byte, chunkDelay time.Duration, repeat ...int) error {
	count := 1
	if len(repeat) > 0 {
		count = repeat[0]
	}
	for i := 0; i < count; i++ {
		data := pcm
		for len(data) > 0 {
			chunkSize := 3200
			if len(data) < chunkSize {
				chunkSize = len(data)
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, data[:chunkSize]); err != nil {
				return err
			}
			data = data[chunkSize:]
			time.Sleep(chunkDelay)
		}
	}
	return nil
}
