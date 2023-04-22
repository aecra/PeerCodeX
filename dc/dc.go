package dc

import (
	"encoding/hex"
	"errors"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	Nodes             []*NodeItem
	IsDownloaded      map[string]bool
	IsDownloading     map[string]bool
	ProcessRate       map[string]float64
	Conns             map[string][]net.Conn
	ConnsMutex        map[string]*sync.Mutex
	Encoder           map[string]encoder.Encoder
	Decoder           map[string]decoder.Decoder
	Recoder           map[string]recoder.Recoder
	AddCodedPieceChan map[string]chan *coder.CodedPiece
	isReceiveStarted  map[string]bool
}

type NodeItem struct {
	Addr       string
	IsOn       bool
	HaveClient map[string]bool
}

var (
	FileList = make([]*FileItem, 0)
	host     = "0.0.0.0"
	port     = "8080"
)

func init() {
	// TODO: 检测编码器的活跃时间，如果超过一定时间没有活跃，则删除
	// 相应地修改服务器端的代码
}

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

	_Conns := make(map[string][]net.Conn)
	_AddCodedPieceChan := make(map[string]chan *coder.CodedPiece)
	_isReceiveStarted := make(map[string]bool)
	for _, hash := range ncfile.Info.Hash {
		_Conns[string(hash)] = make([]net.Conn, 0)
		_AddCodedPieceChan[string(hash)] = make(chan *coder.CodedPiece, 100)
		_isReceiveStarted[string(hash)] = false
	}

	fileItem := &FileItem{
		NcFile:            ncfile,
		Path:              path,
		Nodes:             make([]*NodeItem, 0),
		Conns:             _Conns,
		ConnsMutex:        make(map[string]*sync.Mutex),
		AddCodedPieceChan: _AddCodedPieceChan,
		isReceiveStarted:  _isReceiveStarted,
	}

	isDownloadedBools, err := ncfile.IsFileDownloaded(filepath.Dir(path))
	if err != nil {
		return err
	}

	fileItem.IsDownloaded = make(map[string]bool, len(ncfile.Info.Hash))
	fileItem.IsDownloading = make(map[string]bool, len(ncfile.Info.Hash))
	fileItem.ProcessRate = make(map[string]float64, len(ncfile.Info.Hash))
	fileItem.Decoder = make(map[string]decoder.Decoder, len(ncfile.Info.Hash))
	fileItem.Encoder = make(map[string]encoder.Encoder, len(ncfile.Info.Hash))
	fileItem.Recoder = make(map[string]recoder.Recoder, len(ncfile.Info.Hash))
	for i, isDownloadedBool := range isDownloadedBools {
		hashStr := string(ncfile.Info.Hash[i])
		fileItem.ConnsMutex[hashStr] = &sync.Mutex{}
		if !isDownloadedBool {
			fileItem.IsDownloaded[hashStr] = false
			fileItem.IsDownloading[hashStr] = false
			fileItem.ProcessRate[hashStr] = 0
			if i < len(ncfile.Info.Hash)-1 {
				fileItem.Decoder[hashStr] = decoder.NewGaussElimRLNCDecoder(128)
				continue
			}
			pieceCount := uint(math.Ceil(float64(ncfile.Info.Length%(1<<27)) / float64(1<<20)))
			fileItem.Decoder[hashStr] = decoder.NewGaussElimRLNCDecoder(pieceCount)
			break
		} else {
			fileItem.IsDownloaded[hashStr] = true
			fileItem.IsDownloading[hashStr] = false
			fileItem.ProcessRate[hashStr] = 1
			// open file to get file data
			f, err := os.Open(filepath.Join(filepath.Dir(path), ncfile.Info.Name))
			if err != nil {
				return err
			}
			start := i * (1 << 27)
			end := (i + 1) * (1 << 27)
			if end > int(ncfile.Info.Length) {
				end = int(ncfile.Info.Length)
			}
			data := make([]byte, end-start)
			_, err = f.Seek(int64(start), io.SeekStart)
			if err != nil {
				return err
			}
			f.Read(data)
			f.Close()
			enc, err := encoder.NewSparseRLNCEncoderWithPieceSize(data, 1<<20, 0.1)
			if err != nil {
				return err
			}
			fileItem.Encoder[hashStr] = enc
		}
	}

	// add announce and announce-list
	announce := ncfile.Announce
	if announce != "" {
		// add announce
		_HaveClient := make(map[string]bool)
		for _, item := range ncfile.Info.Hash {
			_HaveClient[string(item)] = false
		}
		fileItem.Nodes = append(fileItem.Nodes, &NodeItem{Addr: announce, IsOn: true, HaveClient: _HaveClient})
	}

	announceList := ncfile.AnnounceList
	if announceList != nil || len(announceList) != 0 {
		// add announce-list
		for _, item := range announceList {
			// if item is already in the list, do nothing
			isExist := false
			for _, node := range fileItem.Nodes {
				if node.Addr == item {
					isExist = true
					break
				}
			}
			if isExist {
				continue
			}
			_HaveClient := make(map[string]bool)
			for _, item := range ncfile.Info.Hash {
				_HaveClient[string(item)] = false
			}
			fileItem.Nodes = append(fileItem.Nodes, &NodeItem{Addr: item, IsOn: true, HaveClient: _HaveClient})
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

func IsBlockExist(hash []byte) bool {
	for _, item := range FileList {
		for _, h := range item.NcFile.Info.Hash {
			if tools.CompareHash(hash, h) {
				return true
			}
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
		for _, h := range item.NcFile.Info.Hash {
			if tools.CompareHash(hash, h) {
				if len(item.Nodes) > 10 {
					// return 10 nodes randomly
					rand.Seed(time.Now().UnixNano())
					rand.Shuffle(len(item.Nodes), func(i, j int) {
						item.Nodes[i], item.Nodes[j] = item.Nodes[j], item.Nodes[i]
					})
					return item.Nodes[:10]
				}
				return item.Nodes
			}
		}
	}
	return nil
}

func GetCodedPiece(hash []byte) *coder.CodedPiece {
	for _, item := range FileList {
		for _, h := range item.NcFile.Info.Hash {
			if tools.CompareHash(hash, h) {
				if item.Encoder[string(hash)] != nil {
					return item.Encoder[string(hash)].CodedPiece()
				}
				if item.Recoder[string(hash)] != nil {
					codedPiece, err := item.Recoder[string(hash)].CodedPiece()
					if err != nil {
						return nil
					}
					return codedPiece
				}
			}
		}
	}
	return nil
}

func (f *FileItem) AddCodedPiece(hash []byte, codedPiece *coder.CodedPiece) {
	hashStr := string(hash)
	if f.IsDownloaded[hashStr] {
		log.Println("already downloaded")
		return
	}
	if f.Decoder[hashStr] == nil {
		return
	}
	f.Decoder[hashStr].AddPiece(codedPiece)
	if f.Recoder[hashStr] == nil {
		ps := make([]*coder.CodedPiece, 1)
		ps[0] = codedPiece
		f.Recoder[hashStr] = recoder.NewFullRLNCRecoder(ps)
	}
	f.Recoder[hashStr].AddCodedPiece(codedPiece)

	f.ProcessRate[hashStr] = f.Decoder[hashStr].ProcessRate()

	if f.Decoder[hashStr].IsDecoded() {
		filePath := f.Path[:len(f.Path)-len(filepath.Ext(f.Path))]
		pieces, err := f.Decoder[hashStr].GetPieces()
		if err != nil {
			return
		}

		index := 0
		for _, h := range f.NcFile.Info.Hash {
			if tools.CompareHash(h, hash) {
				break
			}
			index++
		}
		start := index * (1 << 27)
		// if file is not exist, create it
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			file, err := os.Create(filePath)
			if err != nil {
				return
			}
			file.Close()
		}
		// if file is occupyed by other process, wait
		for {
			file, err := os.OpenFile(filePath, os.O_WRONLY, 0666)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			_, err = file.Seek(int64(start), 0)
			if err != nil {
				return
			}

			if index == len(f.NcFile.Info.Hash)-1 {
				currentLength := 0
				for _, piece := range pieces {
					if len(piece)+currentLength > int(f.NcFile.Info.Length%(1<<27)) {
						_, err = file.Write(piece[:int(f.NcFile.Info.Length%(1<<27))-currentLength])
						if err != nil {
							return
						}
						continue
					}
					_, err = file.Write(piece)
					if err != nil {
						return
					}
					currentLength += len(piece)
				}
			} else {
				for _, piece := range pieces {
					_, err = file.Write(piece)
					if err != nil {
						return
					}
				}
			}
			file.Close()
			break
		}

		f.IsDownloaded[hashStr] = true
		f.IsDownloading[hashStr] = false
		f.ProcessRate[hashStr] = 1
	}
}

func (f *FileItem) StartReceiveCodedPiece() {
	for _, h := range f.NcFile.Info.Hash {
		f.StartReceive(h)
	}
}
func (f *FileItem) StartReceive(hash []byte) {
	hashStr := string(hash)
	if f.isReceiveStarted[hashStr] {
		return
	}
	f.isReceiveStarted[hashStr] = true
	// TODO: 考虑多余 goroutine 和 channel 的问题
	go func() {
		for {
			codedPiece := <-f.AddCodedPieceChan[hashStr]
			f.AddCodedPiece(hash, codedPiece)
			if f.IsDownloaded[hashStr] {
				log.Println("Block (hash:" + hex.EncodeToString(hash) + ") of (" + f.Path + ") is downloaded")
				// stop all connections
				for _, conn := range f.Conns[hashStr] {
					if conn == nil {
						continue
					}
					conn.Close()
				}
				// clear f.Conns
				f.Conns[hashStr] = []net.Conn{}
				for _, node := range f.Nodes {
					node.HaveClient[hashStr] = false
				}

				// remove decoder and recoder, add encoder
				pieces, _ := f.Decoder[hashStr].GetPieces()
				data := make([]byte, 0, f.NcFile.Info.Length)
				currentLength := 0
				for _, piece := range pieces {
					if len(piece)+currentLength > int(f.NcFile.Info.Length) {
						data = append(data, piece[:int(f.NcFile.Info.Length)-currentLength]...)
						return
					}
					data = append(data, piece...)
				}
				enc, _ := encoder.NewFullRLNCEncoderWithPieceSize(data, 1<<20)
				f.Decoder[hashStr] = nil
				f.Recoder[hashStr] = nil
				f.Encoder[hashStr] = enc
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
		_HaveClient := make(map[string]bool)
		for _, h := range item.NcFile.Info.Hash {
			_HaveClient[string(h)] = false
		}
		item.Nodes = append(item.Nodes, &NodeItem{
			Addr:       address,
			IsOn:       false,
			HaveClient: _HaveClient,
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
