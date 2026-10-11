// Package meow converts assets/meow-source.wav to assets/meow.pcm (4-bit samples at 3.9 kHz).
package meow

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

const (
	Rate    = 3900
	Start   = 0.13
	Samples = 12 * 256
)

// WAV is a mono 16-bit PCM clip.
type WAV struct {
	Channels, SampleWidth, Rate int
	Samples                     []int16
}

// ReadWAV is a minimal RIFF/WAVE PCM reader (16-bit little-endian data).
func ReadWAV(data []byte) (*WAV, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, errors.New("not a RIFF/WAVE file")
	}
	w := &WAV{}
	var haveFmt bool
	for i := 12; i+8 <= len(data); {
		id := string(data[i : i+4])
		size := int(binary.LittleEndian.Uint32(data[i+4:]))
		body := data[i+8 : min(i+8+size, len(data))]
		switch id {
		case "fmt ":
			if len(body) < 16 || binary.LittleEndian.Uint16(body) != 1 {
				return nil, errors.New("not PCM")
			}
			w.Channels = int(binary.LittleEndian.Uint16(body[2:]))
			w.Rate = int(binary.LittleEndian.Uint32(body[4:]))
			w.SampleWidth = int(binary.LittleEndian.Uint16(body[14:])) / 8
			haveFmt = true
		case "data":
			if !haveFmt || w.SampleWidth != 2 {
				return nil, errors.New("unsupported sample width")
			}
			w.Samples = make([]int16, len(body)/2)
			for j := range w.Samples {
				w.Samples[j] = int16(binary.LittleEndian.Uint16(body[2*j:]))
			}
			return w, nil
		}
		i += 8 + size + size&1
	}
	return nil, errors.New("no data chunk")
}

// Frames is the clip length in frames (as python wave.getnframes()).
func (w *WAV) Frames() int { return len(w.Samples) / max(w.Channels, 1) }

// Convert resamples the source into SAMPLES 4-bit values.
func Convert() ([]byte, error) {
	raw, err := os.ReadFile(k65src.Path("assets", "meow-source.wav"))
	if err != nil {
		return nil, err
	}
	wav, err := ReadWAV(raw)
	if err != nil {
		return nil, err
	}
	if wav.Channels != 1 || wav.SampleWidth != 2 {
		return nil, errors.New("meow source must be mono 16-bit")
	}
	source := wav.Samples
	filtered := make([]float64, Samples)
	for i := range filtered {
		center := (Start + float64(i)/Rate) * float64(wav.Rate)
		left := int(center) - 12
		weighted, total := 0.0, 0.0
		for j := left; j < left+25; j++ {
			weight := 12 - math.Abs(float64(j)-center)
			if weight > 0 && j >= 0 && j < len(source) {
				weighted += float64(source[j]) * weight
				total += weight
			}
		}
		filtered[i] = weighted / total
	}
	peak := 0.0
	for _, v := range filtered {
		peak = max(peak, math.Abs(v))
	}
	result := make([]byte, Samples)
	lo, hi := byte(255), byte(0)
	for i, v := range filtered {
		q := math.RoundToEven(7.5 + 7.5*v/peak)
		result[i] = byte(max(0, min(15, q)))
		lo, hi = min(lo, result[i]), max(hi, result[i])
	}
	if int(hi)-int(lo) < 12 {
		return nil, fmt.Errorf("meow dynamic range too small")
	}
	return result, nil
}

// Run writes assets/meow.pcm.
func Run() error {
	b, err := Convert()
	if err != nil {
		return err
	}
	return os.WriteFile(k65src.Path("assets", "meow.pcm"), b, 0o644)
}
