package decoder_test

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/aecra/PeerCodeX/coder/decoder"
	"github.com/aecra/PeerCodeX/coder/encoder"
	"github.com/aecra/PeerCodeX/coder"
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

func TestNewGaussElimRLNCDecoder(t *testing.T) {
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

	dec := decoder.NewGaussElimRLNCDecoder(uint(pieceCount))
	neededPieceCount := uint(pieceCount)
	for i := 0; i < codedPieceCount; i++ {

		// test whether required piece count is monotonically decreasing or not
		switch i {
		case 0:
			if req_ := dec.Required(); req_ != neededPieceCount {
				t.Fatalf("expected still needed piece count to be %d, found it to be %d\n", neededPieceCount, req_)
			}
			// skip unnecessary assignment to `needPieceCount`

		default:
			if req_ := dec.Required(); !(req_ <= neededPieceCount) {
				t.Fatal("expected required piece count to monotonically decrease")
			} else {
				neededPieceCount = req_
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
