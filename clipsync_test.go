package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
)

// loudness pattern with structure (like music): bursts of different lengths
func fakeEnvelope(n int, seed int64) []float64 {
	r := rand.New(rand.NewSource(seed))
	env := make([]float64, n)
	level := 0.0
	for i := 0; i < n; i++ {
		if i%13 == 0 {
			level = r.Float64() * 3
		}
		env[i] = level + r.Float64()*0.3
	}
	return standardise(env)
}

func TestFindLagRecoversKnownShift(t *testing.T) {
	song := fakeEnvelope(envRate*140, 1)
	for _, shift := range []int{0, 23, 50, 8 * envRate, 31*envRate + 7, -(5 * envRate), -(40*envRate + 3)} {
		clip := make([]float64, len(song)+abs(shift)+envRate*10)
		for i := range clip {
			j := i - shift // clip[i+shift] = song[i]
			if j >= 0 && j < len(song) {
				clip[i] = song[j]
			}
		}
		// noise on the clip (different mastering, sound effects)
		r := rand.New(rand.NewSource(int64(shift) + 99))
		for i := range clip {
			clip[i] += (r.Float64() - 0.5) * 0.4
		}
		lag, score := findLag(song, standardise(clip), int(maxOffset*envRate))
		if lag != shift {
			t.Errorf("shift %d: found %d (score %.2f)", shift, lag, score)
		}
		if score < 0.5 {
			t.Errorf("shift %d: confidence too low: %.2f", shift, score)
		}
	}
}

func TestFindLagUnrelatedSignalsAreLowConfidence(t *testing.T) {
	_, score := findLag(fakeEnvelope(envRate*100, 3), fakeEnvelope(envRate*100, 4), int(maxOffset*envRate))
	if score > 0.35 {
		t.Fatalf("unrelated signals must not look aligned (score %.2f)", score)
	}
}

func TestEnvelopeFromPCM(t *testing.T) {
	// 1 s of silence followed by 1 s of a loud tone -> envelope must step up
	pcm := make([]byte, 16000*2/2*2) // 2 s at 8 kHz mono s16
	for i := 8000; i < 16000; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(8000*math.Sin(float64(i)/5))))
	}
	env := envelopeFromPCM(pcm)
	if len(env) != 2*envRate {
		t.Fatalf("got %d frames", len(env))
	}
	if env[10] >= env[envRate+10] {
		t.Fatalf("silence must be quieter than the tone: %.2f vs %.2f", env[10], env[envRate+10])
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
