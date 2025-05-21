package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	windowMode := flag.Bool("window", true, "use compression window optimizing mode")
	bruteMode := flag.Bool("brute", false, "use brute force mode")
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s [--window|--brute] <input.tar.gz> <output.tar.gz>\n", os.Args[0])
		os.Exit(1)
	}

	if *bruteMode {
		*windowMode = false
	}

	inPath := flag.Arg(0)
	outPath := flag.Arg(1)

	files, err := readTarGZ(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %v\n", err)
		os.Exit(1)
	}

	var ordered []TarFile
	if *windowMode {
		ordered = reorderWindow(files)
	} else {
		ordered = reorderBrute(files)
	}

	err = writeTarGZ(outPath, ordered)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}

	beforeInfo, err := os.Stat(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to stat input: %v\n", err)
		os.Exit(1)
	}
	afterInfo, err := os.Stat(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to stat output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Before: %d KB\n", beforeInfo.Size()/1024)
	fmt.Printf("After: %d KB\n", afterInfo.Size()/1024)
}
