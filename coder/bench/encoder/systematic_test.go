package encoder_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder/encoder"
)

func BenchmarkSystematicRLNCEncoder(b *testing.B) {
	b.Run("1M", func(b *testing.B) {
		b.Run("16Pieces", func(b *testing.B) { systematicEncode(b, 1<<4, 1<<20) })
		b.Run("32Pieces", func(b *testing.B) { systematicEncode(b, 1<<5, 1<<20) })
		b.Run("64Pieces", func(b *testing.B) { systematicEncode(b, 1<<6, 1<<20) })
		b.Run("128Pieces", func(b *testing.B) { systematicEncode(b, 1<<7, 1<<20) })
		b.Run("256Pieces", func(b *testing.B) { systematicEncode(b, 1<<8, 1<<20) })
		b.Run("512Pieces", func(b *testing.B) { systematicEncode(b, 1<<9, 1<<20) })
	})
}

func systematicEncode(t *testing.B, pieceCount uint, total uint) {
	// non-reproducible random number sequence
	rand.Seed(time.Now().UnixNano())

	data := generateData(total)
	enc, err := encoder.NewSystematicRLNCEncoderWithPieceCount(data, pieceCount)
	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	t.ReportAllocs()
	t.SetBytes(int64(total+enc.Padding()) + int64(enc.CodedPieceLen()))
	t.ResetTimer()

	// keep generating encoded pieces on-the-fly
	for i := 0; i < t.N; i++ {
		enc.CodedPiece()
	}
}
