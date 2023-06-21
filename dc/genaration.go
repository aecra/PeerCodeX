package dc

import (
	"context"
	"encoding/hex"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/coder/decoder"
	"github.com/aecra/PeerCodeX/coder/encoder"
	"github.com/aecra/PeerCodeX/coder/recoder"
)

type Generation struct {
	Hash              []byte                 // hash of the file
	File              *File                  // file which this generation belongs to
	Nodes             []*Node                // nodes which have this generation
	NodesMutex        *sync.RWMutex          // mutex of nodes
	isDownloaded      bool                   // whether this generation is downloaded
	isDownloading     bool                   // whether this generation is downloading
	Conns             []net.Conn             // connections of this generation
	connsMutex        *sync.Mutex            // mutex of conns
	Encoder           encoder.Encoder        // encoder of this generation
	encoderActiveTime time.Time              // time when last codedPiece is generated
	Decoder           decoder.Decoder        // decoder of this generation
	Recoder           recoder.Recoder        // recoder of this generation
	AddCodedPieceChan chan *coder.CodedPiece // channel to receive coded piece
	cancelReceiving   context.CancelFunc     // cancel function of receiving
}

func NewGeneration(file *File, hash []byte, announceList []string, isDownloaded bool) *Generation {
	generation := &Generation{
		Hash:       hash,
		File:       file,
		Nodes:      make([]*Node, 0),
		NodesMutex: &sync.RWMutex{},
		Conns:      make([]net.Conn, 0),
		connsMutex: &sync.Mutex{},
	}
	if isDownloaded {
		generation.isDownloaded = true
		generation.isDownloading = false
	} else {
		generation.isDownloaded = false
		generation.isDownloading = false
	}

	for _, item := range announceList {
		generation.Nodes = append(generation.Nodes, &Node{
			Addr:       item,
			IsOn:       true,
			HaveClient: false,
		})
	}

	return generation
}

func (g *Generation) AddCodedPiece(codedPiece *coder.CodedPiece) {
	if g.isDownloaded {
		return
	}
	if g.Decoder == nil {
		g.Decoder = decoder.NewGaussElimRLNCDecoder(g.File.GetPieceCount(g.Hash))
	}
	g.Decoder.AddPiece(codedPiece)

	if g.Recoder == nil {
		ps := make([]*coder.CodedPiece, 1)
		ps[0] = codedPiece
		g.Recoder = recoder.NewFullRLNCRecoder(ps)
	} else {
		g.Recoder.AddCodedPiece(codedPiece)
	}

	if g.Decoder.IsDecoded() {
		log.Println("Generation(" + hex.EncodeToString(g.Hash) + ") is downloaded")
		g.isDownloaded = true
		g.Save()
		g.Decoder = nil
		g.Recoder = nil
	}
}

func (g *Generation) Save() {
	if g.Decoder == nil || g.Decoder.IsDecoded() == false {
		return
	}

	targetFile := g.File.GetTargetFile()
	// if file is not exist, create it
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		file, err := os.Create(targetFile)
		if err != nil {
			return
		}
		file.Close()
	}

	for {
		file, err := os.OpenFile(targetFile, os.O_WRONLY, 0666)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		_, err = file.Seek(int64(g.File.GetSerialNumber(g.Hash)*1<<27), 0)
		if err != nil {
			return
		}

		pieces, err := g.Decoder.GetPieces()
		if err != nil {
			return
		}

		currentLength := 0
		generationLenght := g.File.GetGenerationLength(g.Hash)
		for _, piece := range pieces {
			if currentLength+len(piece) > int(generationLenght) {
				file.Write(piece[:int(generationLenght)-currentLength])
				break
			}
			file.Write(piece)
			currentLength += len(piece)
		}

		file.Close()
		break
	}
}

func (g *Generation) GetCodedPiece() *coder.CodedPiece {
	if g.Recoder != nil {
		codedPiece, err := g.Recoder.CodedPiece()
		if err != nil {
			return nil
		}
		return codedPiece
	}

	g.encoderActiveTime = time.Now()

	if g.Encoder != nil {
		return g.Encoder.CodedPiece()
	}

	// create encoder
	file, err := os.Open(g.File.GetTargetFile())
	if err != nil {
		return nil
	}
	data := make([]byte, g.File.GetGenerationLength(g.Hash))
	_, err = file.ReadAt(data, int64(g.File.GetSerialNumber(g.Hash)*1<<27))
	if err != nil {
		return nil
	}
	file.Close()

	g.Encoder, err = encoder.NewSparseRLNCEncoderWithPieceCount(data, g.File.GetPieceCount(g.Hash), 0.95)
	if err != nil {
		return nil
	}
	return g.Encoder.CodedPiece()
}

func (g *Generation) StartReceiving() {
	if g.isDownloading || g.isDownloaded {
		return
	}

	if g.Decoder != nil {
		g.Decoder = decoder.NewGaussElimRLNCDecoder(g.File.GetPieceCount(g.Hash))
	}
	g.isDownloading = true
	g.isDownloaded = false
	g.AddCodedPieceChan = make(chan *coder.CodedPiece, 10)
	ctx, cancel := context.WithCancel(context.Background())
	g.cancelReceiving = cancel

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case codedPiece := <-g.AddCodedPieceChan:
				g.AddCodedPiece(codedPiece)
				if g.isDownloaded {
					go g.StopReceiving()
				}
			}
		}
	}(ctx)
}

func (g *Generation) StopReceiving() {
	if g.isDownloading == false {
		return
	}
	g.isDownloading = false

	g.connsMutex.Lock()
	for _, conn := range g.Conns {
		if conn != nil {
			conn.Close()
		}
	}
	g.Conns = []net.Conn{}
	g.connsMutex.Unlock()

	g.NodesMutex.RLock()
	for _, node := range g.Nodes {
		node.HaveClient = false
	}
	g.NodesMutex.RUnlock()

	g.cancelReceiving()
	g.AddCodedPieceChan = nil
}

func (g *Generation) AddNode(addr string) {
	g.NodesMutex.Lock()
	defer g.NodesMutex.Unlock()
	for _, node := range g.Nodes {
		if node.Addr == addr {
			return
		}
	}
	g.Nodes = append(g.Nodes, &Node{
		Addr:       addr,
		IsOn:       true,
		HaveClient: false,
	})
}

func (g *Generation) DeleteNode(addr string) {
	g.NodesMutex.Lock()
	defer g.NodesMutex.Unlock()
	oldNeighbours := g.Nodes
	g.Nodes = make([]*Node, 0)
	for _, node := range oldNeighbours {
		if node.Addr != addr {
			g.Nodes = append(g.Nodes, node)
		}
	}
}

func (g *Generation) GetDecodedSize() uint {
	if g.isDownloaded {
		return g.File.GetGenerationLength(g.Hash)
	}
	if g.Decoder == nil {
		return 0
	}
	return uint(g.Decoder.ProcessRate() * float64(g.File.GetGenerationLength(g.Hash)))
}

func (g *Generation) GetProcessRate() float64 {
	if g.isDownloaded {
		return 1
	}
	if g.Decoder == nil {
		return 0
	}
	return g.Decoder.ProcessRate()
}

func (g *Generation) AddConn(conn net.Conn) {
	g.connsMutex.Lock()
	defer g.connsMutex.Unlock()
	g.Conns = append(g.Conns, conn)
}

func (g *Generation) IsDownloading() bool {
	return g.isDownloading
}

func (g *Generation) DropIdleEncoder() {
	if g.Encoder == nil {
		return
	}
	if time.Now().Sub(g.encoderActiveTime) > 10*time.Second {
		g.Encoder = nil
	}
}
