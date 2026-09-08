package convert

// Plan is the compiled description of a conversion — what the flow decided to do, separated
// from how it's executed. A flow (Rules v2) builds a Plan by walking its nodes; for now the
// "Save space" preset builds it from the global settings. The compiler (compileOutputArgs)
// turns a Plan into an ffmpeg command, and preview reads the Plan directly — which is what
// makes the preview exact (you preview the literal thing that will run).
type Plan struct {
	// Video.
	VideoCodec  string // target video codec: "hevc" | "h264" | "av1"; "" = copy (remux only)
	Quality     int    // CRF/CQ target; 0 = codec default (hw encoders map internally)
	VFRToCFR    bool   // normalize variable frame rate when present
	ScaleHeight int    // downscale to this height (0 = keep); never upscales

	Audio     AudioPlan
	Subs      SubPlan
	Container string // "mkv" | "mp4"

	// HealthCheck, with no transcode (VideoCodec == ""), turns the job into a read-only
	// corruption scan that reports issues instead of replacing the file (R5).
	HealthCheck bool
	// ExtraArgs are raw ffmpeg output args appended verbatim — the advanced escape hatch
	// for anything the structured actions don't cover (R5). Empty for the common case.
	ExtraArgs []string
}

// SubPlan is the subtitle portion of a Plan.
//
// A WEB-DL commonly ships thirty-odd subtitle tracks for languages nobody in the house
// reads. They cost little space, but they clutter every player's track menu and some
// clients pick one at random.
type SubPlan struct {
	// KeepLangs keeps only these languages (empty = keep all). Matched the same way as
	// audio, so "en" and "eng" both work, and an untagged track is kept rather than
	// guessed at.
	KeepLangs []string
	// DropImage removes image-based subtitle tracks (PGS, VobSub, DVB). Plex can't send
	// those to most clients as-is: it burns them into the picture, which means
	// transcoding the video every time that subtitle is on. A text subtitle for the
	// same language direct-plays.
	//
	// Guarded: an image track is only dropped when a TEXT subtitle for its language
	// exists — an embedded text track that survives the filter, or an .srt sidecar (see
	// TextSidecarLangs). Otherwise the image track is the only subtitle there is, and
	// it stays until the Subtitles module has produced a text one.
	DropImage bool
	// TextSidecarLangs lists the languages that have an external .srt beside THIS file.
	// Per-file, filled in by the caller (withSidecars) right before use; the rest of the
	// plan comes from settings.
	TextSidecarLangs []string
}

// AudioPlan is the audio portion of a Plan.
type AudioPlan struct {
	KeepLangs []string // keep only these languages (empty = keep all)
	AddStereo bool     // add an AAC 2.0 downmix beside surround tracks
	Loudnorm  bool     // EBU R128 loudness normalize (re-encodes to AAC)
}
