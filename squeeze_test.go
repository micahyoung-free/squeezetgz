package main

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"os"
	"testing"
)

func genTestFiles() []TarFile {
	half := windowSize / 2
	makeSeq := func(b byte) []byte { return bytes.Repeat([]byte{b}, half) }

	seqA1 := makeSeq('A')
	seqA2 := makeSeq('B')
	seqB1 := makeSeq('C')
	seqC2 := makeSeq('D')

	files := []TarFile{
		{Header: &tar.Header{Name: "file1", Mode: 0644, Size: int64(len(seqA1) + len(seqA2))}, Data: append(append([]byte{}, seqA1...), seqA2...)},
		{Header: &tar.Header{Name: "file2", Mode: 0644, Size: int64(len(seqB1) + len(seqA1))}, Data: append(append([]byte{}, seqB1...), seqA1...)},
		{Header: &tar.Header{Name: "file3", Mode: 0644, Size: int64(len(seqA2) + len(seqC2))}, Data: append(append([]byte{}, seqA2...), seqC2...)},
		{Header: &tar.Header{Name: "file4", Mode: 0644, Size: int64(len(seqC2) + len(seqB1))}, Data: append(append([]byte{}, seqC2...), seqB1...)},
	}

	noise1 := make([]byte, windowSize)
	rand.Read(noise1)
	noise2 := make([]byte, windowSize)
	rand.Read(noise2)
	files = append(files, TarFile{Header: &tar.Header{Name: "noise1", Mode: 0644, Size: int64(len(noise1))}, Data: noise1})
	files = append(files, TarFile{Header: &tar.Header{Name: "noise2", Mode: 0644, Size: int64(len(noise2))}, Data: noise2})

	return files
}

func fileNames(fs []TarFile) []string {
	names := make([]string, len(fs))
	for i, f := range fs {
		names[i] = f.Header.Name
	}
	return names
}

func TestReorderAlgorithmsAgree(t *testing.T) {
	files := genTestFiles()
	win := reorderWindow(files)
	brute := reorderBrute(files)

	wn := fileNames(win)
	bn := fileNames(brute)
	if len(wn) != len(bn) {
		t.Fatalf("length mismatch")
	}
	for i := range wn {
		if wn[i] != bn[i] {
			t.Fatalf("order mismatch at %d: %s vs %s", i, wn[i], bn[i])
		}
	}
}

func TestWriteRead(t *testing.T) {
	files := genTestFiles()
	sums := checksumFiles(files)
	tmp, err := os.CreateTemp(t.TempDir(), "out*.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTarGZ(tmp.Name(), files); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	out, err := readTarGZ(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !validateChecksums(out, sums) {
		t.Fatalf("checksums mismatch")
	}
}
