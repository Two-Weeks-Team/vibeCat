// videotest sends screenshot frames to Gemini Live API via SendRealtimeInput(Video)
// and SendClientContent, measuring response time + quality at various resolutions.
//
// Usage: go run ./cmd/videotest
//
// Requires GOOGLE_API_KEY or GOOGLE_GENAI_API_KEY env var.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"google.golang.org/genai"
	"vibecat/realtime-gateway/internal/geminiconfig"
)

// Models to try for text+image Live API tests.
// Native audio models do NOT support ModalityText output and are validated separately below.
var modelsToTry = []string{
	"gemini-live-2.5-flash-preview",
}

type testCase struct {
	name   string
	width  int
	height int
	tiles  string
	tokens int
}

var cases = []testCase{
	{"768x432 (1-tile 16:9)", 768, 432, "1x1", 258},
	{"768x480 (1-tile 16:10)", 768, 480, "1x1", 258},
	{"768x768 (1-tile square)", 768, 768, "1x1", 258},
	{"1024x576 (2x1 tile 16:9)", 1024, 576, "2x1", 516},
	{"1280x720 (2x1 tile HD)", 1280, 720, "2x1", 516},
	{"1536x864 (2x2 tile 16:9)", 1536, 864, "2x2", 1032},
}

type mediaResCase struct {
	name string
	mr   genai.MediaResolution
}

var mrCases = []mediaResCase{
	{"Unspecified", ""},
	{"LOW (64tok)", genai.MediaResolutionLow},
	{"MEDIUM (256tok)", genai.MediaResolutionMedium},
	{"HIGH (256tok+zoom)", genai.MediaResolutionHigh},
}

func main() {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_GENAI_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("Set GOOGLE_API_KEY or GOOGLE_GENAI_API_KEY")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("genai client: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("=== PRE-PHASE: Listing models that support Live API (bidiGenerateContent) ===")
	fmt.Println(strings.Repeat("=", 80))

	var liveModels []string
	liveModels = listLiveModels(ctx, client)

	log.Println("\nCapturing screenshot...")
	srcImg := captureScreen()
	log.Printf("Screenshot: %dx%d", srcImg.Bounds().Dx(), srcImg.Bounds().Dy())

	// ============================================================
	// PHASE 0: Find a working model for text+image
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("=== PHASE 0: Model Discovery (find working model for text+image) ===")
	fmt.Println(strings.Repeat("=", 80))

	// Try explicit list first, then fallback to hardcoded candidates
	candidates := append(liveModels, modelsToTry...)
	testJPEG := encodeJPEG(resizeImage(srcImg, 768, 432), 70)
	var workingModel string
	tried := make(map[string]bool)

	for _, model := range candidates {
		// Strip "models/" prefix if present (ListModels returns "models/xxx")
		model = strings.TrimPrefix(model, "models/")
		if tried[model] {
			continue
		}
		tried[model] = true

		// Skip native-audio models (don't support ModalityText)
		if strings.Contains(model, "native-audio") {
			fmt.Printf("  ⏭️  %s: Skipping (native-audio, no text output)\n", model)
			continue
		}

		log.Printf("\n--- Trying model: %s ---", model)
		result := runLiveTest(ctx, client, model, testJPEG, "", 1)
		if result.err != nil {
			fmt.Printf("  ❌ %s: %v\n", model, result.err)
		} else if result.text == "" {
			fmt.Printf("  ⚠️  %s: Connected but empty response (latency: %dms)\n", model, result.latencyMs)
		} else {
			fmt.Printf("  ✅ %s: Latency=%dms | Response: %s\n", model, result.latencyMs, truncate(result.text, 100))
			workingModel = model
			break
		}
	}

	if workingModel == "" {
		log.Fatal("❌ No working model found! Check the model list above for available Live API models.")
	}
	fmt.Printf("\n🎯 Using model: %s\n", workingModel)

	// ============================================================
	// PHASE 1: Resolution Tests
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("=== PHASE 1: Resolution Tests (model=%s, MediaResolution=unspecified) ===\n", workingModel)
	fmt.Println(strings.Repeat("=", 80))

	for i, tc := range cases {
		resized := resizeImage(srcImg, tc.width, tc.height)
		jpegData := encodeJPEG(resized, 70)
		log.Printf("\n--- Test %d/%d: %s (JPEG: %dKB, expected %d tokens) ---",
			i+1, len(cases), tc.name, len(jpegData)/1024, tc.tokens)

		result := runLiveTest(ctx, client, workingModel, jpegData, "", 1)
		if result.err != nil {
			fmt.Printf("  ❌ ERROR: %v\n", result.err)
		} else if result.text == "" {
			fmt.Printf("  ⚠️  Empty response (latency: %dms)\n", result.latencyMs)
		} else {
			fmt.Printf("  ✅ Latency: %dms | Response: %s\n", result.latencyMs, truncate(result.text, 120))
		}
	}

	// ============================================================
	// PHASE 2: MediaResolution Tests
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("=== PHASE 2: MediaResolution Tests (768x432, model=%s) ===\n", workingModel)
	fmt.Println(strings.Repeat("=", 80))

	resized768 := resizeImage(srcImg, 768, 432)
	jpeg768 := encodeJPEG(resized768, 70)

	for i, mrc := range mrCases {
		log.Printf("\n--- Test %d/%d: MediaResolution=%s ---", i+1, len(mrCases), mrc.name)
		result := runLiveTest(ctx, client, workingModel, jpeg768, mrc.mr, 1)
		if result.err != nil {
			fmt.Printf("  ❌ ERROR: %v\n", result.err)
		} else if result.text == "" {
			fmt.Printf("  ⚠️  Empty response (latency: %dms)\n", result.latencyMs)
		} else {
			fmt.Printf("  ✅ Latency: %dms | Response: %s\n", result.latencyMs, truncate(result.text, 120))
		}
	}

	// ============================================================
	// PHASE 3: Stability Test (5 consecutive sends)
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("=== PHASE 3: Stability Test (768x432, 5 frames, model=%s) ===\n", workingModel)
	fmt.Println(strings.Repeat("=", 80))

	for i := 0; i < 5; i++ {
		log.Printf("\n--- Frame %d/5 ---", i+1)
		result := runLiveTest(ctx, client, workingModel, jpeg768, genai.MediaResolutionHigh, 1)
		if result.err != nil {
			fmt.Printf("  ❌ ERROR: %v\n", result.err)
		} else if result.text == "" {
			fmt.Printf("  ⚠️  Empty response (latency: %dms)\n", result.latencyMs)
		} else {
			fmt.Printf("  ✅ Latency: %dms | Response: %s\n", result.latencyMs, truncate(result.text, 120))
		}
		time.Sleep(500 * time.Millisecond)
	}

	// ============================================================
	// PHASE 4: Multi-frame session test
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("=== PHASE 4: Multi-frame session test (3 frames then ask, model=%s) ===\n", workingModel)
	fmt.Println(strings.Repeat("=", 80))

	result := runMultiFrameTest(ctx, client, workingModel, srcImg)
	if result.err != nil {
		fmt.Printf("  ❌ ERROR: %v\n", result.err)
	} else if result.text == "" {
		fmt.Printf("  ⚠️  Empty response (latency: %dms)\n", result.latencyMs)
	} else {
		fmt.Printf("  ✅ Total latency: %dms | Response: %s\n", result.latencyMs, truncate(result.text, 200))
	}

	// ============================================================
	// PHASE 5: SendRealtimeInput(Video) test — Fast Path validation
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("=== PHASE 5: SendRealtimeInput(Video) Fast Path test (model=%s) ===\n", workingModel)
	fmt.Println(strings.Repeat("=", 80))

	for _, res := range []struct{ w, h int }{{768, 432}, {1024, 576}} {
		resized := resizeImage(srcImg, res.w, res.h)
		jpegData := encodeJPEG(resized, 70)
		log.Printf("\n--- RealtimeInput Video %dx%d (%dKB) ---", res.w, res.h, len(jpegData)/1024)

		result := runRealtimeVideoTest(ctx, client, workingModel, jpegData)
		if result.err != nil {
			fmt.Printf("  ❌ ERROR: %v\n", result.err)
		} else if result.text == "" {
			fmt.Printf("  ⚠️  Empty response (latency: %dms)\n", result.latencyMs)
		} else {
			fmt.Printf("  ✅ Latency: %dms | Response: %s\n", result.latencyMs, truncate(result.text, 120))
		}
	}

	// ============================================================
	// PHASE 6: Native Audio model + Video (production Fast Path simulation)
	// ============================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("=== PHASE 6: Native Audio + Video (production simulation) ===")
	fmt.Println(strings.Repeat("=", 80))

	nativeAudioModel := geminiconfig.LiveNativeAudioModel
	log.Printf("Testing %s with SendRealtimeInput(Video)...", nativeAudioModel)

	result = runNativeAudioVideoTest(ctx, client, nativeAudioModel, jpeg768)
	if result.err != nil {
		fmt.Printf("  ❌ ERROR: %v\n", result.err)
	} else {
		fmt.Printf("  ✅ Audio response received in %dms (audioBytes=%d)\n", result.latencyMs, len(result.text))
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("=== ALL TESTS COMPLETE (text model=%s) ===\n", workingModel)
	fmt.Println(strings.Repeat("=", 80))
}

type testResult struct {
	latencyMs int64
	text      string
	err       error
}

func runLiveTest(ctx context.Context, client *genai.Client, model string, jpegData []byte, mr genai.MediaResolution, testNum int) testResult {
	sessionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	triggerTokens := int64(100000)
	targetTokens := int64(50000)

	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityText},
		ContextWindowCompression: &genai.ContextWindowCompressionConfig{
			TriggerTokens: &triggerTokens,
			SlidingWindow: &genai.SlidingWindow{TargetTokens: &targetTokens},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "You are analyzing a developer's screen. Describe what you see concisely in 1-2 sentences. Focus on: app name, visible text/code, any errors. Respond in Korean."}},
		},
	}
	if mr != "" {
		config.MediaResolution = mr
	}

	session, err := client.Live.Connect(sessionCtx, model, config)
	if err != nil {
		return testResult{err: fmt.Errorf("connect(%s): %w", model, err)}
	}
	defer session.Close()

	// Wait for setup
	msg, err := session.Receive()
	if err != nil {
		return testResult{err: fmt.Errorf("setup receive: %w", err)}
	}
	if msg.SetupComplete == nil {
		return testResult{err: fmt.Errorf("expected setupComplete, got: %+v", msg)}
	}

	start := time.Now()
	tc := true
	err = session.SendClientContent(genai.LiveClientContentInput{
		Turns: []*genai.Content{
			{
				Role: genai.RoleUser,
				Parts: []*genai.Part{
					{InlineData: &genai.Blob{Data: jpegData, MIMEType: "image/jpeg"}},
					{Text: "이 화면에서 무엇이 보이나요? 구체적으로 설명해주세요."},
				},
			},
		},
		TurnComplete: &tc,
	})
	if err != nil {
		return testResult{err: fmt.Errorf("send content: %w", err)}
	}

	return collectResponse(session, start)
}

// runRealtimeVideoTest uses SendRealtimeInput(Video) which is the Fast Path approach.
// After sending the video frame, we send a text prompt via SendClientContent to trigger response.
func runRealtimeVideoTest(ctx context.Context, client *genai.Client, model string, jpegData []byte) testResult {
	sessionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	triggerTokens := int64(100000)
	targetTokens := int64(50000)

	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityText},
		MediaResolution:    genai.MediaResolutionHigh,
		ContextWindowCompression: &genai.ContextWindowCompressionConfig{
			TriggerTokens: &triggerTokens,
			SlidingWindow: &genai.SlidingWindow{TargetTokens: &targetTokens},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "You are analyzing a developer's screen. Describe what you see concisely in 1-2 sentences. Focus on: app name, visible text/code, any errors. Respond in Korean."}},
		},
	}

	session, err := client.Live.Connect(sessionCtx, model, config)
	if err != nil {
		return testResult{err: fmt.Errorf("connect(%s): %w", model, err)}
	}
	defer session.Close()

	// Wait for setup
	msg, err := session.Receive()
	if err != nil {
		return testResult{err: fmt.Errorf("setup receive: %w", err)}
	}
	if msg.SetupComplete == nil {
		return testResult{err: fmt.Errorf("expected setupComplete, got: %+v", msg)}
	}

	// Send video frame via SendRealtimeInput
	err = session.SendRealtimeInput(genai.LiveRealtimeInput{
		Video: &genai.Blob{
			Data:     jpegData,
			MIMEType: "image/jpeg",
		},
	})
	if err != nil {
		return testResult{err: fmt.Errorf("sendRealtimeInput video: %w", err)}
	}

	// Small delay for the model to process the video frame
	time.Sleep(200 * time.Millisecond)

	// Now send text prompt to trigger response about the video frame
	start := time.Now()
	tc := true
	err = session.SendClientContent(genai.LiveClientContentInput{
		Turns: []*genai.Content{
			{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{{Text: "방금 보낸 화면 캡처에서 무엇이 보이나요? 구체적으로 설명해주세요."}},
			},
		},
		TurnComplete: &tc,
	})
	if err != nil {
		return testResult{err: fmt.Errorf("send text after video: %w", err)}
	}

	return collectResponse(session, start)
}

func runMultiFrameTest(ctx context.Context, client *genai.Client, model string, srcImg image.Image) testResult {
	sessionCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	triggerTokens := int64(100000)
	targetTokens := int64(50000)

	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityText},
		MediaResolution:    genai.MediaResolutionHigh,
		ContextWindowCompression: &genai.ContextWindowCompressionConfig{
			TriggerTokens: &triggerTokens,
			SlidingWindow: &genai.SlidingWindow{TargetTokens: &targetTokens},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "You analyze a developer's screen over time. Describe what changed between frames. Be specific about app name, code, errors. Respond in Korean."}},
		},
	}

	session, err := client.Live.Connect(sessionCtx, model, config)
	if err != nil {
		return testResult{err: fmt.Errorf("connect: %w", err)}
	}
	defer session.Close()

	msg, err := session.Receive()
	if err != nil || msg.SetupComplete == nil {
		return testResult{err: fmt.Errorf("setup: %v", err)}
	}

	resolutions := []struct{ w, h int }{{768, 432}, {768, 432}, {768, 432}}
	for i, res := range resolutions {
		resized := resizeImage(srcImg, res.w, res.h)
		jpegData := encodeJPEG(resized, 70)
		log.Printf("  Sending frame %d/3 (%dx%d, %dKB)...", i+1, res.w, res.h, len(jpegData)/1024)

		tcFalse := false
		err = session.SendClientContent(genai.LiveClientContentInput{
			Turns: []*genai.Content{{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{{InlineData: &genai.Blob{Data: jpegData, MIMEType: "image/jpeg"}}},
			}},
			TurnComplete: &tcFalse,
		})
		if err != nil {
			return testResult{err: fmt.Errorf("frame %d: %w", i+1, err)}
		}
		if i < len(resolutions)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	start := time.Now()
	tcTrue := true
	err = session.SendClientContent(genai.LiveClientContentInput{
		Turns: []*genai.Content{{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{{Text: "방금 보낸 화면 프레임들에서 무엇이 보이나요? 화면의 내용을 구체적으로 설명해주세요."}},
		}},
		TurnComplete: &tcTrue,
	})
	if err != nil {
		return testResult{err: fmt.Errorf("send text: %w", err)}
	}

	return collectResponse(session, start)
}

// collectResponse reads from the session until TurnComplete or timeout.
func collectResponse(session *genai.Session, start time.Time) testResult {
	var responseText string
	firstTokenTime := time.Duration(0)

	for {
		msg, err := session.Receive()
		if err != nil {
			if responseText != "" {
				break // Got some text, connection closed — acceptable
			}
			return testResult{err: fmt.Errorf("receive: %w", err)}
		}

		if msg.ServerContent != nil {
			if msg.ServerContent.ModelTurn != nil {
				for _, part := range msg.ServerContent.ModelTurn.Parts {
					if part.Text != "" {
						if firstTokenTime == 0 {
							firstTokenTime = time.Since(start)
						}
						responseText += part.Text
					}
				}
			}
			if msg.ServerContent.TurnComplete {
				break
			}
		}

		// Log non-content messages for debugging
		if msg.ServerContent == nil && msg.SetupComplete == nil {
			log.Printf("  [debug] Received non-content message: ToolCall=%v, UsageMetadata=%v",
				msg.ToolCall != nil, msg.UsageMetadata != nil)
		}
	}

	latency := firstTokenTime.Milliseconds()
	if latency == 0 {
		latency = time.Since(start).Milliseconds()
	}

	return testResult{latencyMs: latency, text: responseText}
}

func runNativeAudioVideoTest(ctx context.Context, client *genai.Client, model string, jpegData []byte) testResult {
	sessionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	triggerTokens := int64(100000)
	targetTokens := int64(50000)

	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
		MediaResolution:    genai.MediaResolutionMedium,
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{VoiceName: "Zephyr"},
			},
		},
		ContextWindowCompression: &genai.ContextWindowCompressionConfig{
			TriggerTokens: &triggerTokens,
			SlidingWindow: &genai.SlidingWindow{TargetTokens: &targetTokens},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "You are a desktop companion. Describe what you see on the developer's screen in 1 sentence. Respond in Korean."}},
		},
	}

	session, err := client.Live.Connect(sessionCtx, model, config)
	if err != nil {
		return testResult{err: fmt.Errorf("connect(%s): %w", model, err)}
	}
	defer session.Close()

	msg, err := session.Receive()
	if err != nil || msg.SetupComplete == nil {
		return testResult{err: fmt.Errorf("setup: %v", err)}
	}

	err = session.SendRealtimeInput(genai.LiveRealtimeInput{
		Video: &genai.Blob{Data: jpegData, MIMEType: "image/jpeg"},
	})
	if err != nil {
		return testResult{err: fmt.Errorf("sendVideo: %w", err)}
	}

	time.Sleep(200 * time.Millisecond)

	err = session.SendRealtimeInput(genai.LiveRealtimeInput{
		Text: "이 화면에서 뭐가 보여?",
	})
	if err != nil {
		return testResult{err: fmt.Errorf("sendText: %w", err)}
	}

	start := time.Now()
	var totalAudioBytes int
	firstAudioTime := time.Duration(0)
	for {
		msg, err = session.Receive()
		if err != nil {
			if totalAudioBytes > 0 {
				break
			}
			return testResult{err: fmt.Errorf("receive: %w", err)}
		}
		if msg.ServerContent != nil {
			if msg.ServerContent.ModelTurn != nil {
				for _, part := range msg.ServerContent.ModelTurn.Parts {
					if part.InlineData != nil && strings.HasPrefix(part.InlineData.MIMEType, "audio/") {
						if firstAudioTime == 0 {
							firstAudioTime = time.Since(start)
						}
						totalAudioBytes += len(part.InlineData.Data)
					}
				}
			}
			if msg.ServerContent.TurnComplete {
				break
			}
		}
	}

	latency := firstAudioTime.Milliseconds()
	if latency == 0 {
		latency = time.Since(start).Milliseconds()
	}
	return testResult{latencyMs: latency, text: fmt.Sprintf("%d bytes", totalAudioBytes)}
}

func listLiveModels(ctx context.Context, client *genai.Client) []string {
	var liveModels []string
	page, err := client.Models.List(ctx, nil)
	if err != nil {
		log.Printf("  list error: %v", err)
		return nil
	}
	for {
		for _, model := range page.Items {
			for _, method := range model.SupportedActions {
				if method == "bidiGenerateContent" {
					fmt.Printf("  LIVE: %-55s actions=%v\n", model.Name, model.SupportedActions)
					liveModels = append(liveModels, model.Name)
					break
				}
			}
		}
		if page.NextPageToken == "" {
			break
		}
		page, err = page.Next(ctx)
		if err != nil {
			log.Printf("  page error: %v", err)
			break
		}
	}
	if len(liveModels) == 0 {
		fmt.Println("  (No bidiGenerateContent models found — listing flash/live models)")
		page2, err2 := client.Models.List(ctx, nil)
		if err2 == nil {
			for {
				for _, model := range page2.Items {
					if strings.Contains(model.Name, "flash") || strings.Contains(model.Name, "live") {
						fmt.Printf("  MODEL: %-55s actions=%v\n", model.Name, model.SupportedActions)
					}
				}
				if page2.NextPageToken == "" {
					break
				}
				page2, _ = page2.Next(ctx)
			}
		}
	}
	return liveModels
}

func captureScreen() image.Image {
	tmpFile := "/tmp/vibecat_videotest.png"
	cmd := exec.Command("screencapture", "-x", "-C", tmpFile)
	if err := cmd.Run(); err != nil {
		log.Fatalf("screencapture failed: %v", err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		log.Fatalf("open screenshot: %v", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatalf("decode screenshot: %v", err)
	}
	return img
}

func resizeImage(src image.Image, targetW, targetH int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image, quality int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		log.Fatalf("jpeg encode: %v", err)
	}
	return buf.Bytes()
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
