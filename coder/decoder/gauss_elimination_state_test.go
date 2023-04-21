package decoder_test

import (
	"testing"

	"github.com/aecra/PeerCodeX/coder/decoder"
	galoisfield "github.com/aecra/PeerCodeX/coder/galoisfield/table"
	"github.com/aecra/PeerCodeX/coder/matrix"
)

func TestMatrixRref(t *testing.T) {
	field := galoisfield.DefaultGF256

	{
		m := matrix.Matrix{{70, 137, 2, 152}, {223, 92, 234, 98}, {217, 141, 33, 44}, {145, 135, 71, 45}}
		m_rref := matrix.Matrix{{1, 0, 0, 105}, {0, 1, 0, 181}, {0, 0, 1, 42}}
		coded := matrix.Matrix{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}

		dec := decoder.NewGaussElimDecoderState(field, m, coded)
		dec.Rref()
		res := dec.CoefficientMatrix()
		if !res.Cmp(m_rref) {
			t.Fatal("rref doesn't match !")
		}
	}

	{
		m := matrix.Matrix{{68, 54, 6, 230}, {16, 56, 215, 78}, {159, 186, 146, 163}, {122, 41, 205, 133}}
		m_rref := matrix.Matrix{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}}
		coded := matrix.Matrix{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}

		dec := decoder.NewGaussElimDecoderState(field, m, coded)
		dec.Rref()
		res := dec.CoefficientMatrix()
		if !res.Cmp(m_rref) {
			t.Fatal("rref doesn't match !")
		}
	}

	{
		m := matrix.Matrix{{100, 31, 76, 199, 119}, {207, 34, 207, 208, 18}, {62, 20, 54, 6, 187}, {66, 8, 52, 73, 54}, {122, 138, 247, 211, 165}}
		m_rref := matrix.Matrix{{1, 0, 0, 0, 0}, {0, 1, 0, 0, 0}, {0, 0, 1, 0, 0}, {0, 0, 0, 1, 0}, {0, 0, 0, 0, 1}}
		coded := matrix.Matrix{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}

		dec := decoder.NewGaussElimDecoderState(field, m, coded)
		dec.Rref()
		res := dec.CoefficientMatrix()
		if !res.Cmp(m_rref) {
			t.Fatal("rref doesn't match !")
		}
	}
}

func TestMatrixRank(t *testing.T) {
	field := galoisfield.DefaultGF256

	{
		m := matrix.Matrix{{70, 137, 2, 152}, {223, 92, 234, 98}, {217, 141, 33, 44}, {145, 135, 71, 45}}
		coded := matrix.Matrix{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}

		dec := decoder.NewGaussElimDecoderState(field, m, coded)
		dec.Rref()
		if rank := dec.Rank(); rank != 3 {
			t.Fatalf("expected rank 3, received %d", rank)
		}
	}

	{

		m := matrix.Matrix{{68, 54, 6, 230}, {16, 56, 215, 78}, {159, 186, 146, 163}, {122, 41, 205, 133}}
		coded := matrix.Matrix{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}

		dec := decoder.NewGaussElimDecoderState(field, m, coded)
		dec.Rref()
		if rank := dec.Rank(); rank != 4 {
			t.Fatalf("expected rank 4, received %d", rank)
		}
	}

	{
		m := matrix.Matrix{{100, 31, 76, 199, 119}, {207, 34, 207, 208, 18}, {62, 20, 54, 6, 187}, {66, 8, 52, 73, 54}, {122, 138, 247, 211, 165}}
		coded := matrix.Matrix{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}

		dec := decoder.NewGaussElimDecoderState(field, m, coded)
		dec.Rref()
		if rank := dec.Rank(); rank != 5 {
			t.Fatalf("expected rank 5, received %d", rank)
		}
	}
}
