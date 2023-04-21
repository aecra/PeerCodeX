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

func BenchmarkGaussElimRLNCDecoder(t *testing.B) {
	t.Run("256K", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { decode(b, 1<<4, 1<<18) })
		b.Run("32 Pieces", func(b *testing.B) { decode(b, 1<<5, 1<<18) })
		b.Run("64 Pieces", func(b *testing.B) { decode(b, 1<<6, 1<<18) })
		b.Run("128 Pieces", func(b *testing.B) { decode(b, 1<<7, 1<<18) })
		b.Run("256 Pieces", func(b *testing.B) { decode(b, 1<<8, 1<<18) })
	})

	t.Run("512K", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { decode(b, 1<<4, 1<<19) })
		b.Run("32 Pieces", func(b *testing.B) { decode(b, 1<<5, 1<<19) })
		b.Run("64 Pieces", func(b *testing.B) { decode(b, 1<<6, 1<<19) })
		b.Run("128 Pieces", func(b *testing.B) { decode(b, 1<<7, 1<<19) })
		b.Run("256 Pieces", func(b *testing.B) { decode(b, 1<<8, 1<<19) })
	})

	t.Run("1M", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { decode(b, 1<<4, 1<<20) })
		b.Run("32 Pieces", func(b *testing.B) { decode(b, 1<<5, 1<<20) })
		b.Run("64 Pieces", func(b *testing.B) { decode(b, 1<<6, 1<<20) })
		b.Run("128 Pieces", func(b *testing.B) { decode(b, 1<<7, 1<<20) })
		b.Run("256 Pieces", func(b *testing.B) { decode(b, 1<<8, 1<<20) })
	})

	t.Run("2M", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { decode(b, 1<<4, 1<<21) })
		b.Run("32 Pieces", func(b *testing.B) { decode(b, 1<<5, 1<<21) })
		b.Run("64 Pieces", func(b *testing.B) { decode(b, 1<<6, 1<<21) })
		b.Run("128 Pieces", func(b *testing.B) { decode(b, 1<<7, 1<<21) })
		b.Run("256 Pieces", func(b *testing.B) { decode(b, 1<<8, 1<<21) })
	})
}

func BenchmarkGaussElimRLNCDecoder1(t *testing.B) {
	t.Run("128M", func(b *testing.B) {
		// b.Run("16 Pieces", func(b *testing.B) { decode(b, 1<<4, 1<<27) })
		// b.Run("32 Pieces", func(b *testing.B) { decode(b, 1<<5, 1<<27) })
		// b.Run("64 Pieces", func(b *testing.B) { decode(b, 1<<6, 1<<27) })
		// b.Run("128 Pieces", func(b *testing.B) { decode(b, 1<<7, 1<<27) })
		b.Run("256 Pieces", func(b *testing.B) { decode(b, 1<<8, 1<<27) })
		// b.Run("512 Pieces", func(b *testing.B) { decode(b, 1<<9, 1<<27) })
		// b.Run("1024 Pieces", func(b *testing.B) { decode(b, 1<<10, 1<<27) })
		// b.Run("2048 Pieces", func(b *testing.B) { decode(b, 1<<11, 1<<27) })
		// b.Run("4096 Pieces", func(b *testing.B) { decode(b, 1<<12, 1<<27) })
		// b.Run("8192 Pieces", func(b *testing.B) { decode(b, 1<<13, 1<<27) })
	})
}

func decode(t *testing.B, pieceCount uint, total uint) {
	rand.Seed(time.Now().UnixNano())

	data := generateData(total)
	// enc, err := encoder.NewFullRLNCEncoderWithPieceCount(data, pieceCount)
	enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, 0.4)
	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	pieces := make([]*coder.CodedPiece, 0, 2*pieceCount)
	for i := 0; i < int(2*pieceCount); i++ {
		pieces = append(pieces, enc.CodedPiece())
	}

	t.ResetTimer()

	totalDuration := 0 * time.Second
	for i := 0; i < t.N; i++ {
		totalDuration += decode_(t, pieceCount, pieces)
	}

	t.ReportMetric(0, "ns/op")
	t.ReportMetric(float64(totalDuration.Seconds())/float64(t.N), "second/decode")
}

func decode_(t *testing.B, pieceCount uint, pieces []*coder.CodedPiece) time.Duration {
	dec := decoder.NewGaussElimRLNCDecoder(pieceCount)

	// randomly shuffle piece ordering
	rand.Shuffle(int(2*pieceCount), func(i, j int) {
		pieces[i], pieces[j] = pieces[j], pieces[i]
	})

	totalDuration := 0 * time.Second
	for j := 0; j < int(2*pieceCount); j++ {
		if j+1 >= int(pieceCount) && dec.IsDecoded() {
			break
		}

		begin := time.Now()
		dec.AddPiece(pieces[j])
		totalDuration += time.Since(begin)
	}

	if !dec.IsDecoded() {
		t.Fatal("expected pieces to be decoded")
	}

	return totalDuration
}
