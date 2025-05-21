package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/example/squeezetgz/internal/squeeze"
)

func main() {
	var brute bool
	flag.BoolVar(&brute, "brute", false, "use brute force mode")
	flag.BoolVar(&brute, "b", false, "use brute force mode (shorthand)")
	var window bool
	flag.BoolVar(&window, "window", true, "use compression window optimizing mode")
	flag.Parse()

	if flag.NArg() != 2 {
		log.Fatalf("usage: squeezetgz <input.tar.gz> <output.tar.gz>")
	}

	mode := squeeze.ModeWindow
	if brute {
		mode = squeeze.ModeBrute
	}

	in := flag.Arg(0)
	out := flag.Arg(1)

	stats, err := squeeze.Process(in, out, mode)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Before: %d KB %.2f%%\n", stats.BeforeKB, stats.BeforeRatio)
	fmt.Printf("After: %d KB %.2f%%\n", stats.AfterKB, stats.AfterRatio)
}
