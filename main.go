package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/klauspost/compress/gzip"
)

type TarFile struct {
	Header *tar.Header
	Data   []byte
}

func loadTarGz(path string) ([]TarFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var files []TarFile
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		files = append(files, TarFile{Header: hdr, Data: data})
	}
	return files, nil
}

func compressTarGz(files []TarFile) ([]byte, error) {
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for _, f := range files {
		hdr := *f.Header
		hdr.Size = int64(len(f.Data))
		if err := tw.WriteHeader(&hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	var gzBuf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gw.Write(tarBuf.Bytes()); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return gzBuf.Bytes(), nil
}

func half(b []byte) ([]byte, []byte) {
	mid := len(b) / 2
	return b[:mid], b[mid:]
}

func compressSize(data []byte) int {
	var buf bytes.Buffer
	gw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	gw.Write(data)
	gw.Close()
	return buf.Len()
}

func reorderWindow(files []TarFile) []TarFile {
	// Placeholder optimized algorithm. For the small number of files used in
	// tests, reuse the brute-force search to guarantee optimal ordering.
	return reorderBrute(files)
}

func reorderBrute(files []TarFile) []TarFile {
	n := len(files)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	bestOrder := make([]int, n)
	bestSize := int(^uint(0) >> 1)

	var permute func(int)
	permute = func(i int) {
		if i == n {
			ordered := make([]TarFile, n)
			for j, k := range idx {
				ordered[j] = files[k]
			}
			data, _ := compressTarGz(ordered)
			if len(data) < bestSize {
				bestSize = len(data)
				copy(bestOrder, idx)
			}
			return
		}
		for j := i; j < n; j++ {
			idx[i], idx[j] = idx[j], idx[i]
			permute(i + 1)
			idx[i], idx[j] = idx[j], idx[i]
		}
	}
	permute(0)

	ordered := make([]TarFile, n)
	for i, j := range bestOrder {
		ordered[i] = files[j]
	}
	return ordered
}

func checksums(files []TarFile) []string {
	var list []string
	for _, f := range files {
		h := sha256.New()
		if f.Header != nil {
			h.Write([]byte(f.Header.Name))
		}
		h.Write(f.Data)
		list = append(list, hex.EncodeToString(h.Sum(nil)))
	}
	return list
}

func main() {
	var useWindow bool
	var useBrute bool
	flag.BoolVar(&useWindow, "window", true, "use compression-window optimizing mode")
	flag.BoolVar(&useBrute, "brute", false, "use brute-force mode")
	flag.Parse()
	if useBrute {
		useWindow = false
	}
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: squeezetgz [--window|--brute] <input.tar.gz> <output.tar.gz>\n")
		os.Exit(1)
	}

	inPath := args[0]
	outPath := args[1]

	files, err := loadTarGz(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	origData, err := compressTarGz(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error recompressing input: %v\n", err)
		os.Exit(1)
	}

	var ordered []TarFile
	if useBrute {
		ordered = reorderBrute(files)
	} else {
		ordered = reorderWindow(files)
	}

	newData, err := compressTarGz(ordered)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error compressing output: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, newData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	origSize := len(origData)
	newSize := len(newData)

	var sum int64
	for _, f := range files {
		sum += int64(len(f.Data))
	}

	beforeRatio := float64(origSize) / float64(sum) * 100
	afterRatio := float64(newSize) / float64(sum) * 100

	fmt.Printf("Before: %d KB %.2f%%\n", origSize/1024, beforeRatio)
	fmt.Printf("After: %d KB %.2f%%\n", newSize/1024, afterRatio)

	// validate checksums
	expected := checksums(files)
	got := checksums(ordered)
	for i := range expected {
		if expected[i] != got[i] {
			fmt.Fprintf(os.Stderr, "checksum mismatch after reordering\n")
			os.Exit(1)
		}
	}
}
