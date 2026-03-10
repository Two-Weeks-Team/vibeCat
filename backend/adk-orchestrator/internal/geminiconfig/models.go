package geminiconfig

const (
	// VisionModel is the default non-Live multimodal model across the orchestrator.
	VisionModel = "gemini-3.1-flash-lite-preview"

	// SearchModel uses a stable 2.5 Flash variant because grounding tools such as
	// Google Search and Google Maps are broadly supported there.
	SearchModel = "gemini-2.5-flash"

	// LiteTextModel is the shared low-latency classifier/support model.
	LiteTextModel = "gemini-2.5-flash-lite"

	// ToolModel powers explicit grounded tool invocations from the orchestrator.
	ToolModel = "gemini-2.5-flash"
)
