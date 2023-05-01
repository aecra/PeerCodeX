package encoder

import (
	"math/rand"

	"github.com/aecra/PeerCodeX/coder"
	galoisfield "github.com/aecra/PeerCodeX/coder/galoisfield/table"
)

type SparseRLNCEncoder struct {
	field          *galoisfield.GF
	probability    float64
	currentPieceId uint
	pieces         []coder.Piece
	extra          uint
}

// Total #-of pieces being coded together --- denoting
// these many linearly independent pieces are required
// successfully decoding back to original pieces
func (s *SparseRLNCEncoder) PieceCount() uint {
	return uint(len(s.pieces))
}

// Pieces which are coded together are all of same size
//
// Total data being coded = pieceSize * pieceCount ( may include
// some padding bytes )
func (s *SparseRLNCEncoder) PieceSize() uint {
	return uint(len(s.pieces[0]))
}

// How many bytes of data, constructed by concatenating
// coded pieces together, required at minimum for decoding
// back to original pieces ?
//
// As I'm coding N-many pieces together, I need at least N-many
// linearly independent pieces, which are concatenated together
// to form a byte slice & can be used for original data reconstruction.
//
// So it computes N * codedPieceLen
func (s *SparseRLNCEncoder) DecodableLen() uint {
	return s.PieceCount() * s.CodedPieceLen()
}

// If N-many original pieces are coded together
// what could be length of one such coded piece
// obtained by invoking `CodedPiece` ?
//
// Here N = len(pieces), original pieces which are
// being coded together
func (s *SparseRLNCEncoder) CodedPieceLen() uint {
	return s.PieceCount() + s.PieceSize()
}

// How many extra padding bytes added at end of
// original data slice so that splitted pieces are
// all of same size ?
func (s *SparseRLNCEncoder) Padding() uint {
	return s.extra
}

// Generates a systematic coded piece's coding vector, which has
// only one non-zero element ( 1 )
func (s *SparseRLNCEncoder) systematicCodingVector(idx uint) coder.CodingVector {
	if idx >= s.PieceCount() {
		return nil
	}

	vector := make(coder.CodingVector, s.PieceCount())
	vector[idx] = 1
	return vector
}

// Returns a coded piece, which is constructed on-the-fly
// by randomly drawing elements from finite field i.e.
// coding coefficients & performing sparse-RLNC with
// all original pieces
func (s *SparseRLNCEncoder) CodedPiece() *coder.CodedPiece {
	if s.currentPieceId < s.PieceCount() {
		// `nil` coding vector can be returned, which is
		// not being checked at all, as in that case we'll
		// never get into `if` branch
		vector := s.systematicCodingVector(s.currentPieceId)
		piece := make(coder.Piece, s.PieceSize())
		copy(piece, s.pieces[s.currentPieceId])

		s.currentPieceId++
		return &coder.CodedPiece{
			Vector: vector,
			Piece:  piece,
		}
	}

	vector := coder.GenerateCodingVector(s.PieceCount())
	// set some elements to zero
	for i := range vector {
		if rand.Float64() <= s.probability {
			vector[i] = 0
		}
	}
	piece := make(coder.Piece, s.PieceSize())
	for i := range s.pieces {
		piece.Multiply(s.pieces[i], vector[i], s.field)
	}
	return &coder.CodedPiece{
		Vector: vector,
		Piece:  piece,
	}
}

// Provide with original pieces on which sparseRLNC to be performed
// & get encoder, to be used for on-the-fly generation
// to N-many coded pieces
func NewSparseRLNCEncoder(pieces []coder.Piece, probability float64) Encoder {
	return &SparseRLNCEncoder{
		pieces:         pieces,
		field:          galoisfield.DefaultGF256,
		probability:    probability,
		currentPieceId: 0,
	}
}

// If you know #-of pieces you want to code together, invoking
// this function splits whole data chunk into N-pieces, with padding
// bytes appended at end of last piece, if required & prepares
// sparse RLNC encoder for obtaining coded pieces
func NewSparseRLNCEncoderWithPieceCount(data []byte, pieceCount uint, probability float64) (Encoder, error) {
	pieces, padding, err := coder.OriginalPiecesFromDataAndPieceCount(data, pieceCount)
	if err != nil {
		return nil, err
	}

	// make sure probability is not too high
	if probability > 1-float64(6)/float64(pieceCount) {
		probability = 1 - float64(6)/float64(pieceCount)
	}

	enc := NewSparseRLNCEncoder(pieces, probability)
	fenc := enc.(*SparseRLNCEncoder)
	fenc.extra = padding
	return fenc, nil
}

// If you want to have N-bytes piece size for each, this
// function generates M-many pieces each of N-bytes size, which are ready
// to be coded together with sparse RLNC
func NewSparseRLNCEncoderWithPieceSize(data []byte, pieceSize uint, probability float64) (Encoder, error) {
	pieces, padding, err := coder.OriginalPiecesFromDataAndPieceSize(data, pieceSize)
	if err != nil {
		return nil, err
	}

	enc := NewSparseRLNCEncoder(pieces, probability)
	fenc := enc.(*SparseRLNCEncoder)
	fenc.extra = padding
	return fenc, nil
}
