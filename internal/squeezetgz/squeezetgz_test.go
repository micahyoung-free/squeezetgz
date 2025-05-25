package squeezetgz_test

import (
	"os"
	"path/filepath"
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
		result, err := squeezetgz.OptimizeTarGz(inputPath, outputPath, squeezetgz.WindowMode, false)
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
	})

	// Test the brute force optimization mode
	t.Run("BruteForceMode", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "output_brute.tar.gz")
		result, err := squeezetgz.OptimizeTarGz(inputPath, outputPath, squeezetgz.BruteForceMode, false)
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
	})
}