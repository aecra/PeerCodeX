package recoder_test

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/coder/decoder"
	"github.com/aecra/PeerCodeX/coder/encoder"
	"github.com/aecra/PeerCodeX/coder/recoder"
)

// Generates `N`-bytes of random data from default
// randomization source
func generateData(n uint) []byte {
	data := make([]byte, n)
	// can safely ignore error
	rand.Read(data)
	return data
}

// Generates N-many pieces each of M-bytes length, to be used
// for testing purposes
func generatePieces(pieceCount uint, pieceLength uint) []coder.Piece {
	pieces := make([]coder.Piece, 0, pieceCount)
	for i := 0; i < int(pieceCount); i++ {
		pieces = append(pieces, generateData(pieceLength))
	}
	return pieces
}

func recoderFlow(t *testing.T, rec recoder.Recoder, pieceCount int, pieces []coder.Piece) {
	dec := decoder.NewGaussElimRLNCDecoder(uint(pieceCount))
	for {
		r_piece, err := rec.CodedPiece()
		if err != nil {
			t.Fatalf("Error: %s\n", err.Error())
		}
		if err := dec.AddPiece(r_piece); errors.Is(err, coder.ErrAllUsefulPiecesReceived) {
			break
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

func TestNewFullRLNCRecoder(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	pieceCount := 128
	pieceLength := 8192
	codedPieceCount := pieceCount + 2
	pieces := generatePieces(uint(pieceCount), uint(pieceLength))
	enc := encoder.NewFullRLNCEncoder(pieces)

	coded := make([]*coder.CodedPiece, 0, codedPieceCount)
	for i := 0; i < codedPieceCount; i++ {
		coded = append(coded, enc.CodedPiece())
	}

	rec := recoder.NewFullRLNCRecoder(coded)
	recoderFlow(t, rec, pieceCount, pieces)
}

func TestNewFullRLNCRecoderWithFlattenData(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	pieceCount := 128
	pieceLength := 8192
	codedPieceCount := pieceCount + 2
	pieces := generatePieces(uint(pieceCount), uint(pieceLength))
	enc := encoder.NewFullRLNCEncoder(pieces)

	coded := make([]*coder.CodedPiece, 0, codedPieceCount)
	for i := 0; i < codedPieceCount; i++ {
		coded = append(coded, enc.CodedPiece())
	}

	codedFlattened := make([]byte, 0)
	for i := 0; i < len(coded); i++ {
		codedFlattened = append(codedFlattened, coded[i].Flatten()...)
	}

	rec, err := recoder.NewFullRLNCRecoderWithFlattenData(codedFlattened, uint(codedPieceCount), uint(pieceCount))
	if err != nil {
		t.Fatal(err.Error())
	}

	recoderFlow(t, rec, pieceCount, pieces)
}
