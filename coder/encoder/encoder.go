package encoder

import "github.com/aecra/PeerCodeX/coder"

type Encoder interface {
	PieceCount() uint
	PieceSize() uint
	DecodableLen() uint
	CodedPieceLen() uint
	Padding() uint
	CodedPiece() *coder.CodedPiece
}
