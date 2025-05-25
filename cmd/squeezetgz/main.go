package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/micahyoung-free/squeezetgz/internal/squeezetgz"
)

func main() {
	// Define command line flags
	bruteMode := flag.Bool("brute", false, "Use brute-force mode")
	_ = flag.Bool("window", true, "Use compression-window optimizing mode (default)")

	// Parse flags
	flag.Parse()

	// Check if we have enough arguments
	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input.tar.gz> <output.tar.gz>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Get input and output file paths
	inputFile := flag.Arg(0)
	outputFile := flag.Arg(1)

	// Determine the optimization mode
	var mode squeezetgz.OptimizationMode
	if *bruteMode {
		mode = squeezetgz.BruteForceMode
	} else {
		mode = squeezetgz.WindowMode
	}

	// Run the optimization
	result, err := squeezetgz.OptimizeTarGz(inputFile, outputFile, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print the results
	fmt.Printf("Before: %.2f KB %.2f%%\n", float64(result.BeforeSize)/1024, result.BeforeRatio*100)
	fmt.Printf("After: %.2f KB %.2f%%\n", float64(result.AfterSize)/1024, result.AfterRatio*100)
}