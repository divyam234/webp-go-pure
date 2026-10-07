package webp

import (
	"bytes"
	"math/bits"
	"testing"
	"time"
)

func (w *vp8BoolWriter) putBitReference(bit bool, prob uint8) bool {
	split := (w.rng * uint32(prob)) >> 8
	if bit {
		w.value += split + 1
		w.rng -= split + 1
	} else {
		w.rng = split
	}
	if w.rng < 127 {
		shift := int32(7) - int32(bits.Len32(w.rng+1)-1)
		w.rng = ((w.rng + 1) << uint(shift)) - 1
		w.value <<= uint(shift)
		w.nbBits += shift
		if w.nbBits > 0 {
			w.flush()
		}
	}
	return bit
}

func boolWriterSequence() ([4096]bool, [4096]uint8) {
	var bitValues [4096]bool
	var probs [4096]uint8
	state := uint32(0x9e3779b9)
	for i := range probs {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		bitValues[i] = state&1 != 0
		probs[i] = uint8(state>>24) | 1
	}
	return bitValues, probs
}

func TestVP8BoolWriterOptimizedMatchesReference(t *testing.T) {
	bitValues, probs := boolWriterSequence()
	current := newVp8BoolWriter(4096)
	reference := newVp8BoolWriter(4096)
	for i, bit := range bitValues {
		current.putBit(bit, probs[i])
		reference.putBitReference(bit, probs[i])
	}
	got := current.finish()
	want := reference.finish()
	if !bytes.Equal(got, want) {
		t.Fatalf("optimized bool writer changed output: got %d bytes, want %d", len(got), len(want))
	}
}

func BenchmarkVP8BoolWriterPaired(b *testing.B) {
	bitValues, probs := boolWriterSequence()
	var currentTotal, referenceTotal time.Duration

	runCurrent := func() {
		w := newVp8BoolWriter(4096)
		start := time.Now()
		for i, bit := range bitValues {
			w.putBit(bit, probs[i])
		}
		currentTotal += time.Since(start)
		_ = w.finish()
	}
	runReference := func() {
		w := newVp8BoolWriter(4096)
		start := time.Now()
		for i, bit := range bitValues {
			w.putBitReference(bit, probs[i])
		}
		referenceTotal += time.Since(start)
		_ = w.finish()
	}

	for i := 0; i < b.N; i++ {
		if i&1 == 0 {
			runCurrent()
			runReference()
		} else {
			runReference()
			runCurrent()
		}
	}

	if b.N > 0 {
		b.ReportMetric(float64(currentTotal.Nanoseconds())/float64(b.N*len(bitValues)), "current-ns/bit")
		b.ReportMetric(float64(referenceTotal.Nanoseconds())/float64(b.N*len(bitValues)), "reference-ns/bit")
		b.ReportMetric(float64(currentTotal)/float64(referenceTotal), "current/reference")
	}
}

func BenchmarkVP8UniformVsProbability128(b *testing.B) {
	bitValues, _ := boolWriterSequence()
	var uniformTotal, generalTotal time.Duration

	runUniform := func() {
		w := newVp8BoolWriter(4096)
		start := time.Now()
		for _, bit := range bitValues {
			w.putBitUniform(bit)
		}
		uniformTotal += time.Since(start)
	}
	runGeneral := func() {
		w := newVp8BoolWriter(4096)
		start := time.Now()
		for _, bit := range bitValues {
			w.putBit(bit, 128)
		}
		generalTotal += time.Since(start)
	}

	for i := 0; i < b.N; i++ {
		if i&1 == 0 {
			runUniform()
			runGeneral()
		} else {
			runGeneral()
			runUniform()
		}
	}

	if b.N > 0 {
		b.ReportMetric(float64(uniformTotal.Nanoseconds())/float64(b.N*len(bitValues)), "uniform-ns/bit")
		b.ReportMetric(float64(generalTotal.Nanoseconds())/float64(b.N*len(bitValues)), "general-ns/bit")
		b.ReportMetric(float64(uniformTotal)/float64(generalTotal), "uniform/general")
	}
}
