# SqueezeTGZ

Command-line tool to maximize compression of tar.gz files by re-ordering the files within the tar archive to fully utilize compression window.

## Installation

```sh
go install github.com/micahyoung-free/squeezetgz/cmd/squeezetgz@latest
```

Or clone the repository and build manually:

```sh
git clone https://github.com/micahyoung-free/squeezetgz.git
cd squeezetgz
make build
```

## Usage

```sh
squeezetgz <input.tar.gz> <output.tar.gz>
```

Output:

```text
Before: <size in KB> <compression ratio in %>
After: <size in KB> <compression ratio in %>
```

### Options

* `--window`: use compression-window optimizing mode (default)
* `--brute`: use brute-force mode

## Optimization Modes

### Graph-based, compression-window optimizing mode (--window)

This mode:
* Loops through all files within the tar contents
* Starts by selecting the file with worst overall compression ratio (least compressible file)
* Next, selects the file with maximum compression ratio for the first (compression window / 2) bytes combined with the previous file's last (compression window / 2) bytes
* Collects checksums of all individual, original file contents and headers
* Continues until an optimally-compressed chain of files is built
* Generates a new tar.gz from the chain of files
* Validates checksums of all indivual file contents and headers
* Outputs the new tar.gz file

### Brute-force mode (--brute)

This mode:
* Tries every possibile combination of files and choose the one with the best overall compression ratio
* Only works with a small number of files (up to 10) due to factorial complexity

## Features

* Uses `klauspost/compress` for optimized compression speed
* Uses maximum gzip compression settings
* Recompresses input tar.gz to calculate and print comparable `Before` and `After` size output
* Maintains file integrity through checksum verification