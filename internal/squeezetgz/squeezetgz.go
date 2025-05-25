package squeezetgz

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"sync"

	stdgzip "compress/gzip"
	kgzip "github.com/klauspost/compress/gzip"
)

// OptimizationMode represents the optimization strategy
type OptimizationMode int

const (
	// WindowMode uses the compression-window optimizing approach
	WindowMode OptimizationMode = iota
	// BruteForceMode tries all possible permutations
	BruteForceMode
)

// OptimizationResult contains statistics about the optimization
type OptimizationResult struct {
	BeforeSize  int64
	AfterSize   int64
	BeforeRatio float64
	AfterRatio  float64
}

// TarFile represents a file from the tar archive
type TarFile struct {
	Header      *tar.Header
	Content     []byte
	Checksum    [sha256.Size]byte
	HeaderHash  [sha256.Size]byte
	FirstWindow []byte
	LastWindow  []byte
}

// OptimizeTarGz optimizes a tar.gz file by reordering its contents
func OptimizeTarGz(inputPath, outputPath string, mode OptimizationMode) (*OptimizationResult, error) {
	// Read the input file
	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	// Calculate the original compression ratio
	originalUncompressed, files, err := extractTarGz(inputBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tar.gz: %w", err)
	}

	// Determine the compression window size
	// Note: In gzip, the compression window is typically 32KB
	windowSize := 32 * 1024
	halfWindowSize := windowSize / 2

	// Prepare the first and last windows for each file
	for i := range files {
		// Only prepare windows for regular files
		if files[i].Header.Typeflag == tar.TypeReg {
			if len(files[i].Content) <= halfWindowSize {
				files[i].FirstWindow = files[i].Content
				files[i].LastWindow = files[i].Content
			} else {
				files[i].FirstWindow = files[i].Content[:halfWindowSize]
				files[i].LastWindow = files[i].Content[len(files[i].Content)-halfWindowSize:]
			}
		}
	}

	// Separate regular files from special files
	var regularFiles []*TarFile
	var specialFiles []*TarFile
	
	for _, file := range files {
		if file.Header.Typeflag == tar.TypeReg {
			regularFiles = append(regularFiles, file)
		} else {
			specialFiles = append(specialFiles, file)
		}
	}

	// Reorder only the regular files based on the selected optimization mode
	var orderedRegularFiles []*TarFile
	if mode == BruteForceMode {
		orderedRegularFiles, err = optimizeBruteForce(regularFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize with brute force: %w", err)
		}
	} else {
		orderedRegularFiles, err = optimizeWindow(regularFiles, halfWindowSize)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize with window mode: %w", err)
		}
	}

	// Combine the ordered regular files with the special files
	orderedFiles := append(orderedRegularFiles, specialFiles...)

	// Create a new tar.gz with the optimized order
	optimizedTarGz, err := createTarGz(orderedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to create optimized tar.gz: %w", err)
	}

	// Validate checksums before writing output
	if !validateChecksums(files, orderedFiles) {
		return nil, fmt.Errorf("checksum validation failed, file integrity compromised")
	}

	// Write the output file
	if err := os.WriteFile(outputPath, optimizedTarGz, 0644); err != nil {
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}

	// Calculate compression statistics
	result := &OptimizationResult{
		BeforeSize:  int64(len(inputBytes)),
		AfterSize:   int64(len(optimizedTarGz)),
		BeforeRatio: float64(len(inputBytes)) / float64(originalUncompressed),
		AfterRatio:  float64(len(optimizedTarGz)) / float64(originalUncompressed),
	}

	return result, nil
}

// extractTarGz extracts files from a tar.gz byte array
func extractTarGz(data []byte) (int64, []*TarFile, error) {
	gzr, err := stdgzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var files []*TarFile
	var totalUncompressedSize int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Process based on file type
		var contentBytes []byte
		if header.Typeflag == tar.TypeReg {
			// For regular files, read the content
			var content bytes.Buffer
			if _, err := io.Copy(&content, tr); err != nil {
				return 0, nil, fmt.Errorf("failed to read file content: %w", err)
			}
			contentBytes = content.Bytes()
			totalUncompressedSize += int64(len(contentBytes))
		}
		// For all other types (symlinks, etc.), content remains empty slice

		// Calculate checksums
		contentChecksum := sha256.Sum256(contentBytes)
		
		// Calculate header checksum
		headerBytes := &bytes.Buffer{}
		if err := tar.NewWriter(headerBytes).WriteHeader(header); err != nil {
			return 0, nil, fmt.Errorf("failed to write header for checksum: %w", err)
		}
		headerChecksum := sha256.Sum256(headerBytes.Bytes())

		files = append(files, &TarFile{
			Header:     header,
			Content:    contentBytes,
			Checksum:   contentChecksum,
			HeaderHash: headerChecksum,
		})
	}

	return totalUncompressedSize, files, nil
}

// optimizeWindow implements the graph-based, compression-window optimizing mode
func optimizeWindow(files []*TarFile, halfWindowSize int) ([]*TarFile, error) {
	if len(files) == 0 {
		return files, nil
	}

	// Make a copy of the files slice to avoid modifying the original
	remaining := make([]*TarFile, len(files))
	copy(remaining, files)

	// Start with the file that has the worst compression ratio overall (least compressible)
	var ordered []*TarFile
	bestStartIdx := findBestStartFile(remaining, halfWindowSize)
	ordered = append(ordered, remaining[bestStartIdx])
	remaining = append(remaining[:bestStartIdx], remaining[bestStartIdx+1:]...)

	// Build the chain by finding the best next file
	for len(remaining) > 0 {
		lastFile := ordered[len(ordered)-1]
		bestNextIdx := findBestNextFile(lastFile, remaining, halfWindowSize)
		ordered = append(ordered, remaining[bestNextIdx])
		remaining = append(remaining[:bestNextIdx], remaining[bestNextIdx+1:]...)
	}

	return ordered, nil
}

// optimizeBruteForce implements the brute-force optimization mode
func optimizeBruteForce(files []*TarFile) ([]*TarFile, error) {
	if len(files) == 0 {
		return files, nil
	}

	// For small number of files, try all permutations
	if len(files) > 10 {
		return nil, fmt.Errorf("too many files for brute force optimization (max 10)")
	}

	bestOrder := make([]*TarFile, len(files))
	copy(bestOrder, files)
	bestSize := math.MaxInt64

	// Generate all permutations and find the one with the best compression
	permuteAndCompress(files, 0, &bestOrder, &bestSize)

	return bestOrder, nil
}

// permuteAndCompress generates permutations of files and keeps track of the best compression
func permuteAndCompress(files []*TarFile, index int, bestOrder *[]*TarFile, bestSize *int) {
	if index == len(files) {
		// Calculate compression size for this permutation
		tarGz, err := createTarGz(files)
		if err != nil {
			return
		}

		size := len(tarGz)
		if size < *bestSize {
			*bestSize = size
			copy(*bestOrder, files)
		}
		return
	}

	for i := index; i < len(files); i++ {
		// Swap elements
		files[index], files[i] = files[i], files[index]

		// Recursively permute the remaining elements
		permuteAndCompress(files, index+1, bestOrder, bestSize)

		// Restore the original order
		files[index], files[i] = files[i], files[index]
	}
}

// findBestStartFile finds the file with the worst compression ratio overall (least compressible)
func findBestStartFile(files []*TarFile, halfWindowSize int) int {
	bestIdx := 0
	worstRatio := 0.0

	for i, file := range files {
		// Skip files smaller than the compression window
		if len(file.Content) < halfWindowSize*2 {
			continue
		}
		
		// Compress the entire file content
		compressed, err := compressBytes(file.Content)
		if err != nil {
			continue
		}
		ratio := float64(len(compressed)) / float64(len(file.Content))

		// Find the file with the highest ratio (least compressible)
		if ratio > worstRatio {
			worstRatio = ratio
			bestIdx = i
		}
	}

	return bestIdx
}

// headerToBytes converts a tar.Header to bytes
func headerToBytes(header *tar.Header) ([]byte, error) {
	headerBytes := &bytes.Buffer{}
	tw := tar.NewWriter(headerBytes)
	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("failed to convert header to bytes: %w", err)
	}
	tw.Close()
	return headerBytes.Bytes(), nil
}

// findBestNextFile finds the file that compresses best when appended to the given file
func findBestNextFile(lastFile *TarFile, candidates []*TarFile, halfWindowSize int) int {
	if len(candidates) == 0 {
		return 0
	}

	type result struct {
		index int
		ratio float64
	}

	// Create a channel to collect results
	results := make(chan result, len(candidates))
	
	// Process each candidate in parallel
	var wg sync.WaitGroup
	for i, candidate := range candidates {
		wg.Add(1)
		go func(i int, candidate *TarFile) {
			defer wg.Done()

			// Get the candidate's header as bytes
			headerBytes, err := headerToBytes(candidate.Header)
			if err != nil {
				// If we can't get header bytes, just use an empty slice
				headerBytes = []byte{}
			}

			// Combine the last window of the previous file with the candidate's header and first window
			combined := append(append(lastFile.LastWindow, headerBytes...), candidate.FirstWindow...)
			compressed, err := compressBytes(combined)
			if err != nil {
				// Skip this candidate
				return
			}
			ratio := float64(len(compressed)) / float64(len(combined))

			// Send result to channel
			results <- result{index: i, ratio: ratio}
		}(i, candidate)
	}

	// Close the results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Find the best ratio from all results
	bestIdx := 0
	bestRatio := math.MaxFloat64
	resultsFound := false

	for r := range results {
		resultsFound = true
		if r.ratio < bestRatio {
			bestRatio = r.ratio
			bestIdx = r.index
		}
	}

	// If no valid results were found, return the first candidate as a fallback
	if !resultsFound && len(candidates) > 0 {
		return 0
	}

	return bestIdx
}

// compressBytes compresses a byte slice using klauspost/compress/gzip
func compressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw, err := kgzip.NewWriterLevel(&buf, kgzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}
	if _, err := gzw.Write(data); err != nil {
		gzw.Close() // Ensure the writer is closed even if Write fails
		return nil, fmt.Errorf("failed to write data to gzip writer: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return buf.Bytes(), nil
}

// createTarGz creates a tar.gz file from the provided files
func createTarGz(files []*TarFile) ([]byte, error) {
	var buf bytes.Buffer

	// Create gzip writer with maximum compression
	gzw, err := kgzip.NewWriterLevel(&buf, kgzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	// Create tar writer
	tw := tar.NewWriter(gzw)

	// Add files to the tar archive
	for _, file := range files {
		if err := tw.WriteHeader(file.Header); err != nil {
			return nil, fmt.Errorf("failed to write tar header: %w", err)
		}
		if _, err := tw.Write(file.Content); err != nil {
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

// validateChecksums validates that all files have the same checksums as the original
func validateChecksums(original, reordered []*TarFile) bool {
	if len(original) != len(reordered) {
		return false
	}

	// Create maps of checksums for easy lookup
	originalChecksums := make(map[[sha256.Size]byte]*TarFile, len(original))
	originalHeaderChecksums := make(map[[sha256.Size]byte]*TarFile, len(original))

	for _, file := range original {
		originalChecksums[file.Checksum] = file
		originalHeaderChecksums[file.HeaderHash] = file
	}

	// Verify all reordered files match the originals
	for _, file := range reordered {
		// For non-regular files, we only check header checksums
		if file.Header.Typeflag != tar.TypeReg {
			if _, exists := originalHeaderChecksums[file.HeaderHash]; !exists {
				return false
			}
		} else {
			// For regular files, check both content and header checksums
			if _, exists := originalChecksums[file.Checksum]; !exists {
				return false
			}
			if _, exists := originalHeaderChecksums[file.HeaderHash]; !exists {
				return false
			}
		}
	}

	return true
}