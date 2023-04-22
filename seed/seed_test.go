package seed_test

import (
	"testing"

	"github.com/aecra/PeerCodeX/seed"
)

func TestCreateSeedFile(t *testing.T) {
	err := seed.CreateSeedFile("/home/aecra/project/PeerCodeX/第01集 游戏高手.mp4", "This is a test", "127.0.0.1:8080", "")
	if err != nil {
		t.Error(err)
	}
}
