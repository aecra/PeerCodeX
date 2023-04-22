package client

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/dc"
)

func CheckServer(addr string) bool {
	return true
}

// client type is used to connect to a server
type Client struct {
	Addr              string
	Hash              []byte
	LocalServerPort   string
	Conn              net.Conn
	AddCodedPieceChan chan *coder.CodedPiece
}

func NewClient(addr string, hash []byte, localServerPort string, addCodedPieceChan chan *coder.CodedPiece) *Client {
	return &Client{Addr: addr, Hash: hash, LocalServerPort: localServerPort, AddCodedPieceChan: addCodedPieceChan}
}

var StartClientChan = make(chan *Client)

func init() {
	// check server status every 60 seconds
	go func() {
		time.Sleep(60 * time.Second)
		CkeckAllServerStatus()
	}()

	// check dc.FileList.Nodes to start new client every 60 seconds
	go func() {
		time.Sleep(60 * time.Second)
		for _, file := range dc.FileList {
			for _, hash := range file.NcFile.Info.Hash {
				hashStr := string(hash)
				if file.IsDownloaded[hashStr] {
					continue
				}
				if file.AddCodedPieceChan[hashStr] == nil {
					file.AddCodedPieceChan[hashStr] = make(chan *coder.CodedPiece, 100)
				}
				for _, node := range file.Nodes {
					if node.IsOn == true && node.HaveClient[hashStr] == false {
						// start a new client
						c := NewClient(node.Addr, hash, dc.GetPort(), file.AddCodedPieceChan[hashStr])
						connChan := make(chan net.Conn)
						go c.Start(connChan)
						if file.Conns[hashStr] == nil {
							file.Conns[hashStr] = make([]net.Conn, 0)
						}
						c.Conn = <-connChan
						file.ConnsMutex[hashStr].Lock()
						file.Conns[hashStr] = append(file.Conns[hashStr], c.Conn)
						file.ConnsMutex[hashStr].Unlock()
						node.HaveClient[hashStr] = true
					}
				}
			}
		}
	}()

	// TODO：检测是否有足够的邻居节点，如果没有则请求
}

func CkeckAllServerStatus() {
	nodeItems := dc.GetNodeStatusList()
	for _, node := range nodeItems {
		c := NewClient(node.Addr, make([]byte, 20), dc.GetPort(), nil)
		status := c.IsServerAlive()
		dc.UpdateNodeStatus(node.Addr, status)
	}
}

func CkeckServerStatus(addr string) {
	c := NewClient(addr, make([]byte, 20), dc.GetPort(), nil)
	status := c.IsServerAlive()
	dc.UpdateNodeStatus(addr, status)
	return
}

func handleShake(client *Client, conn net.Conn, infohash []byte, reserved []byte) ([]byte, uint16, error) {
	// handshake
	pstrlen := []byte{0x0e}
	pstr := []byte("Network Coding")
	serverport := []byte{0x00, 0x00}
	port, _ := strconv.Atoi(client.LocalServerPort)
	binary.BigEndian.PutUint16(serverport, uint16(port))
	// combine all
	sbuf := append(pstrlen, pstr...)
	sbuf = append(sbuf, reserved...)
	sbuf = append(sbuf, infohash...)
	sbuf = append(sbuf, serverport...)
	// send
	_, err := conn.Write(sbuf)
	if err != nil {
		return nil, 0, errors.New("send handshake failed")
	}
	// read response
	rbuf := make([]byte, 45)
	n, err := io.ReadFull(conn, rbuf)
	if err != nil || n != 45 {
		return nil, 0, errors.New("read handshake failed")
	}
	// pstrlen
	if rbuf[0] != 0x0e {
		return nil, 0, errors.New("pstrlen is not 14")
	}
	// protocolName
	if string(rbuf[1:15]) != "Network Coding" {
		return nil, 0, errors.New("protocolName is not Network Coding")
	}
	if string(rbuf[23:43]) != string(infohash) {
		return nil, 0, errors.New("infohash is not equal")
	}
	// serverport
	serverport = rbuf[43:45]
	return rbuf[15:23], binary.BigEndian.Uint16(serverport), nil
}

func (c *Client) IsServerAlive() bool {
	// create a TCP connection
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	reserved := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	reserved1, _, err := handleShake(c, conn, c.Hash, reserved)
	if err != nil || reserved1[0] != 0x01 {
		return false
	}
	return true
}

func (c *Client) GetNeighbours() []string {
	// create a TCP connection
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		return nil
	}
	defer conn.Close()

	reserved := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	reserved1, _, err := handleShake(c, conn, c.Hash, reserved)
	if err != nil || reserved1[1] != 0x01 {
		return nil
	}
	// receive byte 0x01
	typeBuf := make([]byte, 1)
	_, err = io.ReadFull(conn, typeBuf)
	if err != nil || typeBuf[0] != 0x01 {
		return nil
	}
	// read response
	pstrlenbuf := make([]byte, 4)
	_, err = io.ReadFull(conn, pstrlenbuf)
	if err != nil {
		return nil
	}
	// get the length of neighbours str
	length := binary.BigEndian.Uint32(pstrlenbuf)
	rbuf := make([]byte, length)
	l, err := io.ReadFull(conn, rbuf)
	if err != nil || uint32(l) != length {
		return nil
	}

	return strings.Split(string(rbuf), ",")
}

func (c *Client) Start(connChan chan net.Conn) {
	// create a TCP connection
	log.Println("Dialed to ", c.Addr, "for ", hex.EncodeToString(c.Hash))
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		return
	}
	defer conn.Close()
	connChan <- conn

	reserved := []byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}
	reserved1, _, err := handleShake(c, conn, c.Hash, reserved)
	if err != nil || reserved1[2] != 0x01 {
		return
	}

	// receive codedPieces
	for {
		// receive byte 0x02
		typeBuf := make([]byte, 1)
		_, err = io.ReadFull(conn, typeBuf)
		if err != nil || typeBuf[0] != 0x02 {
			log.Println("receive byte 0x02 failed or ", err)
			return
		}

		codedPiece := coder.CodedPiece{}
		// read vector
		lenBuf := make([]byte, 8)
		n, err := io.ReadFull(conn, lenBuf)
		if err != nil || n != 8 {
			log.Println("read vector length length is not 8 or ", err)
			return
		}
		vlen := binary.BigEndian.Uint64(lenBuf)
		vectorBuf := make([]byte, vlen)
		n, err = io.ReadFull(conn, vectorBuf)
		if err != nil || n != int(vlen) {
			log.Println("read vector length is not ", vlen, " or ", err)
			return
		}
		codedPiece.Vector = vectorBuf
		// read piece
		lenBuf = make([]byte, 8)
		n, err = io.ReadFull(conn, lenBuf)
		if err != nil || n != 8 {
			log.Println("read piece length length is not 8 or ", err)
			return
		}
		plen := binary.BigEndian.Uint64(lenBuf)
		pieceBuf := make([]byte, plen)
		n, err = io.ReadFull(conn, pieceBuf)
		if err != nil || n != int(plen) {
			log.Println("read piece length is not ", plen, " or ", err)
			return
		}
		codedPiece.Piece = pieceBuf
		c.AddCodedPieceChan <- &codedPiece
	}
}
