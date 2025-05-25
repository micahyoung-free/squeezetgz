package squeezetgz_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/micahyoung-free/squeezetgz/internal/squeezetgz"
)

func TestSpecialFileTypes(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "squeezetgz-special-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a tar.gz with special file types
	inputPath := filepath.Join(tempDir, "special_input.tar.gz")
	err = createSpecialTarGz(inputPath)
	if err != nil {
		t.Fatalf("Failed to create special tar.gz: %v", err)
	}

	// Optimize the tar.gz
	outputPath := filepath.Join(tempDir, "special_output.tar.gz")
	_, err = squeezetgz.OptimizeTarGz(inputPath, outputPath, squeezetgz.WindowMode)
	if err != nil {
		t.Fatalf("Failed to optimize tar.gz: %v", err)
	}

	// Verify all file types are preserved
	inputFiles, err := listTarGzContents(inputPath)
	if err != nil {
		t.Fatalf("Failed to list input tar.gz contents: %v", err)
	}

	outputFiles, err := listTarGzContents(outputPath)
	if err != nil {
		t.Fatalf("Failed to list output tar.gz contents: %v", err)
	}

	// Check that all file types and entries are preserved
	if len(inputFiles) != len(outputFiles) {
		t.Fatalf("Input had %d entries, output has %d entries", len(inputFiles), len(outputFiles))
	}

	// Verify all file types are present in output
	typesFound := make(map[byte]bool)
	for _, file := range outputFiles {
		typesFound[file.Type] = true
	}

	// We should have at least regular files, symlinks, and empty files
	if !typesFound[tar.TypeReg] {
		t.Errorf("Regular files are missing from output")
	}
	if !typesFound[tar.TypeSymlink] {
		t.Errorf("Symlinks are missing from output")
	}

	// Check if empty files are present
	emptyFileFound := false
	for _, file := range outputFiles {
		if file.Type == tar.TypeReg && file.Size == 0 {
			emptyFileFound = true
			break
		}
	}
	if !emptyFileFound {
		t.Errorf("Empty files are missing from output")
	}
}

// TarFileInfo holds information about a tar entry
type TarFileInfo struct {
	Name     string
	Type     byte
	Size     int64
	LinkName string
}

// createSpecialTarGz creates a tar.gz file with special file types
func createSpecialTarGz(outputPath string) error {
	// Create a buffer to write our archive to
	var buf bytes.Buffer
	
	// Create gzip writer
	gzw := gzip.NewWriter(&buf)
	
	// Create tar writer
	tw := tar.NewWriter(gzw)
	
	// Fixed time for deterministic output
	fixedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	// Add a regular file
	regFileContent := []byte("This is a regular file")
	regHeader := &tar.Header{
		Name:    "regular.txt",
		Mode:    0644,
		Size:    int64(len(regFileContent)),
		ModTime: fixedTime,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(regHeader); err != nil {
		return err
	}
	if _, err := tw.Write(regFileContent); err != nil {
		return err
	}
	
	// Add an empty file
	emptyHeader := &tar.Header{
		Name:    "empty.txt",
		Mode:    0644,
		Size:    0,
		ModTime: fixedTime,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(emptyHeader); err != nil {
		return err
	}
	
	// Add a symlink
	symlinkHeader := &tar.Header{
		Name:     "link.txt",
		Linkname: "regular.txt",
		Mode:     0777,
		ModTime:  fixedTime,
		Typeflag: tar.TypeSymlink,
	}
	if err := tw.WriteHeader(symlinkHeader); err != nil {
		return err
	}
	
	// Close writers
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	
	// Write to file
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// listTarGzContents lists all entries in a tar.gz file
func listTarGzContents(tarGzPath string) ([]TarFileInfo, error) {
	// Read the tar.gz file
	data, err := os.ReadFile(tarGzPath)
	if err != nil {
		return nil, err
	}
	
	// Create a gzip reader
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	
	// Create a tar reader
	tr := tar.NewReader(gzr)
	var files []TarFileInfo
	
	// Read all entries
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		files = append(files, TarFileInfo{
			Name:     header.Name,
			Type:     header.Typeflag,
			Size:     header.Size,
			LinkName: header.Linkname,
		})
	}
	
	return files, nil
}