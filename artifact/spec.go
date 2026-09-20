package artifact

const (
	// ModelRevision is the immutable upstream Hugging Face revision.
	ModelRevision = "c5d78730f3493e4fe16d61507ef4b78eef7318cf"
	// UpstreamModel is the Hugging Face repository used for the export.
	UpstreamModel = "convaiinnovations/laya"
	// ONNXRuntimeVersion is the bundled native runtime version.
	ONNXRuntimeVersion = "1.23.2"
	// ReleaseTag contains the pinned model assets.
	ReleaseTag = "model-c5d78730"
	// DefaultModelURL is the immutable GitHub release asset base URL.
	DefaultModelURL = "https://github.com/MstyAI/laya-onnx/releases/download/" + ReleaseTag
)
