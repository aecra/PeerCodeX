package encoder_test

import (
	"bytes"
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder/decoder"
	"github.com/aecra/PeerCodeX/coder/encoder"
	"github.com/aecra/PeerCodeX/coder"
)

func sparseEncoderFlow(t *testing.T, enc encoder.Encoder, pieceCount, codedPieceCount int, pieces []coder.Piece) {
	coded := make([]*coder.CodedPiece, 0, codedPieceCount)
	for i := 0; i < codedPieceCount; i++ {
		coded = append(coded, enc.CodedPiece())
	}

	dec := decoder.NewGaussElimRLNCDecoder(uint(pieceCount))
	for i := 0; i < codedPieceCount; i++ {
		if i < pieceCount {
			if _, err := dec.GetPieces(); !(err != nil && errors.Is(err, coder.ErrMoreUsefulPiecesRequired)) {
				t.Fatal("expected error indicating more pieces are required for decoding")
			}
		}

		if err := dec.AddPiece(coded[i]); errors.Is(err, coder.ErrAllUsefulPiecesReceived) {
			break
		}
	}

	if !dec.IsDecoded() {
		t.Fatal("expected to be fully decoded !")
	}

	for i := 0; i < codedPieceCount-pieceCount; i++ {
		if err := dec.AddPiece(coded[pieceCount+i]); !(err != nil && errors.Is(err, coder.ErrAllUsefulPiecesReceived)) {
			t.Fatal("expected error indication, received nothing !")
		}
	}

	d_pieces, err := dec.GetPieces()
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(pieces) != len(d_pieces) {
		t.Fatal("didn't decode all !")
	}

	for i := 0; i < pieceCount; i++ {
		if !bytes.Equal(pieces[i], d_pieces[i]) {
			t.Fatal("decoded data doesn't match !")
		}
	}
}

func TestNewSparseRLNCEncoder(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	pieceCount := 128
	pieceLength := 8192
	codedPieceCount := pieceCount + 2
	pieces := generatePieces(uint(pieceCount), uint(pieceLength))
	enc := encoder.NewSparseRLNCEncoder(pieces, 0.5)

	sparseEncoderFlow(t, enc, pieceCount, codedPieceCount, pieces)
}

func TestNewSparseRLNCEncoderWithPieceCount(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	size := uint(2<<10 + rand.Intn(2<<10))
	pieceCount := uint(2<<1 + rand.Intn(2<<8))
	codedPieceCount := pieceCount + 2
	data := generateData(size)
	t.Logf("\nTotal Data: %d bytes\nPiece Count: %d\nCoded Piece Count: %d\n", size, pieceCount, codedPieceCount)

	pieces, _, err := coder.OriginalPiecesFromDataAndPieceCount(data, pieceCount)
	if err != nil {
		t.Fatal(err.Error())
	}

	enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, 0.5)
	if err != nil {
		t.Fatal(err.Error())
	}

	sparseEncoderFlow(t, enc, int(pieceCount), int(codedPieceCount), pieces)
}

func TestNewSparseRLNCEncoderWithPieceSize(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	size := uint(2<<10 + rand.Intn(2<<10))
	pieceSize := uint(2<<5 + rand.Intn(2<<5))
	pieceCount := int(math.Ceil(float64(size) / float64(pieceSize)))
	codedPieceCount := pieceCount + 2
	data := generateData(size)
	t.Logf("\nTotal Data: %d bytes\nPiece Size: %d bytes\nPiece Count: %d\nCoded Piece Count: %d\n", size, pieceSize, pieceCount, codedPieceCount)

	pieces, _, err := coder.OriginalPiecesFromDataAndPieceSize(data, pieceSize)
	if err != nil {
		t.Fatal(err.Error())
	}

	enc, err := encoder.NewSparseRLNCEncoderWithPieceSize(data, pieceSize, 0.5)
	if err != nil {
		t.Fatal(err.Error())
	}

	sparseEncoderFlow(t, enc, pieceCount, codedPieceCount, pieces)
}

func TestSparseRLNCEncoderPadding(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	t.Run("WithPieceCount", func(t *testing.T) {
		for i := 0; i < 1<<5; i++ {
			size := uint(2<<10 + rand.Intn(2<<10))
			pieceCount := uint(2<<1 + rand.Intn(2<<8))
			data := generateData(size)

			enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, 0.5)
			if err != nil {
				t.Fatalf("Error: %s\n", err.Error())
			}

			extra := enc.Padding()
			pieceSize := (size + extra) / pieceCount
			c_piece := enc.CodedPiece()
			if uint(len(c_piece.Piece)) != pieceSize {
				t.Fatalf("expected pieceSize to be %dB, found to be %dB\n", pieceSize, len(c_piece.Piece))
			}
		}
	})

	t.Run("WithPieceSize", func(t *testing.T) {
		for i := 0; i < 1<<5; i++ {
			size := uint(2<<10 + rand.Intn(2<<10))
			pieceSize := uint(2<<5 + rand.Intn(2<<5))
			pieceCount := uint(math.Ceil(float64(size) / float64(pieceSize)))
			data := generateData(size)

			enc, err := encoder.NewSparseRLNCEncoderWithPieceSize(data, pieceSize, 0.5)
			if err != nil {
				t.Fatalf("Error: %s\n", err.Error())
			}

			extra := enc.Padding()
			c_pieceSize := (size + extra) / pieceCount
			c_piece := enc.CodedPiece()
			if pieceSize != c_pieceSize || uint(len(c_piece.Piece)) != pieceSize {
				t.Fatalf("expected pieceSize to be %dB, found to be %dB\n", c_pieceSize, len(c_piece.Piece))
			}
		}
	})
}

func TestSparseRLNCEncoder_CodedPieceLen(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	t.Run("WithPieceCount", func(t *testing.T) {
		size := uint(2<<10 + rand.Intn(2<<10))
		pieceCount := uint(2<<1 + rand.Intn(2<<8))
		data := generateData(size)

		enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, 0.5)
		if err != nil {
			t.Fatalf("Error: %s\n", err.Error())
		}

		for i := 0; i <= int(pieceCount); i++ {
			c_piece := enc.CodedPiece()
			if c_piece.Len() != enc.CodedPieceLen() {
				t.Fatalf("expected coded piece to be of %dB, found to be of %dB\n", enc.CodedPieceLen(), c_piece.Len())
			}
		}
	})

	t.Run("WithPieceSize", func(t *testing.T) {
		size := uint(2<<10 + rand.Intn(2<<10))
		pieceSize := uint(2<<5 + rand.Intn(2<<5))
		pieceCount := uint(math.Ceil(float64(size) / float64(pieceSize)))
		data := generateData(size)

		enc, err := encoder.NewSparseRLNCEncoderWithPieceSize(data, pieceSize, 0.5)
		if err != nil {
			t.Fatalf("Error: %s\n", err.Error())
		}

		for i := 0; i <= int(pieceCount); i++ {
			c_piece := enc.CodedPiece()
			if c_piece.Len() != enc.CodedPieceLen() {
				t.Fatalf("expected coded piece to be of %dB, found to be of %dB\n", enc.CodedPieceLen(), c_piece.Len())
			}
		}
	})
}

func TestSparseRLNCEncoder_DecodableLen(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	flow := func(enc encoder.Encoder, dec decoder.Decoder) {
		consumed_len := uint(0)
		for !dec.IsDecoded() {
			c_piece := enc.CodedPiece()
			// randomly drop piece
			if rand.Intn(2) == 0 {
				continue
			}
			if err := dec.AddPiece(c_piece); errors.Is(err, coder.ErrAllUsefulPiecesReceived) {
				break
			}

			// as consumed this piece --- accounting
			consumed_len += c_piece.Len()
		}

		if consumed_len < enc.DecodableLen() {
			t.Fatalf("expected to consume >=%dB for decoding, but actually consumed %dB\n", enc.DecodableLen(), consumed_len)
		}
	}

	t.Run("WithPieceCount", func(t *testing.T) {
		size := uint(2<<10 + rand.Intn(2<<10))
		pieceCount := uint(2<<1 + rand.Intn(2<<8))
		data := generateData(size)

		enc, err := encoder.NewSparseRLNCEncoderWithPieceCount(data, pieceCount, 0.5)
		if err != nil {
			t.Fatalf("Error: %s\n", err.Error())
		}

		dec := decoder.NewGaussElimRLNCDecoder(pieceCount)
		flow(enc, dec)
	})

	t.Run("WithPieceSize", func(t *testing.T) {
		size := uint(2<<10 + rand.Intn(2<<10))
		pieceSize := uint(2<<5 + rand.Intn(2<<5))
		pieceCount := uint(math.Ceil(float64(size) / float64(pieceSize)))
		data := generateData(size)

		enc, err := encoder.NewSparseRLNCEncoderWithPieceSize(data, pieceSize, 0.5)
		if err != nil {
			t.Fatalf("Error: %s\n", err.Error())
		}

		dec := decoder.NewGaussElimRLNCDecoder(pieceCount)
		flow(enc, dec)
	})
}
