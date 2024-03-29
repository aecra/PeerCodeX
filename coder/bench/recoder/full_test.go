package recoder_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/coder/encoder"
	"github.com/aecra/PeerCodeX/coder/recoder"
)

func BenchmarkFullRLNCRecoder(t *testing.B) {
	t.Run("1M", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { recode(b, 1<<4, 1<<20) })
		b.Run("32 Pieces", func(b *testing.B) { recode(b, 1<<5, 1<<20) })
		b.Run("64 Pieces", func(b *testing.B) { recode(b, 1<<6, 1<<20) })
		b.Run("128 Pieces", func(b *testing.B) { recode(b, 1<<7, 1<<20) })
		b.Run("256 Pieces", func(b *testing.B) { recode(b, 1<<8, 1<<20) })
	})

	t.Run("16M", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { recode(b, 1<<4, 1<<24) })
		b.Run("32 Pieces", func(b *testing.B) { recode(b, 1<<5, 1<<24) })
		b.Run("64 Pieces", func(b *testing.B) { recode(b, 1<<6, 1<<24) })
		b.Run("128 Pieces", func(b *testing.B) { recode(b, 1<<7, 1<<24) })
		b.Run("256 Pieces", func(b *testing.B) { recode(b, 1<<8, 1<<24) })
	})

	t.Run("32M", func(b *testing.B) {
		b.Run("16 Pieces", func(b *testing.B) { recode(b, 1<<4, 1<<25) })
		b.Run("32 Pieces", func(b *testing.B) { recode(b, 1<<5, 1<<25) })
		b.Run("64 Pieces", func(b *testing.B) { recode(b, 1<<6, 1<<25) })
		b.Run("128 Pieces", func(b *testing.B) { recode(b, 1<<7, 1<<25) })
		b.Run("256 Pieces", func(b *testing.B) { recode(b, 1<<8, 1<<25) })
	})
}

func recode(t *testing.B, pieceCount uint, total uint) {
	// non-reproducible sequence
	rand.Seed(time.Now().UnixNano())

	// -- encode
	data := generateData(total)
	enc, err := encoder.NewFullRLNCEncoderWithPieceCount(data, pieceCount)
	if err != nil {
		t.Fatalf("Error: %s\n", err.Error())
	}

	pieces := make([]*coder.CodedPiece, 0, pieceCount)
	for i := 0; i < int(pieceCount); i++ {
		pieces = append(pieces, enc.CodedPiece())
	}
	// -- encoding ends

	// -- recode
	rec := recoder.NewFullRLNCRecoder(pieces)

	t.ReportAllocs()
	t.SetBytes(int64((pieceCount+total/pieceCount)*pieceCount) + int64(pieceCount+total/pieceCount))
	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		if _, err := rec.CodedPiece(); err != nil {
			t.Fatalf("Error: %s\n", err.Error())
		}
	}
	// -- recoding ends
}

// generate random data of N-bytes
func generateData(n uint) []byte {
	data := make([]byte, n)
	// can safely ignore error
	rand.Read(data)
	return data
}
