package dc

import (
	"math"
	"path/filepath"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/seed"
	"github.com/aecra/PeerCodeX/tools"
)

type File struct {
	NcFile      *seed.NcFile
	Path        string
	Generations []*Generation
}

func NewFile(path string) (*File, error) {
	ncfile, err := seed.NewNcFileFromSeedFile(path)
	if err != nil {
		return nil, err
	}

	file := &File{
		NcFile:      ncfile,
		Path:        path,
		Generations: make([]*Generation, len(ncfile.Info.Hash)),
	}
	isDownloadedBools, err := ncfile.IsFileDownloaded(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	announceList := make([]string, 0)
	announceList = append(announceList, ncfile.Announce)
	for _, item := range announceList {
		isExist := false
		for _, announce := range announceList {
			if announce == item {
				isExist = true
				break
			}
		}
		if !isExist {
			announceList = append(announceList, item)
		}
	}
	for i, h := range ncfile.Info.Hash {
		file.Generations[i] = NewGeneration(file, h, announceList, isDownloadedBools[i])
	}
	return file, nil
}

func (f *File) AddCodedPiece(hash []byte, codedPiece *coder.CodedPiece) {
	for _, g := range f.Generations {
		if tools.CompareHash(g.Hash, hash) {
			g.AddCodedPiece(codedPiece)
			return
		}
	}
}

func (f *File) StartReceivingCodedPiece() {
	for _, h := range f.NcFile.Info.Hash {
		f.StartReceiving(h)
	}
}

func (f *File) StartReceiving(hash []byte) {
	for _, g := range f.Generations {
		if tools.CompareHash(g.Hash, hash) {
			g.StartReceiving()
			return
		}
	}
}

func (f *File) StopReceivingCodedPiece() {
	for _, h := range f.NcFile.Info.Hash {
		f.StopReceiving(h)
	}
}

func (f *File) StopReceiving(hash []byte) {
	for _, g := range f.Generations {
		if tools.CompareHash(g.Hash, hash) {
			g.StopReceiving()
			return
		}
	}
}

func (f *File) GetSerialNumber(hash []byte) uint {
	for i, h := range f.NcFile.Info.Hash {
		if tools.CompareHash(h, hash) {
			return uint(i)
		}
	}
	return 0
}

func (f *File) GetPieceCount(hash []byte) uint {
	return uint(math.Ceil(float64(f.GetGenerationLength(hash)) / float64(1<<20)))
}

func (f *File) GetGenerationLength(hash []byte) uint {
	if f.GetSerialNumber(hash) == uint(len(f.NcFile.Info.Hash)-1) {
		return uint(f.NcFile.Info.Length % (1 << 27))
	}
	return 1 << 27
}

func (f *File) GetTargetFile() string {
	return f.Path[:len(f.Path)-len(filepath.Ext(f.Path))]
}

func (f *File) AddNode(addr string) {
	for _, g := range f.Generations {
		g.AddNode(addr)
	}
}

func (f *File) DeleteNode(addr string) {
	for _, g := range f.Generations {
		g.DeleteNode(addr)
	}
}

func (f *File) GetProcessRate() float64 {
	decodedSize := uint(0)
	for _, g := range f.Generations {
		decodedSize += g.GetDecodedSize()
	}
	return float64(decodedSize) / float64(f.NcFile.Info.Length)
}

func (f *File) IsDownloading() bool {
	for _, g := range f.Generations {
		if g.IsDownloading() {
			return true
		}
	}
	return false
}

func (f *File) DropIdleEncoder() {
	for _, g := range f.Generations {
		g.DropIdleEncoder()
	}
}
