package decoder_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/coder/decoder"
	"github.com/aecra/PeerCodeX/coder/encoder"
)

// generate random data of N-bytes
func generateData(n uint) []byte {
	data := make([]byte, n)
	// can safely ignore error
	rand.Read(data)
	return data
}

// Effect of sparsity on decoding speed
func BenchmarkGaussElimRLNCDecoder1(t *testing.B) {
	t.Run("128M", func(b *testing.B) {
		b.Run("128 Pieces 0", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0) })
		b.Run("128 Pieces 0.1", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.1) })
		b.Run("128 Pieces 0.2", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.2) })
		b.Run("128 Pieces 0.3", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.3) })
		b.Run("128 Pieces 0.4", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.4) })
		b.Run("128 Pieces 0.5", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.5) })
		b.Run("128 Pieces 0.6", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.6) })
		b.Run("128 Pieces 0.7", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.7) })
		b.Run("128 Pieces 0.8", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.8) })
		b.Run("128 Pieces 0.9", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.9) })
		b.Run("128 Pieces 0.91", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.91) })
		b.Run("128 Pieces 0.92", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.92) })
		b.Run("128 Pieces 0.93", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.93) })
		b.Run("128 Pieces 0.94", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.94) })
		b.Run("128 Pieces 0.95", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.95) })
		b.Run("128 Pieces 0.96", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.96) })
		b.Run("128 Pieces 0.97", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.97) })
		b.Run("128 Pieces 0.98", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.98) })
		b.Run("128 Pieces 0.99", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.99) })
	})
}

// The effect of the number of fragments on the decoding speed
func BenchmarkGaussElimRLNCDecoder2(t *testing.B) {
	t.Run("128M", func(b *testing.B) {
		b.Run("8 Pieces", func(b *testing.B) { decode(b, 1<<3, 1<<27, 0) })
		b.Run("16 Pieces", func(b *testing.B) { decode(b, 1<<4, 1<<27, 0) })
		b.Run("32 Pieces", func(b *testing.B) { decode(b, 1<<5, 1<<27, 0) })
		b.Run("64 Pieces", func(b *testing.B) { decode(b, 1<<6, 1<<27, 0) })
		b.Run("128 Pieces", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0) })
		b.Run("256 Pieces", func(b *testing.B) { decode(b, 1<<8, 1<<27, 0) })
		b.Run("512 Pieces", func(b *testing.B) { decode(b, 1<<9, 1<<27, 0) })
		b.Run("1024 Pieces", func(b *testing.B) { decode(b, 1<<10, 1<<27, 0) })
		b.Run("2048 Pieces", func(b *testing.B) { decode(b, 1<<11, 1<<27, 0) })
		b.Run("4096 Pieces", func(b *testing.B) { decode(b, 1<<12, 1<<27, 0) })
		b.Run("8192 Pieces", func(b *testing.B) { decode(b, 1<<13, 1<<27, 0) })
		b.Run("16384 Pieces", func(b *testing.B) { decode(b, 1<<14, 1<<27, 0) })
		b.Run("32768 Pieces", func(b *testing.B) { decode(b, 1<<15, 1<<27, 0) })
	})
}

// Effect of Generation Size on Decoding Speed
func BenchmarkGaussElimRLNCDecoder3(t *testing.B) {
	t.Run("128 Pieces", func(b *testing.B) {
		b.Run("64 k", func(b *testing.B) { decode(b, 1<<7, 1<<16, 0.95) })
		b.Run("128 k", func(b *testing.B) { decode(b, 1<<7, 1<<17, 0.95) })
		b.Run("256 k", func(b *testing.B) { decode(b, 1<<7, 1<<18, 0.95) })
		b.Run("512 k", func(b *testing.B) { decode(b, 1<<7, 1<<19, 0.95) })
		b.Run("1 M", func(b *testing.B) { decode(b, 1<<7, 1<<20, 0.95) })
		b.Run("2 M", func(b *testing.B) { decode(b, 1<<7, 1<<21, 0.95) })
		b.Run("4 M", func(b *testing.B) { decode(b, 1<<7, 1<<22, 0.95) })
		b.Run("8 M", func(b *testing.B) { decode(b, 1<<7, 1<<23, 0.95) })
		b.Run("16 M", func(b *testing.B) { decode(b, 1<<7, 1<<24, 0.95) })
		b.Run("32 M", func(b *testing.B) { decode(b, 1<<7, 1<<25, 0.95) })
		b.Run("64 M", func(b *testing.B) { decode(b, 1<<7, 1<<26, 0.95) })
		b.Run("128 M", func(b *testing.B) { decode(b, 1<<7, 1<<27, 0.95) })
		b.Run("256 M", func(b *testing.B) { decode(b, 1<<7, 1<<28, 0.95) })
		b.Run("512 M", func(b *testing.B) { decode(b, 1<<7, 1<<29, 0.95) })
	})
}

func decode(t *testing.B, pieceCount uint, total uint, p float64) {
	rand.Seed(time.Now().UnixNano())

	data := generateData(total)
	enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, p)
	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	pieces := make([]*coder.CodedPiece, 0, 8*pieceCount)
	for i := 0; i < int(8*pieceCount); i++ {
		pieces = append(pieces, enc.CodedPiece())
	}

	t.ResetTimer()

	totalDuration := 0 * time.Second
	count := 0
	for i := 0; i < t.N; i++ {
		td, ct := decode_(t, pieceCount, pieces)
		totalDuration += td
		count += ct
	}

	t.ReportMetric(float64(count)/float64(t.N), "piece/decode")
	t.ReportMetric(0, "ns/op")
	t.ReportMetric(float64(totalDuration.Seconds())/float64(t.N), "second/decode")
	t.ReportMetric(float64(total)/(float64(totalDuration.Seconds())/float64(t.N))/(1<<20), "MB/s")
}

func decode_(t *testing.B, pieceCount uint, pieces []*coder.CodedPiece) (time.Duration, int) {
	dec := decoder.NewGaussElimRLNCDecoder(pieceCount)
	count := 0
	// randomly shuffle piece ordering
	rand.Shuffle(int(8*pieceCount), func(i, j int) {
		pieces[i], pieces[j] = pieces[j], pieces[i]
	})

	totalDuration := 0 * time.Second
	for j := 0; j < int(8*pieceCount); j++ {
		if j+1 >= int(pieceCount) && dec.IsDecoded() {
			count = j
			break
		}

		begin := time.Now()
		dec.AddPiece(pieces[j])
		totalDuration += time.Since(begin)
	}

	if !dec.IsDecoded() {
		t.Fatal("expected pieces to be decoded")
	}

	return totalDuration, count
}
