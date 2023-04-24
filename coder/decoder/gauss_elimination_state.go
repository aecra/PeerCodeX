package decoder

import (
	"github.com/aecra/PeerCodeX/coder"
	galoisfield "github.com/aecra/PeerCodeX/coder/galoisfield/table"
	"github.com/aecra/PeerCodeX/coder/matrix"
)

type GaussElimDecoderState struct {
	field      *galoisfield.GF
	pieceCount uint
	coeffs     matrix.Matrix
	coded      matrix.Matrix

	coeffsLI matrix.Matrix
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func (d *GaussElimDecoderState) clean_forward() {
	var (
		rows     int = int(d.coeffs.Rows())
		cols     int = int(d.coeffs.Cols())
		boundary int = min(rows, cols)
	)

	for i := 0; i < boundary; i++ {
		if d.coeffs[i][i] == 0 {
			non_zero_col := false
			pivot := i + 1
			for ; pivot < rows; pivot++ {
				if d.coeffs[pivot][i] != 0 {
					non_zero_col = true
					break
				}
			}

			if !non_zero_col {
				continue
			}

			// row switching in coefficient matrix
			{
				tmp := d.coeffs[i]
				d.coeffs[i] = d.coeffs[pivot]
				d.coeffs[pivot] = tmp
			}
			// row switching in coded piece matrix
			{
				tmp := d.coded[i]
				d.coded[i] = d.coded[pivot]
				d.coded[pivot] = tmp
			}
		}

		for j := i + 1; j < rows; j++ {
			if d.coeffs[j][i] == 0 {
				continue
			}

			quotient := d.field.Div(d.coeffs[j][i], d.coeffs[i][i])
			for k := i; k < cols; k++ {
				d.coeffs[j][k] = d.field.Add(d.coeffs[j][k], d.field.Mul(d.coeffs[i][k], quotient))
			}

			for k := 0; k < len(d.coded[0]); k++ {
				d.coded[j][k] = d.field.Add(d.coded[j][k], d.field.Mul(d.coded[i][k], quotient))
			}
		}
	}
}

func (d *GaussElimDecoderState) clean_forward_li() {
	var (
		rows     int = int(d.coeffsLI.Rows())
		cols     int = int(d.coeffsLI.Cols())
		boundary int = min(rows, cols)
	)

	for i := 0; i < boundary; i++ {
		if d.coeffsLI[i][i] == 0 {
			non_zero_col := false
			pivot := i + 1
			for ; pivot < rows; pivot++ {
				if d.coeffsLI[pivot][i] != 0 {
					non_zero_col = true
					break
				}
			}

			if !non_zero_col {
				continue
			}

			// row switching in coefficient matrix
			{
				tmp := d.coeffsLI[i]
				d.coeffsLI[i] = d.coeffsLI[pivot]
				d.coeffsLI[pivot] = tmp
			}
		}

		for j := i + 1; j < rows; j++ {
			if d.coeffsLI[j][i] == 0 {
				continue
			}

			quotient := d.field.Div(d.coeffsLI[j][i], d.coeffsLI[i][i])
			for k := i; k < cols; k++ {
				d.coeffsLI[j][k] = d.field.Add(d.coeffsLI[j][k], d.field.Mul(d.coeffsLI[i][k], quotient))
			}
		}
	}
}

func (d *GaussElimDecoderState) clean_backward() {
	var (
		rows     int = int(d.coeffs.Rows())
		cols     int = int(d.coeffs.Cols())
		boundary int = min(rows, cols)
	)

	for i := boundary - 1; i >= 0; i-- {
		if d.coeffs[i][i] == 0 {
			continue
		}

		for j := 0; j < i; j++ {
			if d.coeffs[j][i] == 0 {
				continue
			}

			quotient := d.field.Div(d.coeffs[j][i], d.coeffs[i][i])
			for k := i; k < cols; k++ {
				d.coeffs[j][k] = d.field.Add(d.coeffs[j][k], d.field.Mul(d.coeffs[i][k], quotient))
			}

			for k := 0; k < len(d.coded[0]); k++ {
				d.coded[j][k] = d.field.Add(d.coded[j][k], d.field.Mul(d.coded[i][k], quotient))
			}

		}

		if d.coeffs[i][i] == 1 {
			continue
		}

		inv := d.field.Div(1, d.coeffs[i][i])
		d.coeffs[i][i] = 1
		for j := i + 1; j < cols; j++ {
			if d.coeffs[i][j] == 0 {
				continue
			}

			d.coeffs[i][j] = d.field.Mul(d.coeffs[i][j], inv)
		}

		for j := 0; j < len(d.coded[0]); j++ {
			d.coded[i][j] = d.field.Mul(d.coded[i][j], inv)
		}
	}
}

func (d *GaussElimDecoderState) clean_backward_li() {
	var (
		rows     int = int(d.coeffsLI.Rows())
		cols     int = int(d.coeffsLI.Cols())
		boundary int = min(rows, cols)
	)

	for i := boundary - 1; i >= 0; i-- {
		if d.coeffsLI[i][i] == 0 {
			continue
		}

		for j := 0; j < i; j++ {
			if d.coeffsLI[j][i] == 0 {
				continue
			}

			quotient := d.field.Div(d.coeffsLI[j][i], d.coeffsLI[i][i])
			for k := i; k < cols; k++ {
				d.coeffsLI[j][k] = d.field.Add(d.coeffsLI[j][k], d.field.Mul(d.coeffsLI[i][k], quotient))
			}
		}

		if d.coeffsLI[i][i] == 1 {
			continue
		}

		inv := d.field.Div(1, d.coeffsLI[i][i])
		d.coeffsLI[i][i] = 1
		for j := i + 1; j < cols; j++ {
			if d.coeffsLI[i][j] == 0 {
				continue
			}

			d.coeffsLI[i][j] = d.field.Mul(d.coeffsLI[i][j], inv)
		}
	}
}

func (d *GaussElimDecoderState) remove_zero_rows() {
	var (
		cols = len(d.coeffs[0])
	)

	for i := 0; i < len(d.coeffs); i++ {
		yes := true
		for j := 0; j < cols; j++ {
			if d.coeffs[i][j] != 0 {
				yes = false
				break
			}
		}
		if !yes {
			continue
		}

		// resize `coeffs` matrix
		d.coeffs[i] = nil
		copy((d.coeffs)[i:], (d.coeffs)[i+1:])
		d.coeffs = (d.coeffs)[:len(d.coeffs)-1]

		// resize `coded` matrix
		d.coded[i] = nil
		copy((d.coded)[i:], (d.coded)[i+1:])
		d.coded = (d.coded)[:len(d.coded)-1]

		i = i - 1
	}
}

func (d *GaussElimDecoderState) remove_zero_rows_li() {
	var (
		cols = len(d.coeffsLI[0])
	)

	for i := 0; i < len(d.coeffsLI); i++ {
		yes := true
		for j := 0; j < cols; j++ {
			if d.coeffsLI[i][j] != 0 {
				yes = false
				break
			}
		}
		if !yes {
			continue
		}

		// resize `coeffsLI` matrix
		d.coeffsLI[i] = nil
		copy((d.coeffsLI)[i:], (d.coeffsLI)[i+1:])
		d.coeffsLI = (d.coeffsLI)[:len(d.coeffsLI)-1]

		i = i - 1
	}
}

// Calculates Reduced Row Echelon Form of coefficient
// matrix, while also modifying coded piece matrix
// First it forward, backward cleans up matrix
// i.e. cells other than pivots are zeroed,
// later it checks if some rows of coefficient matrix
// are linearly dependent or not, if yes it removes those,
// while respective rows of coded piece matrix is also
// removed --- considered to be `not useful piece`
//
// Note: All operations are in-place, no more memory
// allocations are performed
func (d *GaussElimDecoderState) Rref() {
	d.clean_forward()
	d.clean_backward()
	d.remove_zero_rows()
}

func (d *GaussElimDecoderState) rref_li() {
	d.clean_forward_li()
	d.clean_backward_li()
	d.remove_zero_rows_li()
}

func (d *GaussElimDecoderState) IsLinearIndependent(vector coder.CodingVector) bool {
	if d.coeffs.Rows() == 0 {
		return true
	}

	if d.coeffsLI == nil {
		d.coeffsLI = make([][]byte, d.coeffs.Rows(), d.pieceCount)
		for i := 0; i < len(d.coeffsLI); i++ {
			d.coeffsLI[i] = make([]byte, d.coeffs.Cols())
		}
	}

	// adjust size of `coeffsLI` matrix to len(d.coeffs) + 1
	if len(d.coeffsLI) < int(d.coeffs.Rows())+1 {
		k := len(d.coeffsLI)
		d.coeffsLI = append(d.coeffsLI, make([][]byte, int(d.coeffs.Rows())+1-len(d.coeffsLI))...)
		for i := len(d.coeffsLI) - 1; i >= k; i-- {
			d.coeffsLI[i] = make([]byte, d.coeffs.Cols())
		}
	}

	// copy `d.coeffs` to `d.coeffsLI`
	for i := 0; i < len(d.coeffs); i++ {
		copy(d.coeffsLI[i], d.coeffs[i])
	}

	// copy `vector` to `d.coeffsLI`
	copy(d.coeffsLI[len(d.coeffs)], vector)

	// calculate rref of `d.coeffsLI`
	d.rref_li()
	if d.rank_li() == d.Rank() {
		return false
	}
	return true
}

// Expected to be invoked after RREF-ed, in other words
// it won't rref matrix first to calculate rank,
// rather that needs to first invoked
func (d *GaussElimDecoderState) Rank() uint {
	return d.coeffs.Rows()
}

func (d *GaussElimDecoderState) rank_li() uint {
	return d.coeffsLI.Rows()
}

// Current state of coding coefficient matrix
func (d *GaussElimDecoderState) CoefficientMatrix() matrix.Matrix {
	return d.coeffs
}

// Current state of coded piece matrix, which is updated
// along side coding coefficient matrix ( during rref )
func (d *GaussElimDecoderState) CodedPieceMatrix() matrix.Matrix {
	return d.coded
}

// Adds a new coded piece to decoder state, which will hopefully
// help in decoding pieces, if linearly independent with other rows
// i.e. read pieces
func (d *GaussElimDecoderState) AddPiece(codedPiece *coder.CodedPiece) {
	d.coeffs = append(d.coeffs, codedPiece.Vector)
	d.coded = append(d.coded, codedPiece.Piece)
}

// Request decoded piece by index ( 0 based, definitely )
//
// If piece not yet decoded/ requested index is >= #-of
// pieces coded together, returns error message indicating so
//
// # Otherwise piece is returned, without any error
//
// Note: This method will copy decoded piece into newly allocated memory
// when whole decoding hasn't yet happened, to prevent any chance
// that user mistakenly modifies slice returned ( read piece )
// & that affects next round of decoding ( when new piece is received )
func (d *GaussElimDecoderState) GetPiece(idx uint) (coder.Piece, error) {
	if idx >= d.pieceCount {
		return nil, coder.ErrPieceOutOfBound
	}
	if idx >= d.coeffs.Rows() {
		return nil, coder.ErrPieceNotDecodedYet
	}

	if d.Rank() >= d.pieceCount {
		return d.coded[idx], nil
	}

	cols := int(d.coeffs.Cols())
	decoded := true

OUT:
	for i := 0; i < cols; i++ {
		switch i {
		case int(idx):
			if d.coeffs[idx][i] != 1 {
				decoded = false
				break OUT
			}

		default:
			if d.coeffs[idx][i] == 0 {
				decoded = false
				break OUT
			}

		}
	}

	if !decoded {
		return nil, coder.ErrPieceNotDecodedYet
	}

	buf := make([]byte, d.coded.Cols())
	copy(buf, d.coded[idx])
	return buf, nil
}

func NewGaussElimDecoderStateWithPieceCount(gf *galoisfield.GF, pieceCount uint) *GaussElimDecoderState {
	coeffs := make([][]byte, 0, pieceCount)
	coded := make([][]byte, 0, pieceCount)
	return &GaussElimDecoderState{field: gf, pieceCount: pieceCount, coeffs: coeffs, coded: coded}
}

func NewGaussElimDecoderState(gf *galoisfield.GF, coeffs, coded matrix.Matrix) *GaussElimDecoderState {
	return &GaussElimDecoderState{field: gf, pieceCount: uint(len(coeffs)), coeffs: coeffs, coded: coded}
}
