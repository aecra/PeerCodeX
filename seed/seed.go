package seed

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aecra/PeerCodeX/tools"
	"github.com/zeebo/bencode"
)

type NcFile struct {
	Announce     string   `bencode:"announce"`
	AnnounceList []string `bencode:"announce-list"`

	Comment string `bencode:"comment"`

	CreateBy     string    `bencode:"created by"`
	CreationDate time.Time `bencode:"creation date"`

	Info struct {
		Name        string `bencode:"name"`
		PieceLength int64  `bencode:"piece length"`
		Hash        []byte `bencode:"hash"`
		Length      int64  `bencode:"length"`
	} `bencode:"info"`
}

type NcInfo struct {
	Name        string
	PieceLength int64
	Hash        []byte
	Length      int64
}

func (f *NcFile) GenarateInfo(path string, pieceLength int64) error {
	// generate NcInfo from path
	// if path is a file, then SingleFile is true
	f.Info.PieceLength = (1 << 10) * pieceLength
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("directory is not supported yet")
	} else {
		f.Info.Name = info.Name()
		hash, err := tools.GetHashofFile(path)
		if err != nil {
			return err
		}
		f.Info.Hash = hash
		f.Info.Length = info.Size()
	}
	return nil
}

func (f *NcFile) Bencoding() (res []byte, err error) {
	// convert NcFile to BitTorrent bencoding
	res, err = bencode.EncodeBytes(f)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (f *NcFile) Save(file *os.File) error {
	// save NcFile to file
	bencoding, err := f.Bencoding()
	if err != nil {
		return err
	}
	_, err = file.Write(bencoding)
	if err != nil {
		return err
	}
	return nil
}

func (f *NcFile) Load(file *os.File) error {
	// load NcFile from file
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	bencode.DecodeBytes(content, f)
	return nil
}

func (f *NcFile) IsFileDownloaded(dir string) bool {
	if f.Info.Length == 0 {
		return false
	}
	hash, err := tools.GetHashofFile(filepath.Join(dir, f.Info.Name))
	if err != nil {
		return false
	}
	if !tools.CompareHash(f.Info.Hash, hash) {
		return false
	}

	return true
}

func NewNcFileFromSeedFile(path string) (*NcFile, error) {
	ncFile := NcFile{}
	// open seed file
	seedFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer seedFile.Close()
	// load NcFile from seed file
	err = ncFile.Load(seedFile)
	if err != nil {
		return nil, err
	}
	return &ncFile, nil
}

func CreateSeedFile(path string, pieceLength int, comment string, announce string, announceList string) error {
	// create seed from path
	ncFile := NcFile{
		Announce:     announce,
		AnnounceList: strings.Split(announceList, ","),
		Comment:      comment,
		CreateBy:     "PeerCodeX 0.0.1",
		CreationDate: time.Now(),
	}
	err := ncFile.GenarateInfo(path, int64(pieceLength))
	if err != nil {
		return err
	}
	// get name of seed
	seedName := ncFile.Info.Name + ".nc"
	// create seed file
	seedFile, err := os.Create(filepath.Join(filepath.Dir(path), seedName))
	if err != nil {
		return err
	}
	defer seedFile.Close()
	// save NcFile to seed file
	err = ncFile.Save(seedFile)
	if err != nil {
		return err
	}
	fmt.Println("Seed file created: " + seedName)
	return nil
}
