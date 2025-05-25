package testutils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"math/rand"
	"time"

	kgzip "github.com/klauspost/compress/gzip"
)

const (
	// WindowSize is the default compression window size for gzip
	WindowSize = 32 * 1024
)

// GenerateTestFiles generates a set of test files as specified in the requirements
func GenerateTestFiles() (map[string][]byte, error) {
	files := make(map[string][]byte)
	
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
	files["file1.txt"] = append(sameFirstHalf, uniqueSection1...)

	// Assemble file 2: same first half + unique middle
	files["file2.txt"] = append(sameFirstHalf, uniqueSection2...)

	// Assemble file 3: unique middle + same last half
	files["file3.txt"] = append(uniqueSection3, sameLastHalf...)

	// Assemble file 4: unique middle + same last half
	files["file4.txt"] = append(uniqueSection4, sameLastHalf...)

	// Create 2 random noise files
	// Use deterministic seed for reproducible test data
	randomSource := rand.New(rand.NewSource(42))
	
	randomData1 := make([]byte, WindowSize)
	if _, err := randomSource.Read(randomData1); err != nil {
		return nil, fmt.Errorf("failed to generate random data: %w", err)
	}
	files["random1.dat"] = randomData1

	randomData2 := make([]byte, WindowSize)
	if _, err := randomSource.Read(randomData2); err != nil {
		return nil, fmt.Errorf("failed to generate random data: %w", err)
	}
	files["random2.dat"] = randomData2

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
func CreateTarGz(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer

	// Create gzip writer with maximum compression
	gzw, err := kgzip.NewWriterLevel(&buf, kgzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	// Create tar writer
	tw := tar.NewWriter(gzw)

	// Add files to the tar archive
	for name, content := range files {
		header := &tar.Header{
			Name:    name,
			Mode:    0644,
			Size:    int64(len(content)),
			ModTime: time.Now(),
		}

		if err := tw.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write tar header: %w", err)
		}

		if _, err := tw.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write file content: %w", err)
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