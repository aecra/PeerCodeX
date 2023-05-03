package encoder_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder/encoder"
)

// Effect of sparsity on encoding speed
func BenchmarkSparseRLNCEncoder1(b *testing.B) {
	b.Run("128M", func(b *testing.B) {
		b.Run("0", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0) })
		b.Run("0.10", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.10) })
		b.Run("0.20", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.20) })
		b.Run("0.30", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.30) })
		b.Run("0.40", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.40) })
		b.Run("0.50", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.50) })
		b.Run("0.60", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.60) })
		b.Run("0.70", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.70) })
		b.Run("0.80", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.80) })
		b.Run("0.90", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.90) })
		b.Run("1.00", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 1.00) })
	})
}

// Effect of the number of fragments on the encoding speed
func BenchmarkSparseRLNCEncoder2(b *testing.B) {
	b.Run("128M", func(b *testing.B) {
		b.Run("2", func(b *testing.B) { sparseEncode(b, 1<<1, 1<<27, 0) })
		b.Run("4", func(b *testing.B) { sparseEncode(b, 1<<2, 1<<27, 0) })
		b.Run("8", func(b *testing.B) { sparseEncode(b, 1<<3, 1<<27, 0) })
		b.Run("16", func(b *testing.B) { sparseEncode(b, 1<<4, 1<<27, 0) })
		b.Run("32", func(b *testing.B) { sparseEncode(b, 1<<5, 1<<27, 0) })
		b.Run("64", func(b *testing.B) { sparseEncode(b, 1<<6, 1<<27, 0) })
		b.Run("128", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0) })
		b.Run("256", func(b *testing.B) { sparseEncode(b, 1<<8, 1<<27, 0) })
		b.Run("512", func(b *testing.B) { sparseEncode(b, 1<<9, 1<<27, 0) })
		b.Run("1024", func(b *testing.B) { sparseEncode(b, 1<<10, 1<<27, 0) })
		b.Run("2048", func(b *testing.B) { sparseEncode(b, 1<<11, 1<<27, 0) })
		b.Run("4096", func(b *testing.B) { sparseEncode(b, 1<<12, 1<<27, 0) })
		b.Run("8192", func(b *testing.B) { sparseEncode(b, 1<<13, 1<<27, 0) })
		b.Run("16384", func(b *testing.B) { sparseEncode(b, 1<<14, 1<<27, 0) })
		b.Run("32768", func(b *testing.B) { sparseEncode(b, 1<<15, 1<<27, 0) })
	})
}

// Effect of Generation Size on Encoding Speed
func BenchmarkSparseRLNCEncoder3(b *testing.B) {
	b.Run("128 piece", func(b *testing.B) {
		b.Run("64KB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<16, 0.95) })
		b.Run("128KB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<17, 0.95) })
		b.Run("256KB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<18, 0.95) })
		b.Run("512KB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<19, 0.95) })
		b.Run("1MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<20, 0.95) })
		b.Run("2MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<21, 0.95) })
		b.Run("4MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<22, 0.95) })
		b.Run("8MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<23, 0.95) })
		b.Run("16MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<24, 0.95) })
		b.Run("32MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<25, 0.95) })
		b.Run("64MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<26, 0.95) })
		b.Run("128MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<27, 0.95) })
		b.Run("256MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<28, 0.95) })
		b.Run("512MB", func(b *testing.B) { sparseEncode(b, 1<<7, 1<<29, 0.95) })
	})
}

func sparseEncode(t *testing.B, pieceCount uint, total uint, p float64) {
	// non-reproducible random number sequence
	rand.Seed(time.Now().UnixNano())

	data := generateData(total)
	enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, p)
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
