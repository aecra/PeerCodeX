package tools

import (
	"crypto/sha1"
	"io"
	"os"
	"path/filepath"
)

func GetHashofFile(path string) (hash []byte, err error) {
	// get hash of a file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	hashCalculator := sha1.New()
	for {
		buf := make([]byte, 1<<20)
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
	}
	return hashCalculator.Sum(hash), nil
}

func GetHashofDir(path string) (hash []byte, err error) {
	hashCalculator := sha1.New()
	err = filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// read file to get SHA1
		file, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		defer file.Close()

		for {
			buf := make([]byte, 1<<20)
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if n == 0 {
				break
			}

			if _, err := hashCalculator.Write(buf[:n]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hashCalculator.Sum(hash), nil
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