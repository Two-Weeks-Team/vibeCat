package geminiconfig

const (
	// VisionModel is the default non-Live multimodal model across the orchestrator.
	VisionModel = "gemini-3.1-flash-lite-preview"

	// SearchModel stays aligned with the orchestrator default so search, vision, and support
	// paths all use the same current-generation non-Live model.
	SearchModel = "gemini-3.1-flash-lite-preview"

	// LiteTextModel is the shared default for classifier/support/text generation paths.
	LiteTextModel = "gemini-3.1-flash-lite-preview"
)
