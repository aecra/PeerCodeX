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
	Addr       string
	Hash       []byte
	Generation *dc.Generation
}

func NewClient(addr string, hash []byte, generation *dc.Generation) *Client {
	return &Client{Addr: addr, Hash: hash, Generation: generation}
}

var StartClientChan = make(chan *Client)

func init() {
	// check server status
	go func() {
		time.Sleep(51 * time.Second)
		CkeckAllServerStatus()
	}()

	// check dc.FileList.Nodes to start new client
	go func() {
		for {
			time.Sleep(59 * time.Second)
			for _, file := range dc.FileList {
				for _, generation := range file.Generations {
					RequestForGeneration(generation)
				}
			}
		}
	}()

	// search for enough neighbours
	go func() {
		for {
			time.Sleep(37 * time.Second)
			for _, file := range dc.FileList {
				for _, generation := range file.Generations {
					// delete nodes which is not on
					generation.NodesMutex.Lock()
					oldNeighbours := generation.Nodes
					generation.Nodes = make([]*dc.Node, 0)
					for _, node := range oldNeighbours {
						if node.IsOn == false && node.HaveClient == false {
							continue
						}
						generation.Nodes = append(generation.Nodes, node)
					}

					// get new neighbours
					newNeighbours := make([]string, 0)
					if len(generation.Nodes) < 10 {
						for _, node := range generation.Nodes {
							if node.IsOn == true {
								c := NewClient(node.Addr, generation.Hash, generation)
								neighbours := c.GetNeighbours()
								for _, neighbour := range neighbours {
									if !isSelf(neighbour) {
										newNeighbours = append(newNeighbours, neighbour)
									}
								}
							}
							if len(generation.Nodes)+len(newNeighbours) >= 10 {
								break
							}
						}
					}
					generation.Nodes = append(generation.Nodes, oldNeighbours...)
					generation.NodesMutex.Unlock()
				}
			}
		}
	}()
}

func isSelf(addr string) bool {
	// split ip/host and port
	var host, port string
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			host = addr[:i]
			port = addr[i+1:]
			break
		}
	}
	if host == "" || port == "" {
		return true
	}
	hosts := getLocalHost()
	for _, h := range hosts {
		if h == host {
			if port == dc.GetPort() {
				return true
			}
		}
	}
	return false
}

func getLocalHost() []string {
	inters, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	var hosts []string
	for _, inter := range inters {
		addrs, err := inter.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addrs {
			hosts = append(hosts, addr.String())
		}
	}

	for i, host := range hosts {
		index := 0
		for j, c := range host {
			if c == '/' {
				index = j
			}
		}
		hosts[i] = host[:index]
	}
	return hosts
}

func RequestForFile(file *dc.File) {
	for _, generation := range file.Generations {
		go RequestForGeneration(generation)
	}
}

func RequestForGeneration(generation *dc.Generation) {
	log.Println("RequestForGeneration: ", hex.EncodeToString(generation.Hash))
	generation.StartReceiving()
	for _, node := range generation.Nodes {
		if node.IsOn == true && node.HaveClient == false {
			// start a new client
			c := NewClient(node.Addr, generation.Hash, generation)
			connChan := make(chan net.Conn)
			go c.Start(connChan)
			conn := <-connChan
			node.HaveClient = true
			generation.AddConn(conn)
		}
	}
}

func CkeckAllServerStatus() {
	nodes := dc.GetNodeStatusList()
	for _, node := range nodes {
		c := NewClient(node.Addr, make([]byte, 20), nil)
		status := c.IsServerAlive()
		dc.UpdateNodeStatus(node.Addr, status)
	}
}

func CkeckServerStatus(addr string) {
	c := NewClient(addr, make([]byte, 20), nil)
	status := c.IsServerAlive()
	dc.UpdateNodeStatus(addr, status)
	return
}

func handleShake(client *Client, conn net.Conn, infohash []byte, reserved []byte) ([]byte, uint16, error) {
	// handshake
	pstrlen := []byte{0x0e}
	pstr := []byte("Network Coding")
	serverport := []byte{0x00, 0x00}
	port, _ := strconv.Atoi(dc.GetPort())
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
		if c.Generation.AddCodedPieceChan == nil {
			return
		}
		c.Generation.AddCodedPieceChan <- &codedPiece
	}
}
