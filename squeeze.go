package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"

	kgzip "github.com/klauspost/compress/gzip"
)

type TarFile struct {
	Header *tar.Header
	Data   []byte
}

type checksum struct {
	hdr  [32]byte
	data [32]byte
}

func readTarGZ(path string) ([]TarFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := kgzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var files []TarFile
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data := make([]byte, hdr.Size)
		if _, err := io.ReadFull(tr, data); err != nil {
			return nil, err
		}
		hcopy := *hdr
		files = append(files, TarFile{Header: &hcopy, Data: data})
	}
	return files, nil
}

func writeTarGZ(path string, files []TarFile) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := kgzip.NewWriterLevel(f, kgzip.BestCompression)
	if err != nil {
		return err
	}
	defer gz.Close()

	tw := tar.NewWriter(gz)
	for _, tf := range files {
		if err := tw.WriteHeader(tf.Header); err != nil {
			return err
		}
		if _, err := tw.Write(tf.Data); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

const windowSize = 32 * 1024

func compressRatio(b []byte) float64 {
	var buf bytes.Buffer
	w, _ := kgzip.NewWriterLevel(&buf, kgzip.BestCompression)
	w.Write(b)
	w.Close()
	if len(b) == 0 {
		return 0
	}
	return float64(buf.Len()) / float64(len(b))
}

func checksumFiles(files []TarFile) []checksum {
	sums := make([]checksum, len(files))
	for i, f := range files {
		hdrBytes := []byte(fmt.Sprintf("%v", f.Header))
		sums[i].hdr = sha256.Sum256(hdrBytes)
		sums[i].data = sha256.Sum256(f.Data)
	}
	return sums
}

func validateChecksums(files []TarFile, sums []checksum) bool {
	if len(files) != len(sums) {
		return false
	}
	for i, f := range files {
		hdrBytes := []byte(fmt.Sprintf("%v", f.Header))
		if sha := sha256.Sum256(hdrBytes); sha != sums[i].hdr {
			return false
		}
		if sha := sha256.Sum256(f.Data); sha != sums[i].data {
			return false
		}
	}
	return true
}

func reorderWindow(files []TarFile) []TarFile {
	if len(files) == 0 {
		return nil
	}
	remaining := make([]TarFile, len(files))
	copy(remaining, files)
	// start with file with best compression on last half
	sort.Slice(remaining, func(i, j int) bool {
		a := compressRatio(lastHalf(remaining[i].Data))
		b := compressRatio(lastHalf(remaining[j].Data))
		return a < b
	})
	ordered := []TarFile{remaining[0]}
	remaining = remaining[1:]

	for len(remaining) > 0 {
		prev := ordered[len(ordered)-1]
		sort.Slice(remaining, func(i, j int) bool {
			a := overlapScore(prev.Data, remaining[i].Data)
			b := overlapScore(prev.Data, remaining[j].Data)
			return a > b
		})
		ordered = append(ordered, remaining[0])
		remaining = remaining[1:]
	}
	return ordered
}

func reorderBrute(files []TarFile) []TarFile {
	best := make([]TarFile, len(files))
	var bestSize int64 = -1
	permute(files, func(p []TarFile) {
		size := estimateSize(p)
		if bestSize == -1 || size < bestSize {
			bestSize = size
			copy(best, p)
		}
	})
	return best
}

func estimateSize(files []TarFile) int64 {
	// compress concatenated data
	pr, pw := io.Pipe()
	gz, _ := kgzip.NewWriterLevel(pw, kgzip.BestCompression)
	go func() {
		tw := tar.NewWriter(gz)
		for _, f := range files {
			tw.WriteHeader(f.Header)
			tw.Write(f.Data)
		}
		tw.Close()
		gz.Close()
		pw.Close()
	}()
	n, _ := io.Copy(io.Discard, pr)
	pr.Close()
	return n
}

func permute(files []TarFile, fn func([]TarFile)) {
	var helper func(int)
	helper = func(i int) {
		if i == len(files) {
			tmp := make([]TarFile, len(files))
			copy(tmp, files)
			fn(tmp)
			return
		}
		for j := i; j < len(files); j++ {
			files[i], files[j] = files[j], files[i]
			helper(i + 1)
			files[i], files[j] = files[j], files[i]
		}
	}
	helper(0)
}

func lastHalf(b []byte) []byte {
	if len(b) <= windowSize/2 {
		return b
	}
	return b[len(b)-windowSize/2:]
}

func firstHalf(b []byte) []byte {
	if len(b) <= windowSize/2 {
		return b
	}
	return b[:windowSize/2]
}

func overlapScore(a, b []byte) int {
	ah := lastHalf(a)
	bh := firstHalf(b)
	max := len(ah)
	if len(bh) < max {
		max = len(bh)
	}
	score := 0
	for i := 0; i < max; i++ {
		if ah[len(ah)-1-i] == bh[len(bh)-1-i] {
			score++
		} else {
			break
		}
	}
	return score
}
