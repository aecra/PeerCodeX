package matrix_test

import (
	"bytes"
	"errors"
	"testing"

	galoisfield "github.com/aecra/PeerCodeX/coder/galoisfield/table"
	"github.com/aecra/PeerCodeX/coder/matrix"
	"github.com/aecra/PeerCodeX/coder"
)

func TestMatrixMultiplication(t *testing.T) {
	field := galoisfield.DefaultGF256

	m_1 := matrix.Matrix{{102, 82, 165, 0}}
	m_2 := matrix.Matrix{{157, 233, 247}, {160, 28, 233}, {149, 234, 117}, {200, 181, 55}}
	m_3 := matrix.Matrix{{1, 2, 3}}
	expected := matrix.Matrix{{186, 23, 11}}

	if _, err := m_3.Multiply(field, m_2); !(err != nil && errors.Is(err, coder.ErrMatrixDimensionMismatch)) {
		t.Fatal("expected failed matrix multiplication error indication")
	}

	mult, err := m_1.Multiply(field, m_2)
	if err != nil {
		t.Fatal(err.Error())
	}

	for i := 0; i < int(expected.Rows()); i++ {
		if !bytes.Equal(expected[i], mult[i]) {
			t.Fatal("row mismatch !")
		}
	}
}
