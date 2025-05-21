package squeeze

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	kgzip "github.com/klauspost/compress/gzip"
)

// Mode selects the optimizing strategy.
type Mode int

const (
	ModeWindow Mode = iota
	ModeBrute
)

const windowSize = 32 * 1024 // 32KB

// File represents a tar entry with data.
type File struct {
	Header *tar.Header
	Data   []byte
	Sum    [32]byte
}

// Stats captures before/after statistics.
type Stats struct {
	BeforeKB    int
	BeforeRatio float64
	AfterKB     int
	AfterRatio  float64
}

// Process takes an input tar.gz and writes an optimized tar.gz to output.
func Process(inPath, outPath string, mode Mode) (*Stats, error) {
	files, before, err := readArchive(inPath)
	if err != nil {
		return nil, err
	}

	var order []int
	switch mode {
	case ModeWindow:
		order = orderWindow(files)
	case ModeBrute:
		order = orderBrute(files)
	default:
		return nil, errors.New("unknown mode")
	}

	if err := writeArchive(outPath, files, order); err != nil {
		return nil, err
	}

	afterInfo, err := os.Stat(outPath)
	if err != nil {
		return nil, err
	}

	afterKB := int(afterInfo.Size() / 1024)
	ratioBefore := float64(before) / float64(totalSize(files)) * 100
	ratioAfter := float64(afterInfo.Size()) / float64(totalSize(files)) * 100

	return &Stats{BeforeKB: int(before / 1024), BeforeRatio: ratioBefore, AfterKB: afterKB, AfterRatio: ratioAfter}, nil
}

func readArchive(path string) ([]*File, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, _ := f.Stat()

	gz, err := kgzip.NewReader(f)
	if err != nil {
		return nil, 0, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var files []*File
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		data := make([]byte, hdr.Size)
		if _, err := io.ReadFull(tr, data); err != nil {
			return nil, 0, err
		}
		sum := sha256.Sum256(appendHeaderData(hdr, data))
		files = append(files, &File{Header: hdr, Data: data, Sum: sum})
	}

	return files, info.Size(), nil
}

func writeArchive(path string, files []*File, order []int) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	gz, err := kgzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	for _, idx := range order {
		f := files[idx]
		if err := tw.WriteHeader(f.Header); err != nil {
			return err
		}
		if _, err := tw.Write(f.Data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	// validate checksums
	if err := validate(path, files, order); err != nil {
		return err
	}

	return nil
}

func validate(path string, files []*File, order []int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := kgzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	i := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		data := make([]byte, hdr.Size)
		if _, err := io.ReadFull(tr, data); err != nil {
			return err
		}
		sum := sha256.Sum256(appendHeaderData(hdr, data))
		if sum != files[order[i]].Sum {
			return fmt.Errorf("checksum mismatch for %s", hdr.Name)
		}
		i++
	}
	return nil
}

func appendHeaderData(h *tar.Header, d []byte) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	_ = tw.WriteHeader(h)
	tw.Write(d)
	tw.Close()
	return buf.Bytes()
}

func totalSize(files []*File) int64 {
	var n int64
	for _, f := range files {
		n += int64(len(f.Data))
	}
	return n
}

// orderWindow implements the compression window optimizing mode.
func orderWindow(files []*File) []int {
	n := len(files)
	remaining := make(map[int]bool)
	for i := 0; i < n; i++ {
		remaining[i] = true
	}

	var order []int
	// pick file with best compression on last window/2 bytes
	best := -1
	bestScore := 1<<31 - 1
	for i, f := range files {
		start := 0
		if len(f.Data) > windowSize/2 {
			start = len(f.Data) - windowSize/2
		}
		score := gzipSize(f.Data[start:])
		if score < bestScore {
			best = i
			bestScore = score
		}
	}
	order = append(order, best)
	delete(remaining, best)

	for len(remaining) > 0 {
		prev := files[order[len(order)-1]]
		best = -1
		bestScore = 1<<31 - 1
		for i := range remaining {
			startPrev := 0
			if len(prev.Data) > windowSize/2 {
				startPrev = len(prev.Data) - windowSize/2
			}
			endNext := windowSize / 2
			if endNext > len(files[i].Data) {
				endNext = len(files[i].Data)
			}
			combined := append(prev.Data[startPrev:], files[i].Data[:endNext]...)
			score := gzipSize(combined)
			if score < bestScore {
				best = i
				bestScore = score
			}
		}
		order = append(order, best)
		delete(remaining, best)
	}

	return order
}

func gzipSize(b []byte) int {
	var buf bytes.Buffer
	gz, _ := kgzip.NewWriterLevel(&buf, gzip.BestCompression)
	gz.Write(b)
	gz.Close()
	return buf.Len()
}

// orderBrute tries all permutations and picks best overall compression.
func orderBrute(files []*File) []int {
	idxs := make([]int, len(files))
	for i := range idxs {
		idxs[i] = i
	}
	best := make([]int, len(files))
	bestScore := 1<<31 - 1

	permute(idxs, func(p []int) {
		var buf bytes.Buffer
		gz, _ := kgzip.NewWriterLevel(&buf, gzip.BestCompression)
		tw := tar.NewWriter(gz)
		for _, idx := range p {
			f := files[idx]
			tw.WriteHeader(f.Header)
			tw.Write(f.Data)
		}
		tw.Close()
		gz.Close()
		if buf.Len() < bestScore {
			bestScore = buf.Len()
			copy(best, p)
		}
	})

	return best
}

func permute(a []int, f func([]int)) {
	sort.Ints(a)
	for {
		f(a)
		if !nextPermutation(a) {
			break
		}
	}
}

func nextPermutation(x []int) bool {
	n := len(x)
	i := n - 2
	for i >= 0 && x[i] >= x[i+1] {
		i--
	}
	if i < 0 {
		return false
	}
	j := n - 1
	for x[j] <= x[i] {
		j--
	}
	x[i], x[j] = x[j], x[i]
	for k, l := i+1, n-1; k < l; k, l = k+1, l-1 {
		x[k], x[l] = x[l], x[k]
	}
	return true
}
