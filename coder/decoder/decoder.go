package decoder

import "github.com/aecra/PeerCodeX/coder"

type Decoder interface {
	PieceLength() uint
	IsDecoded() bool
	Required() uint
	AddPiece(piece *coder.CodedPiece) error
	GetPiece(index uint) (coder.Piece, error)
	GetPieces() ([]coder.Piece, error)
}
