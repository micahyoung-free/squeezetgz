# SqueezeTGZ

Command-line tool to maximize compression of tar.gz files by re-ordering the files within the tar archive to fully utilize compression window.

## Requirements

* Use `klauspost/compress` for optimized compression speed
* Use maximum gzip compression settings
* Recompress input tar.gz in order to calculate and print comparable `Before` and `After` size output
* Generate tests that use:
  * four generated alphabetical testdata files whose first and last (compression window / 2) bytes containing identical bytes to exactly one other file
  * two generated testdata files with random noise
  * scenarios that start with statically-shuffled order and confirm that optimal file orders are found for both the compression-window mode and brute-force mode

### Graph-based, compression-window optimizing mode

* Loop through all files within the tar contents
* Start by selecting file with worst overall compression ratio
* Next, select the file with maximum compression ratio for the first (compression window / 2) bytes combined with the previous file's last (compression window / 2) bytes
* Collect checksums of all individual, original file contents and headers
* Continue until an optimally-compressed chain of files is built
* Generate a new tar.gz from the chain of files
* Validate all checksums from original file contents and header are present in the new chain of files
* Output the new tar.gz file

### Brute-force mode

* Try every possibile combination of files and choose the one with the best overall compression ratio

## Usage

```sh
squeezetgz <input.tar.gz> <output.tar.gz>
```

Output:

```text
Before: <size in KB> <compression ratio in %>
After: <size in KB> <compression ratio in %>
```

Options:

* `--window`: use compression-window optimizing mode (default)
* `--brute`: use brute-force mode
