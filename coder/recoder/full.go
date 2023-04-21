package recoder

import (
	"github.com/aecra/PeerCodeX/coder"
	galoisfield "github.com/aecra/PeerCodeX/coder/galoisfield/table"
	"github.com/aecra/PeerCodeX/coder/matrix"
)

type FullRLNCRecoder struct {
	field        *galoisfield.GF
	pieces       []*coder.CodedPiece
	codingMatrix matrix.Matrix
}

func (r *FullRLNCRecoder) fill() {
	codingMatrix := make(matrix.Matrix, len(r.pieces))
	for i := 0; i < len(r.pieces); i++ {
		codingMatrix[i] = make([]byte, len(r.pieces[i].Vector))
		copy(codingMatrix[i], r.pieces[i].Vector)
	}
	r.codingMatrix = codingMatrix
}

// Returns recoded piece, which is constructed on-the-fly
// by randomly drawing some coding coefficients from
// finite field & performing full RLNC with all coded pieces
func (r *FullRLNCRecoder) CodedPiece() (*coder.CodedPiece, error) {
	pieceCount := uint(len(r.pieces))
	vector := coder.GenerateCodingVector(pieceCount)
	piece := make(coder.Piece, len(r.pieces[0].Piece))
	for i := range r.pieces {
		piece.Multiply(r.pieces[i].Piece, vector[i], r.field)
	}

	vector_ := matrix.Matrix{vector}
	mult, err := vector_.Multiply(r.field, r.codingMatrix)
	if err != nil {
		return nil, err
	}

	return &coder.CodedPiece{
		Vector: mult[0],
		Piece:  piece,
	}, nil
}

func (r *FullRLNCRecoder) AddCodedPiece(piece *coder.CodedPiece) {
	r.pieces = append(r.pieces, piece)
	r.fill()
}

// Provide with all coded pieces, which are to be used
// for performing fullRLNC ( read recoding of coded data )
// & get back recoder which is used for on-the-fly construction
// of N-many recoded pieces
func NewFullRLNCRecoder(pieces []*coder.CodedPiece) *FullRLNCRecoder {
	rec := &FullRLNCRecoder{field: galoisfield.DefaultGF256, pieces: pieces}
	rec.fill()
	return rec
}

// A byte slice which is formed by concatenating coded pieces,
// will be splitted into structured coded pieces ( read having two components
// i.e. coding vector & piece ) & recoder to be returned, which can be used
// for on-the-fly random piece recoding
func NewFullRLNCRecoderWithFlattenData(data []byte, pieceCount uint, piecesCodedTogether uint) (*FullRLNCRecoder, error) {
	codedPieces, err := coder.CodedPiecesForRecoding(data, pieceCount, piecesCodedTogether)
	if err != nil {
		return nil, err
	}

	return NewFullRLNCRecoder(codedPieces), nil
}
