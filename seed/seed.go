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
		Name   string   `bencode:"name"`
		Hash   [][]byte `bencode:"hash"`
		Length int64    `bencode:"length"`
	} `bencode:"info"`
}

func (f *NcFile) GenarateInfo(path string) error {
	// generate NcInfo from path
	// if path is a file, then SingleFile is true
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("directory is not supported yet")
	} else {
		f.Info.Name = info.Name()
		hashs, err := tools.GetHashsofFile(path)
		if err != nil {
			return err
		}
		f.Info.Hash = hashs
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
	announcelist := make([]string, 0)
	for _, v := range f.AnnounceList {
		if v != "" {
			announcelist = append(announcelist, v)
		}
	}
	f.AnnounceList = announcelist
	return nil
}

func (f *NcFile) IsFileDownloaded(dir string) ([]bool, error) {
	result := make([]bool, len(f.Info.Hash))
	if f.Info.Length == 0 {
		return nil, errors.New("This is a file is empty")
	}

	fi, err := os.Stat(filepath.Join(dir, f.Info.Name))
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}
	if fi.Size() != f.Info.Length {
		return result, errors.New("the size of existing file is not equal to the size of seed file")
	}

	hashs, err := tools.GetHashsofFile(filepath.Join(dir, f.Info.Name))
	if err != nil {
		return result, err
	}

	if len(f.Info.Hash) != len(hashs) {
		return nil, errors.New("the size of existing file is not equal to the size of seed file")
	}
	for i := 0; i < len(f.Info.Hash); i++ {
		if tools.CompareHash(f.Info.Hash[i], hashs[i]) {
			result[i] = true
		}
	}

	return result, nil
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

func CreateSeedFile(path string, comment string, announce string, announceList string) error {
	// create seed from path
	ncFile := NcFile{
		Announce:     announce,
		AnnounceList: strings.Split(announceList, ","),
		Comment:      comment,
		CreateBy:     "PeerCodeX 0.0.1",
		CreationDate: time.Now(),
	}
	err := ncFile.GenarateInfo(path)
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
