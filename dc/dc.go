package dc

import (
	"errors"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/coder/decoder"
	"github.com/aecra/PeerCodeX/coder/encoder"
	"github.com/aecra/PeerCodeX/coder/recoder"
	"github.com/aecra/PeerCodeX/seed"
	"github.com/aecra/PeerCodeX/tools"
)

type FileItem struct {
	NcFile            *seed.NcFile
	Path              string
	IsDownloaded      bool
	IsDownloading     bool
	ProcessRate       float64
	HideRefresh       func()
	Nodes             []*NodeItem
	Conns             []net.Conn
	Encoder           encoder.Encoder
	Decoder           decoder.Decoder
	Recoder           recoder.Recoder
	AddCodedPieceChan chan *coder.CodedPiece
	isReceiveStarted  bool
}

type NodeItem struct {
	Addr       string
	IsOn       bool
	HaveClient bool
}

var (
	FileList = make([]*FileItem, 0)
	host     = "0.0.0.0"
	port     = "8080"
)

func GetFileItemByPath(path string) *FileItem {
	for _, item := range FileList {
		if item.Path == path {
			return item
		}
	}
	return nil
}

func AddFileItem(path string) error {
	// if file is already in the list, do nothing
	for _, f := range FileList {
		if f.Path == path {
			return errors.New("file is already in the list")
		}
	}

	ncfile, err := seed.NewNcFileFromSeedFile(path)
	if err != nil {
		return err
	}

	fileItem := &FileItem{
		NcFile:            ncfile,
		Path:              path,
		Nodes:             make([]*NodeItem, 0),
		Conns:             make([]net.Conn, 0),
		AddCodedPieceChan: make(chan *coder.CodedPiece, 100),
		isReceiveStarted:  false,
	}

	isDownloaded := ncfile.IsFileDownloaded(filepath.Dir(path))

	if isDownloaded {
		fileItem.IsDownloaded = true
		fileItem.IsDownloading = false
		fileItem.ProcessRate = 1
		// open file to get file data
		f, err := os.Open(filepath.Join(filepath.Dir(path), ncfile.Info.Name))
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := os.ReadFile(f.Name())
		if err != nil {
			return err
		}
		enc, err := encoder.NewFullRLNCEncoderWithPieceSize(data, uint(fileItem.NcFile.Info.PieceLength))
		if err != nil {
			return err
		}
		fileItem.Encoder = enc
	} else {
		fileItem.IsDownloaded = false
		fileItem.IsDownloading = false
		fileItem.ProcessRate = 0
		fileItem.Decoder = decoder.NewGaussElimRLNCDecoder(uint(math.Ceil(float64(ncfile.Info.Length) / float64(ncfile.Info.PieceLength))))
	}
	// add announce and announce-list
	announce := ncfile.Announce
	if announce != "" {
		// add announce
		fileItem.Nodes = append(fileItem.Nodes, &NodeItem{Addr: announce, IsOn: true, HaveClient: false})
	}
	announceList := ncfile.AnnounceList
	if announceList != nil {
		// add announce-list
		for _, item := range announceList {
			fileItem.Nodes = append(fileItem.Nodes, &NodeItem{Addr: item, IsOn: true, HaveClient: false})
		}
	}

	FileList = append(FileList, fileItem)
	return nil
}

func DeleteFileItemByPath(path string) {
	for i, item := range FileList {
		if item.Path == path {
			FileList = append(FileList[:i], FileList[i+1:]...)
		}
	}
}

func IsFileExist(hash []byte) bool {
	for _, item := range FileList {
		if tools.CompareHash(hash, item.NcFile.Info.Hash) {
			return true
		}
	}
	return false
}

func GetHost() string {
	return host
}

func SetHost(h string) {
	host = h
}

func GetPort() string {
	return port
}

func SetPort(p string) {
	port = p
}

func GetNeighbours(hash []byte) []*NodeItem {
	// return atmost 10 neighbours
	for _, item := range FileList {
		if tools.CompareHash(hash, item.NcFile.Info.Hash) {
			if len(item.Nodes) > 10 {
				return item.Nodes[:10]
			}
			return item.Nodes
		}
	}
	return nil
}

func GetCodedPiece(hash []byte) *coder.CodedPiece {
	for _, item := range FileList {
		if tools.CompareHash(hash, item.NcFile.Info.Hash) {
			// if encoder is not nil, use encoder
			// or use recoder
			if item.Encoder != nil {
				return item.Encoder.CodedPiece()
			}
			if item.Recoder != nil {
				codedPiece, err := item.Recoder.CodedPiece()
				if err != nil {
					return nil
				}
				return codedPiece
			}
		}
	}
	return nil
}

func (f *FileItem) AddCodedPiece(codedPiece *coder.CodedPiece) {
	if f.IsDownloaded {
		return
	}
	if f.Decoder == nil {
		return
	}
	f.Decoder.AddPiece(codedPiece)
	if f.Recoder == nil {
		ps := make([]*coder.CodedPiece, 1)
		ps[0] = codedPiece
		f.Recoder = recoder.NewFullRLNCRecoder(ps)
	}
	f.Recoder.AddCodedPiece(codedPiece)

	log.Println(f.Path, "still need", f.Decoder.Required())

	if f.Decoder.IsDecoded() {
		filePath := f.Path[:len(f.Path)-len(filepath.Ext(f.Path))]
		pieces, err := f.Decoder.GetPieces()
		if err != nil {
			return
		}

		// write pieces to file
		file, err := os.Create(filePath)
		if err != nil {
			return
		}
		defer file.Close()
		currentLength := 0
		for _, piece := range pieces {
			if len(piece)+currentLength > int(f.NcFile.Info.Length) {
				_, err = file.Write(piece[:int(f.NcFile.Info.Length)-currentLength])
				if err != nil {
					return
				}
			}
			_, err = file.Write(piece)
			if err != nil {
				return
			}
		}

		f.IsDownloaded = true
		f.IsDownloading = false
		f.ProcessRate = 1
	}
}

func (f *FileItem) StartReceiveCodedPiece() {
	if f.isReceiveStarted {
		return
	}
	f.isReceiveStarted = true
	go func() {
		for {
			codedPiece := <-f.AddCodedPieceChan
			f.AddCodedPiece(codedPiece)
			if f.IsDownloaded {
				log.Println(f.Path, "is downloaded")
				// stop all connections
				for _, conn := range f.Conns {
					if conn == nil {
						continue
					}
					conn.Close()
				}
				// clear f.Conns
				f.Conns = []net.Conn{}
				for _, node := range f.Nodes {
					node.HaveClient = false
				}
				f.HideRefresh()

				// remove decoder and recoder, add encoder
				pieces, _ := f.Decoder.GetPieces()
				data := make([]byte, 0, f.NcFile.Info.Length)
				currentLength := 0
				for _, piece := range pieces {
					if len(piece)+currentLength > int(f.NcFile.Info.Length) {
						data = append(data, piece[:int(f.NcFile.Info.Length)-currentLength]...)
						return
					}
					data = append(data, piece...)
				}
				enc, _ := encoder.NewFullRLNCEncoderWithPieceSize(data, uint(f.NcFile.Info.PieceLength))
				f.Decoder = nil
				f.Recoder = nil
				f.Encoder = enc
				break
			}
		}
	}()
}

func GetNodeStatusList() []*NodeItem {
	nodeItems := make([]*NodeItem, 0)
	for _, item := range FileList {
		for _, node := range item.Nodes {
			nodeItems = append(nodeItems, node)
		}
	}
	result := []*NodeItem{}
	temp := map[string]struct{}{}
	for _, item := range nodeItems {
		if _, ok := temp[item.Addr]; !ok {
			temp[item.Addr] = struct{}{}
			result = append(result, item)
		}
	}
	// if there is a node is on, set it to true
	for i, item := range result {
		for _, node := range nodeItems {
			if item.Addr == node.Addr && node.IsOn {
				result[i].IsOn = true
			}
		}
	}
	return result
}

func AddNode(address string) {
	for _, item := range FileList {
		for _, node := range item.Nodes {
			if node.Addr == address {
				return
			}
		}
		item.Nodes = append(item.Nodes, &NodeItem{
			Addr:       address,
			IsOn:       false,
			HaveClient: false,
		})
	}
}

func UpdateNodeStatus(address string, status bool) {
	for i, item := range FileList {
		for j, node := range item.Nodes {
			if node.Addr == address {
				FileList[i].Nodes[j].IsOn = status
			}
		}
	}
}

func DeleteNode(address string) {
	for i, item := range FileList {
		for j, node := range item.Nodes {
			if node.Addr == address {
				FileList[i].Nodes = append(item.Nodes[:j], item.Nodes[j+1:]...)
			}
		}
	}
}
