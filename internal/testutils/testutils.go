package testutils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"time"

	kgzip "github.com/klauspost/compress/gzip"
)

const (
	// WindowSize is the default compression window size for gzip
	WindowSize = 32 * 1024
)

// FileType represents the type of file to be created
type FileType int

const (
	// RegularFile is a standard file with content
	RegularFile FileType = iota
	// EmptyFile is a file with zero size
	EmptyFile
	// SymlinkFile is a symbolic link to another file
	SymlinkFile
)

// TestFile represents a file with its content and metadata
type TestFile struct {
	Content  []byte
	Type     FileType
	LinkTo   string // Target path for symlinks
}

// GenerateTestFiles generates a set of test files as specified in the requirements
func GenerateTestFiles() (map[string]TestFile, error) {
	files := make(map[string]TestFile)
	
	// Create 4 alphabetical files with matching halves
	// Each pair will have identical first/last half window
	const halfWindow = WindowSize / 2

	// File 1 and 2 will have identical first half windows
	sameFirstHalf := make([]byte, halfWindow)
	for i := range sameFirstHalf {
		sameFirstHalf[i] = 'A' + byte(i%26)
	}

	// File 3 and 4 will have identical last half windows
	sameLastHalf := make([]byte, halfWindow)
	for i := range sameLastHalf {
		sameLastHalf[i] = 'a' + byte(i%26)
	}

	// Create unique content for each file middle section
	uniqueSection1 := generateAlphabeticalBytes(WindowSize, 'B')
	uniqueSection2 := generateAlphabeticalBytes(WindowSize, 'C')
	uniqueSection3 := generateAlphabeticalBytes(WindowSize, 'D')
	uniqueSection4 := generateAlphabeticalBytes(WindowSize, 'E')

	// Assemble file 1: same first half + unique middle
	files["file1.txt"] = TestFile{
		Content: append(sameFirstHalf, uniqueSection1...),
		Type:    RegularFile,
	}

	// Assemble file 2: same first half + unique middle
	files["file2.txt"] = TestFile{
		Content: append(sameFirstHalf, uniqueSection2...),
		Type:    RegularFile,
	}

	// Assemble file 3: unique middle + same last half
	files["file3.txt"] = TestFile{
		Content: append(uniqueSection3, sameLastHalf...),
		Type:    RegularFile,
	}

	// Assemble file 4: unique middle + same last half
	files["file4.txt"] = TestFile{
		Content: append(uniqueSection4, sameLastHalf...),
		Type:    RegularFile,
	}

	// Create 2 random noise files
	// Use deterministic seed for reproducible test data
	randomSource := rand.New(rand.NewSource(42))
	
	randomData1 := make([]byte, WindowSize)
	if _, err := randomSource.Read(randomData1); err != nil {
		return nil, fmt.Errorf("failed to generate random data: %w", err)
	}
	files["random1.dat"] = TestFile{
		Content: randomData1,
		Type:    RegularFile,
	}

	randomData2 := make([]byte, WindowSize)
	if _, err := randomSource.Read(randomData2); err != nil {
		return nil, fmt.Errorf("failed to generate random data: %w", err)
	}
	files["random2.dat"] = TestFile{
		Content: randomData2,
		Type:    RegularFile,
	}

	// Add an empty file
	files["empty.txt"] = TestFile{
		Content: []byte{},
		Type:    EmptyFile,
	}

	// Add a symlink to file1.txt
	files["link-to-file1.txt"] = TestFile{
		Content: []byte{},
		Type:    SymlinkFile,
		LinkTo:  "file1.txt",
	}

	return files, nil
}

// generateAlphabeticalBytes generates alphabetical bytes with the given start character
func generateAlphabeticalBytes(size int, startChar byte) []byte {
	result := make([]byte, size)
	for i := range result {
		// Cycle through alphabet starting from startChar
		result[i] = 'A' + (startChar-'A'+byte(i))%26
		// Ensure result stays within the range of uppercase letters
		// No additional adjustment needed as normalization handles it
	}
	return result
}

// CreateTarGz creates a tar.gz file from the provided files
func CreateTarGz(files map[string]TestFile) ([]byte, error) {
	var buf bytes.Buffer

	// Create gzip writer with maximum compression
	gzw, err := kgzip.NewWriterLevel(&buf, kgzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	// Create tar writer
	tw := tar.NewWriter(gzw)

	// Sort file names for deterministic order
	var fileNames []string
	for name := range files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	// Add files to the tar archive in sorted order
	for _, name := range fileNames {
		file := files[name]
		
		// Use a fixed timestamp for deterministic test results
		fixedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		
		// Create the appropriate header based on file type
		var header *tar.Header
		
		switch file.Type {
		case RegularFile, EmptyFile:
			header = &tar.Header{
				Name:    name,
				Mode:    0644,
				Size:    int64(len(file.Content)),
				ModTime: fixedTime,
				Typeflag: tar.TypeReg,
			}
		case SymlinkFile:
			header = &tar.Header{
				Name:     name,
				Linkname: file.LinkTo,
				Mode:     0777,
				ModTime:  fixedTime,
				Typeflag: tar.TypeSymlink,
			}
		}

		if err := tw.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write tar header: %w", err)
		}

		// Only write content for regular files
		if file.Type == RegularFile || file.Type == EmptyFile {
			if _, err := tw.Write(file.Content); err != nil {
				return nil, fmt.Errorf("failed to write file content: %w", err)
			}
		}
	}

	// Close the writers
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// GetFileOrder extracts the order of files from a tar.gz file
func GetFileOrder(tarGzPath string) ([]string, error) {
	// Read the tar.gz file
	data, err := os.ReadFile(tarGzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tar.gz file: %w", err)
	}

	// Create a gzip reader
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create a tar reader
	tr := tar.NewReader(gzr)
	var fileOrder []string

	// Read all headers to get file order
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Include all file types in the order
		fileOrder = append(fileOrder, header.Name)
	}

	return fileOrder, nil
}