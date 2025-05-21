package main

import (
	"archive/tar"
	"math/rand"
	"testing"
)

const win = 32768

func genFiles() []TarFile {
	letters := []byte{'A', 'B', 'C', 'D'}
	var files []TarFile
	// four files with matching halves
	for i := 0; i < 4; i++ {
		data := make([]byte, win)
		for j := 0; j < win/2; j++ {
			data[j] = letters[(i+1)%4]
		}
		for j := win / 2; j < win; j++ {
			data[j] = letters[i]
		}
		hdr := &tar.Header{Name: string(letters[i]) + ".txt", Mode: 0600, Size: int64(len(data))}
		files = append(files, TarFile{Header: hdr, Data: data})
	}
	// two random files
	rand.Seed(1)
	for i := 0; i < 2; i++ {
		data := make([]byte, win)
		rand.Read(data)
		hdr := &tar.Header{Name: string('E'+byte(i)) + ".txt", Mode: 0600, Size: int64(len(data))}
		files = append(files, TarFile{Header: hdr, Data: data})
	}
	return files
}

func TestOrdersMatch(t *testing.T) {
	files := genFiles()
	a := reorderWindow(files)
	b := reorderBrute(files)
	if len(a) != len(b) {
		t.Fatalf("length mismatch")
	}
	for i := range a {
		if a[i].Header.Name != b[i].Header.Name {
			t.Fatalf("order mismatch at %d: %s != %s", i, a[i].Header.Name, b[i].Header.Name)
		}
	}
}

func TestCompression(t *testing.T) {
	files := genFiles()
	before, err := compressTarGz(files)
	if err != nil {
		t.Fatal(err)
	}
	ordered := reorderWindow(files)
	after, err := compressTarGz(ordered)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) >= len(before) {
		t.Fatalf("expected better compression")
	}
}
