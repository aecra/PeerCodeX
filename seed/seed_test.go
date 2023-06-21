package seed_test

import (
	"crypto/rand"
	"os"
	"testing"

	"github.com/aecra/PeerCodeX/seed"
)

func TestCreateSeedFile(t *testing.T) {
	// create a temp file
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(f.Name())
	// write 200MB data
	data := make([]byte, 1<<20)
	for i := 0; i < 200; i++ {
		rand.Read(data)
		_, err := f.Write(data)
		if err != nil {
			t.Error(err)
		}
	}

	err = seed.CreateSeedFile(f.Name(), "This is a test", "127.0.0.1:8080", "")
	if err != nil {
		t.Error(err)
	}

	os.Remove(f.Name() + ".nc")
}
