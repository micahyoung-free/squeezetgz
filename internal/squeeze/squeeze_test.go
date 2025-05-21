package squeeze

import (
	"archive/tar"
	"bytes"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	kgzip "github.com/klauspost/compress/gzip"
)

// helper to create tar.gz from given files
func createTarGz(path string, files map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, _ := kgzip.NewWriterLevel(f, kgzip.BestCompression)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for name, data := range files {
		hdr := &tar.Header{Name: name, Mode: 0600, Size: int64(len(data))}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
	tw.Close()
	gz.Close()
	return nil
}

// Generate deterministic alphabetic test files
func genAlphaFiles(window int) map[string][]byte {
	data := make(map[string][]byte)
	// A shares prefix with B and suffix with B's prefix
	half := window / 2
	a := bytes.Repeat([]byte{'a'}, half)
	a = append(a, bytes.Repeat([]byte{'b'}, 20)...)
	a = append(a, bytes.Repeat([]byte{'c'}, half)...)
	b := append(bytes.Repeat([]byte{'c'}, half), bytes.Repeat([]byte{'d'}, 20)...)
	b = append(b, bytes.Repeat([]byte{'a'}, half)...)
	c := append(bytes.Repeat([]byte{'d'}, half), bytes.Repeat([]byte{'e'}, 20)...)
	c = append(c, bytes.Repeat([]byte{'f'}, half)...)
	d := append(bytes.Repeat([]byte{'f'}, half), bytes.Repeat([]byte{'g'}, 20)...)
	d = append(d, bytes.Repeat([]byte{'d'}, half)...)
	data["a.txt"] = a
	data["b.txt"] = b
	data["c.txt"] = c
	data["d.txt"] = d
	return data
}

func genRandomFiles() map[string][]byte {
	rand.Seed(42)
	data := make(map[string][]byte)
	r1 := make([]byte, 100)
	r2 := make([]byte, 100)
	rand.Read(r1)
	rand.Read(r2)
	data["noise1.bin"] = r1
	data["noise2.bin"] = r2
	return data
}

func TestModesProduceSameArrangement(t *testing.T) {
	dir := t.TempDir()
	files := genAlphaFiles(windowSize)
	for k, v := range genRandomFiles() {
		files[k] = v
	}

	input := dir + "/in.tar.gz"
	if err := createTarGz(input, files); err != nil {
		t.Fatal(err)
	}

	outWindow := dir + "/out_window.tar.gz"
	outBrute := dir + "/out_brute.tar.gz"

	statsW, err := Process(input, outWindow, ModeWindow)
	if err != nil {
		t.Fatal(err)
	}

	statsB, err := Process(input, outBrute, ModeBrute)
	if err != nil {
		t.Fatal(err)
	}

	if statsW.AfterKB != statsB.AfterKB {
		t.Fatalf("expected sizes to match: %d != %d", statsW.AfterKB, statsB.AfterKB)
	}

	orderW, _ := orderFromArchive(outWindow)
	orderB, _ := orderFromArchive(outBrute)
	if len(orderW) != len(orderB) {
		t.Fatalf("output files mismatch")
	}
	for i := range orderW {
		if orderW[i] != orderB[i] {
			t.Fatalf("arrangement mismatch at %d", i)
		}
	}
}

func orderFromArchive(path string) ([]string, error) {
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
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, hdr.Name)
		io.Copy(io.Discard, tr)
	}
	return names, nil
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}
