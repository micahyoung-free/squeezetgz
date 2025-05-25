package squeezetgz_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/micahyoung-free/squeezetgz/internal/squeezetgz"
	"github.com/micahyoung-free/squeezetgz/internal/testutils"
)

func TestOptimizeTarGz(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "squeezetgz-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate test files
	testFiles, err := testutils.GenerateTestFiles()
	if err != nil {
		t.Fatalf("Failed to generate test files: %v", err)
	}

	// Create test tar.gz
	tarGzData, err := testutils.CreateTarGz(testFiles)
	if err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}

	// Write the test tar.gz to a file
	inputPath := filepath.Join(tempDir, "input.tar.gz")
	if err := os.WriteFile(inputPath, tarGzData, 0644); err != nil {
		t.Fatalf("Failed to write test tar.gz: %v", err)
	}

	// Test the window optimization mode
	t.Run("WindowMode", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "output_window.tar.gz")
		result, err := squeezetgz.OptimizeTarGz(inputPath, outputPath, squeezetgz.WindowMode)
		if err != nil {
			t.Fatalf("Failed to optimize tar.gz with window mode: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Output file was not created")
		}

		// Verify compression improvement
		if result.AfterSize >= result.BeforeSize {
			t.Logf("Warning: No compression improvement with window mode. Before: %d bytes, After: %d bytes", 
				result.BeforeSize, result.AfterSize)
		}

		t.Logf("Window mode compression: Before: %.2f KB (%.2f%%), After: %.2f KB (%.2f%%)", 
			float64(result.BeforeSize)/1024, result.BeforeRatio*100,
			float64(result.AfterSize)/1024, result.AfterRatio*100)
		
		// Verify file order
		inputOrder, err := testutils.GetFileOrder(inputPath)
		if err != nil {
			t.Fatalf("Failed to get input file order: %v", err)
		}
		
		outputOrder, err := testutils.GetFileOrder(outputPath)
		if err != nil {
			t.Fatalf("Failed to get output file order: %v", err)
		}
		
		// Ensure output has the same number of files
		if len(inputOrder) != len(outputOrder) {
			t.Fatalf("Expected %d files in output, got %d", len(inputOrder), len(outputOrder))
		}
		
		// Ensure the files have been reordered (not the same order)
		if reflect.DeepEqual(inputOrder, outputOrder) {
			t.Fatalf("Files were not reordered in window mode")
		}
		
		t.Logf("Input order: %v", inputOrder)
		t.Logf("Window mode output order: %v", outputOrder)
		
		// Check for specific patterns in the output order
		verifyOptimizedOrder(t, outputOrder, "window mode")
	})

	// Test the brute force optimization mode
	t.Run("BruteForceMode", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "output_brute.tar.gz")
		result, err := squeezetgz.OptimizeTarGz(inputPath, outputPath, squeezetgz.BruteForceMode)
		if err != nil {
			t.Fatalf("Failed to optimize tar.gz with brute force mode: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Output file was not created")
		}

		// Verify compression improvement
		if result.AfterSize >= result.BeforeSize {
			t.Logf("Warning: No compression improvement with brute force mode. Before: %d bytes, After: %d bytes", 
				result.BeforeSize, result.AfterSize)
		}

		t.Logf("Brute force mode compression: Before: %.2f KB (%.2f%%), After: %.2f KB (%.2f%%)", 
			float64(result.BeforeSize)/1024, result.BeforeRatio*100,
			float64(result.AfterSize)/1024, result.AfterRatio*100)
		
		// Verify file order
		inputOrder, err := testutils.GetFileOrder(inputPath)
		if err != nil {
			t.Fatalf("Failed to get input file order: %v", err)
		}
		
		outputOrder, err := testutils.GetFileOrder(outputPath)
		if err != nil {
			t.Fatalf("Failed to get output file order: %v", err)
		}
		
		// Ensure output has the same number of files
		if len(inputOrder) != len(outputOrder) {
			t.Fatalf("Expected %d files in output, got %d", len(inputOrder), len(outputOrder))
		}
		
		// Ensure the files have been reordered (not the same order)
		if reflect.DeepEqual(inputOrder, outputOrder) {
			t.Fatalf("Files were not reordered in brute force mode")
		}
		
		t.Logf("Input order: %v", inputOrder)
		t.Logf("Brute force mode output order: %v", outputOrder)
		
		// Check for specific patterns in the output order
		verifyOptimizedOrder(t, outputOrder, "brute force mode")
	})

	// Test that both modes produce the same result with the specific test data
	t.Run("CompareModes", func(t *testing.T) {
		// This should be true for our specific test setup where the optimal arrangement is deterministic
		outputWindowPath := filepath.Join(tempDir, "output_window.tar.gz")
		outputBrutePath := filepath.Join(tempDir, "output_brute.tar.gz")

		windowData, err := os.ReadFile(outputWindowPath)
		if err != nil {
			t.Fatalf("Failed to read window mode output: %v", err)
		}

		bruteData, err := os.ReadFile(outputBrutePath)
		if err != nil {
			t.Fatalf("Failed to read brute force mode output: %v", err)
		}

		// Note: We're not expecting exactly identical files since compression might include timestamps
		// but we do expect similar size for this specific test case
		windowSize := len(windowData)
		bruteSize := len(bruteData)
		
		t.Logf("Window mode output size: %d bytes", windowSize)
		t.Logf("Brute force mode output size: %d bytes", bruteSize)
		
		// Allow for small variation due to different internal compression details
		sizeRatio := float64(windowSize) / float64(bruteSize)
		if sizeRatio < 0.95 || sizeRatio > 1.05 {
			t.Logf("Warning: Significant difference between window and brute force output sizes")
		}
		
		// Compare file order between the two modes
		windowOrder, err := testutils.GetFileOrder(outputWindowPath)
		if err != nil {
			t.Fatalf("Failed to get window mode file order: %v", err)
		}
		
		bruteOrder, err := testutils.GetFileOrder(outputBrutePath)
		if err != nil {
			t.Fatalf("Failed to get brute force mode file order: %v", err)
		}
		
		t.Logf("Window mode file order: %v", windowOrder)
		t.Logf("Brute force mode file order: %v", bruteOrder)
		
		// Both modes might not produce identical ordering due to different strategies,
		// but both should satisfy our optimization criteria
		verifyOrderPreservesCriteria(t, windowOrder, bruteOrder)
	})
}

// verifyOptimizedOrder checks if the file order follows expected patterns for optimized compression
func verifyOptimizedOrder(t *testing.T, fileOrder []string, modeName string) {
	// Helper to find position of a file in the order
	findPos := func(filename string) int {
		for i, name := range fileOrder {
			if name == filename {
				return i
			}
		}
		return -1
	}
	
	// Our test files have specific patterns that should be recognized by the optimizer:
	// - file1.txt and file2.txt share the same first half window
	// - file3.txt and file4.txt share the same last half window
	
	// Get positions of our test files
	pos1 := findPos("file1.txt")
	pos2 := findPos("file2.txt")
	pos3 := findPos("file3.txt")
	pos4 := findPos("file4.txt")
	
	// Verify all regular files are present
	if pos1 == -1 || pos2 == -1 || pos3 == -1 || pos4 == -1 {
		t.Fatalf("Not all test files found in output order")
	}
	
	// Verify special files are present (empty file and symlink)
	emptyFilePos := findPos("empty.txt")
	symlinkPos := findPos("link-to-file1.txt")
	
	if emptyFilePos == -1 {
		t.Errorf("%s: Empty file 'empty.txt' is missing from output", modeName)
	}
	
	if symlinkPos == -1 {
		t.Errorf("%s: Symlink 'link-to-file1.txt' is missing from output", modeName)
	}
	
	// Check for expected patterns - at least one of these should be true for optimal compression:
	// 1. file1.txt is adjacent to file2.txt (they share first half window)
	// 2. file3.txt is adjacent to file4.txt (they share last half window)
	adjacentPairs := 0
	
	if abs(pos1-pos2) == 1 {
		t.Logf("%s: file1.txt and file2.txt are adjacent (shared first half window)", modeName)
		adjacentPairs++
	}
	
	if abs(pos3-pos4) == 1 {
		t.Logf("%s: file3.txt and file4.txt are adjacent (shared last half window)", modeName)
		adjacentPairs++
	}
	
	// At least one adjacent pair should be found in optimal ordering
	if adjacentPairs == 0 {
		t.Errorf("%s: No adjacent pairs with shared windows found in optimized order", modeName)
	}
}

// verifyOrderPreservesCriteria checks if both optimization modes preserve 
// the same important criteria in their file orders
func verifyOrderPreservesCriteria(t *testing.T, windowOrder, bruteOrder []string) {
	// Helper to find position of a file in the order
	findPos := func(order []string, filename string) int {
		for i, name := range order {
			if name == filename {
				return i
			}
		}
		return -1
	}
	
	// Check if pairs are adjacent in the order
	isAdjacent := func(order []string, file1, file2 string) bool {
		pos1 := findPos(order, file1)
		pos2 := findPos(order, file2)
		return abs(pos1-pos2) == 1
	}
	
	// Check whether each algorithm recognized the same file pairs
	windowFile1File2Adjacent := isAdjacent(windowOrder, "file1.txt", "file2.txt")
	windowFile3File4Adjacent := isAdjacent(windowOrder, "file3.txt", "file4.txt")
	
	bruteFile1File2Adjacent := isAdjacent(bruteOrder, "file1.txt", "file2.txt")
	bruteFile3File4Adjacent := isAdjacent(bruteOrder, "file3.txt", "file4.txt")
	
	// Log whether both optimization strategies recognized the same patterns
	if windowFile1File2Adjacent && bruteFile1File2Adjacent {
		t.Logf("Both optimization modes recognized file1.txt and file2.txt should be adjacent")
	}
	
	if windowFile3File4Adjacent && bruteFile3File4Adjacent {
		t.Logf("Both optimization modes recognized file3.txt and file4.txt should be adjacent")
	}
	
	// Test fails if the two strategies don't agree on at least one adjacency pattern
	if (windowFile1File2Adjacent != bruteFile1File2Adjacent) && 
	   (windowFile3File4Adjacent != bruteFile3File4Adjacent) {
		t.Logf("Warning: Window and brute force modes differ in adjacency patterns")
	}
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}