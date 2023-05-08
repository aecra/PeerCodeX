package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/aecra/PeerCodeX/dc"
)

type Server struct {
	host   string
	port   string
	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewServer() *Server {
	// create a new server
	log.Println("NewServer")
	return &Server{host: "127.0.0.1", port: "8080", cancel: nil, mu: sync.Mutex{}}
}

func (s *Server) SetHost(host string) {
	s.host = host
}

func (s *Server) SetPort(port string) {
	s.port = port
}

func handleConnection(ctx context.Context, conn net.Conn, server *Server) {
	// handle a connection
	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	reserved, hash, err := handShake(conn, server)
	if err != nil {
		log.Println(err)
		return
	}
	if reserved[0] == 0x01 {
		// This is a heartbeat
		return
	}

	if reserved[1] == 0x01 {
		// send byte 0x01 to client
		_, err := conn.Write([]byte{0x01})
		if err != nil {
			log.Println(err)
		}
		// This is a request for neighbours
		neighbourItems := dc.GetNeighbours(hash)
		// join neighbours' addr by ','
		neighbours := make([]string, len(neighbourItems))
		for i, item := range neighbourItems {
			neighbours[i] = item.Addr
		}
		neighbourStr := strings.Join(neighbours, ",")
		// send neighbours to client
		pstrlenbuf := make([]byte, 4)
		pstrbuf := []byte(neighbourStr)
		binary.BigEndian.PutUint32(pstrlenbuf, uint32(len(pstrbuf)))
		n, err := conn.Write(pstrlenbuf)
		if err != nil || n != 4 {
			log.Println(err)
		}
		n, err = conn.Write(pstrbuf)
		if err != nil || n != len(pstrbuf) {
			log.Println(err)
		}
		return
	}

	// send codedPieces to client
	// send byte 0x02 to client
	for {
		_, err = conn.Write([]byte{0x02})
		if err != nil {
			log.Println(err)
		}

		codedPiece := dc.GetCodedPiece(hash)
		if codedPiece == nil {
			break
		}
		// data format: [vector length][vector][piece length][piece]
		// send codedPiece to client
		lenbuf := make([]byte, 8)
		binary.BigEndian.PutUint64(lenbuf, uint64(len(codedPiece.Vector)))
		n, err := conn.Write(lenbuf)
		if err != nil || n != 8 {
			log.Println(err)
			break
		}
		n, err = conn.Write(codedPiece.Vector)
		if err != nil || n != len(codedPiece.Vector) {
			log.Println(err)
			break
		}
		binary.BigEndian.PutUint64(lenbuf, uint64(len(codedPiece.Piece)))
		n, err = conn.Write(lenbuf)
		if err != nil || n != 8 {
			log.Println(err)
			break
		}
		n, err = conn.Write(codedPiece.Piece)
		if err != nil || n != len(codedPiece.Piece) {
			log.Println(err)
			break
		}
	}
}

func handShake(conn net.Conn, server *Server) (reserved []byte, hash []byte, err error) {
	rbuf := make([]byte, 45)
	n, err := io.ReadFull(conn, rbuf)
	if err != nil || n != 45 {
		return reserved, nil, err
	}
	// pstrlen
	if rbuf[0] != 0x0e {
		return reserved, nil, fmt.Errorf("pstrlen is not 14")
	}
	// protocol name
	if string(rbuf[1:15]) != "Network Coding" {
		return reserved, nil, fmt.Errorf("protocolName is not Network Coding")
	}
	clientIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
	serverPort := binary.BigEndian.Uint16(rbuf[43:45])
	addr := clientIP + ":" + strconv.Itoa(int(serverPort))
	dc.AddNode(addr)
	exist := dc.IsGenerationExist(rbuf[23:43])

	// response
	sbuf := make([]byte, 45)
	sbuf[0] = 0x0e
	// protocolName
	copy(sbuf[1:15], []byte("Network Coding"))
	// reserved
	copy(sbuf[15:23], rbuf[15:23])
	// infohash
	if exist {
		copy(sbuf[23:43], rbuf[23:43])
	} else {
		copy(sbuf[23:43], make([]byte, 20))
	}
	// serverport
	myUint64, err := strconv.ParseUint(server.port, 10, 16)
	if err != nil {
		return reserved, hash, err
	}
	binary.BigEndian.PutUint16(rbuf[43:45], uint16(myUint64))
	// send response
	n, err = conn.Write(sbuf)
	if err != nil || n != 45 {
		return reserved, hash, err
	}
	reserved = rbuf[15:23]
	hash = rbuf[23:43]
	return reserved, hash, nil
}

func (s *Server) Start(panicOccurred chan error) {
	defer func() {
		if err := recover(); err != nil {
			panicOccurred <- fmt.Errorf("ERROR: %v", err)
			s.Stop()
		}
	}()

	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	// start a tcp server
	listener, err := net.Listen("tcp", s.host+":"+s.port)
	if err != nil {
		panic(err)
	}
	defer func() {
		listener.Close()
		log.Println("Server stopped")
		s.Stop()
	}()

	// print server address
	addr := listener.Addr()
	log.Println("Server started at " + addr.String())

	go func() {
		<-ctx.Done()
		// stop the for loop
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if s.cancel == nil {
			// server is stopped
			return
		}
		if err != nil {
			panic(err)
		}
		go handleConnection(ctx, conn, s)
	}
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel == nil {
		return
	}
	s.cancel()
	s.cancel = nil
}

func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel == nil {
		return false
	}
	return true
}
