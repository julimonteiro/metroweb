package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Clip alignment. The music video of a song is usually not sample-aligned with
// the audio track: it often has a few seconds of intro. To show the video in
// step with the song that is already playing, the server measures that offset
// by cross-correlating the loudness envelopes of the two audio signals.

const (
	envRate   = 50   // envelope frames per second (20 ms resolution)
	maxOffset = 75.0 // seconds of shift we search, either direction
	analyzeS  = 150  // seconds of audio compared
)

type ClipOffset struct {
	Offset     float64 `json:"offset"`     // clip time = song time + offset (seconds)
	Confidence float64 `json:"confidence"` // 0..1, how sure we are
}

var (
	offsetMu    sync.Mutex
	offsetCache = map[string]ClipOffset{}
)

// MeasureClipOffset downloads (or reuses) both files and compares them.
func MeasureClipOffset(ctx context.Context, songID, clipID string) (ClipOffset, error) {
	key := songID + "|" + clipID
	offsetMu.Lock()
	if v, ok := offsetCache[key]; ok {
		offsetMu.Unlock()
		return v, nil
	}
	offsetMu.Unlock()

	songPath, err := ensureCached(ctx, songID, "aac")
	if err != nil {
		return ClipOffset{}, fmt.Errorf("song audio: %w", err)
	}
	clipPath, err := ensureCached(ctx, clipID, "video")
	if err != nil {
		return ClipOffset{}, fmt.Errorf("clip: %w", err)
	}
	s, err := envelope(ctx, songPath)
	if err != nil {
		return ClipOffset{}, err
	}
	c, err := envelope(ctx, clipPath)
	if err != nil {
		return ClipOffset{}, err
	}
	lag, score := findLag(s, c, int(maxOffset*envRate))
	res := ClipOffset{Offset: float64(lag) / envRate, Confidence: math.Max(0, math.Min(1, score))}
	if res.Confidence < 0.35 {
		res.Offset = 0 // no trustworthy alignment: assume they start together
	}
	offsetMu.Lock()
	if len(offsetCache) > 500 {
		offsetCache = map[string]ClipOffset{}
	}
	offsetCache[key] = res
	offsetMu.Unlock()
	return res, nil
}

// envelope decodes the first minutes of a file to mono 8 kHz PCM and returns its
// standardised log-loudness per 20 ms frame.
func envelope(ctx context.Context, path string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-i", path, "-t", fmt.Sprint(analyzeS),
		"-vn", "-ac", "1", "-ar", "8000", "-f", "s16le", "-")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return envelopeFromPCM(out.Bytes()), nil
}

func envelopeFromPCM(b []byte) []float64 {
	const frame = 8000 / envRate
	n := len(b) / 2 / frame
	env := make([]float64, n)
	for i := 0; i < n; i++ {
		var sum float64
		for j := 0; j < frame; j++ {
			v := float64(int16(binary.LittleEndian.Uint16(b[(i*frame+j)*2:])))
			sum += v * v
		}
		env[i] = math.Log1p(math.Sqrt(sum/frame) / 100)
	}
	return standardise(env)
}

func standardise(x []float64) []float64 {
	if len(x) == 0 {
		return x
	}
	var mean float64
	for _, v := range x {
		mean += v
	}
	mean /= float64(len(x))
	var variance float64
	for _, v := range x {
		variance += (v - mean) * (v - mean)
	}
	sd := math.Sqrt(variance / float64(len(x)))
	if sd == 0 {
		sd = 1
	}
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = (v - mean) / sd
	}
	return out
}

// findLag returns k maximising the correlation of s[i] with c[i+k], i.e. how
// many frames later the song's content appears in the clip, and the score
// (about the Pearson coefficient of the overlapping parts).
func findLag(s, c []float64, maxLag int) (int, float64) {
	bestK, best := 0, math.Inf(-1)
	for k := -maxLag; k <= maxLag; k++ {
		lo, hi := 0, len(s)
		if k < 0 {
			lo = -k
		}
		if hi+k > len(c) {
			hi = len(c) - k
		}
		n := hi - lo
		if n < envRate*20 { // need at least 20 s of overlap to trust a lag
			continue
		}
		var dot float64
		for i := lo; i < hi; i++ {
			dot += s[i] * c[i+k]
		}
		if score := dot / float64(n); score > best {
			best, bestK = score, k
		}
	}
	if math.IsInf(best, -1) {
		return 0, 0
	}
	return bestK, best
}
