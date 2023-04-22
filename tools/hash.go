package tools

import (
	"crypto/sha1"
	"io"
	"os"
)

func GetHashsofFile(path string) (hashs [][]byte, err error) {
	// get hash of a file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// calculate hash per 128MB
	hashCalculator := sha1.New()
	for {
		buf := make([]byte, 1<<27)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		if _, err := hashCalculator.Write(buf[:n]); err != nil {
			return nil, err
		}
		hashs = append(hashs, hashCalculator.Sum(nil))
		hashCalculator.Reset()
	}
	return hashs, nil
}

func CompareHash(hash1 []byte, hash2 []byte) bool {
	// compare two hash
	if len(hash1) != len(hash2) {
		return false
	}
	for i := range hash1 {
		if hash1[i] != hash2[i] {
			return false
		}
	}
	return true
}
