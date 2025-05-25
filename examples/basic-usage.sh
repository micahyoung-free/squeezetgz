#!/bin/bash

# This script demonstrates how to use squeezetgz with example data

# Check if squeezetgz is installed
if ! command -v ./bin/squeezetgz &> /dev/null; then
    echo "Building squeezetgz..."
    make build
fi

# Create a temporary directory
TEMP_DIR=$(mktemp -d)
echo "Created temporary directory: $TEMP_DIR"

# Create some example files
echo "Creating example files..."
for i in {1..5}; do
    # Create files with repeating patterns for better compression opportunities
    head -c 50000 /dev/urandom | tr -dc 'a-zA-Z0-9' > "$TEMP_DIR/file$i.txt"
done

# Create a tar.gz file
echo "Creating tar.gz file..."
tar -czf "$TEMP_DIR/original.tar.gz" -C "$TEMP_DIR" file1.txt file2.txt file3.txt file4.txt file5.txt

# Run squeezetgz
echo "Running squeezetgz..."
./bin/squeezetgz "$TEMP_DIR/original.tar.gz" "$TEMP_DIR/optimized.tar.gz"

# Compare sizes
echo "Comparing sizes..."
echo "Original size: $(du -h "$TEMP_DIR/original.tar.gz" | cut -f1)"
echo "Optimized size: $(du -h "$TEMP_DIR/optimized.tar.gz" | cut -f1)"

# Verify the optimized archive
echo "Verifying optimized archive..."
mkdir -p "$TEMP_DIR/extracted"
tar -xzf "$TEMP_DIR/optimized.tar.gz" -C "$TEMP_DIR/extracted"

# Count files to verify integrity
FILE_COUNT=$(find "$TEMP_DIR/extracted" -type f | wc -l)
echo "Extracted $FILE_COUNT files from the optimized archive"

# Cleanup
echo "Cleaning up..."
rm -rf "$TEMP_DIR"
echo "Done!"