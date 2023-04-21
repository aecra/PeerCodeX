package recoder

import "github.com/aecra/PeerCodeX/coder"

type Recoder interface {
	fill()
	AddCodedPiece(piece *coder.CodedPiece)
	CodedPiece() (*coder.CodedPiece, error)
}
