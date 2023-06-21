package dc

import (
	"errors"
	"sync"
	"time"

	"github.com/aecra/PeerCodeX/coder"
	"github.com/aecra/PeerCodeX/tools"
)

var (
	FileList      = make([]*File, 0)
	FileListMutex = sync.RWMutex{}
	host          = "0.0.0.0"
	port          = "8080"
)

func init() {
	// check encoder status
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			FileListMutex.RLock()
			for _, file := range FileList {
				file.DropIdleEncoder()
			}
			FileListMutex.RUnlock()
		}
	}()
}

func GetFileByPath(path string) *File {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	for _, item := range FileList {
		if item.Path == path {
			return item
		}
	}
	return nil
}

func DeleteFileByPath(path string) {
	FileListMutex.Lock()
	defer FileListMutex.Unlock()
	for i, item := range FileList {
		if item.Path == path {
			FileList = append(FileList[:i], FileList[i+1:]...)
		}
	}
}

func AddFile(path string) error {
	f := GetFileByPath(path)
	if f != nil {
		return errors.New("file already exists")
	}

	file, err := NewFile(path)
	if err != nil {
		return err
	}

	FileListMutex.Lock()
	FileList = append(FileList, file)
	FileListMutex.Unlock()
	return nil
}

func IsGenerationExist(hash []byte) bool {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
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

func GetNeighbours(hash []byte) []*Node {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	// return atmost 10 neighbours
	neighbours := make([]*Node, 0)
	for _, f := range FileList {
		for _, g := range f.Generations {
			if tools.CompareHash(g.Hash, hash) {
				g.NodesMutex.RLock()
				for _, n := range g.Nodes {
					if len(neighbours) >= 10 {
						return neighbours
					}
					neighbours = append(neighbours, n)
				}
				g.NodesMutex.RUnlock()
			}
		}
	}
	for _, f := range FileList {
		for _, g := range f.Generations {
			if !tools.CompareHash(g.Hash, hash) {
				g.NodesMutex.RLock()
				for _, n := range g.Nodes {
					if len(neighbours) >= 10 {
						return neighbours
					}
					neighbours = append(neighbours, n)
				}
				g.NodesMutex.RUnlock()
			}
		}
	}
	return neighbours
}

func GetCodedPiece(hash []byte) *coder.CodedPiece {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	for _, f := range FileList {
		for _, g := range f.Generations {
			if tools.CompareHash(g.Hash, hash) {
				return g.GetCodedPiece()
			}
		}
	}
	return nil
}

func GetNodeStatusList() []*Node {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	nodes := make([]*Node, 0)
	for _, f := range FileList {
		for _, g := range f.Generations {
			g.NodesMutex.RLock()
			for _, n := range g.Nodes {
				nodes = append(nodes, n)
			}
			g.NodesMutex.RUnlock()
		}
	}
	result := []*Node{}
	temp := map[string]struct{}{}
	for _, item := range nodes {
		if _, ok := temp[item.Addr]; !ok {
			temp[item.Addr] = struct{}{}
			result = append(result, item)
		}
	}
	// if there is a node is on, set it to true
	for i, item := range result {
		for _, node := range nodes {
			if item.Addr == node.Addr && node.IsOn {
				result[i].IsOn = true
			}
		}
	}
	return result
}

func UpdateNodeStatus(address string, status bool) {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	for _, f := range FileList {
		for _, g := range f.Generations {
			g.NodesMutex.RLock()
			for _, n := range g.Nodes {
				if n.Addr == address {
					n.IsOn = status
				}
			}
			g.NodesMutex.RUnlock()
		}
	}
}

func AddNode(addr string) {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	for _, f := range FileList {
		f.AddNode(addr)
	}
}

func DeleteNode(addr string) {
	FileListMutex.RLock()
	defer FileListMutex.RUnlock()
	for _, f := range FileList {
		f.DeleteNode(addr)
	}
}
