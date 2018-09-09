package changefinder

import (
	"crypto/sha256"
	"fmt"
	"github.com/sarthakn7/rsyncx/commons"
	"github.com/sger/go-hashdir"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Generate a file containing the planned changes
func FindChanges(source *string, destination *string, changeFile *string) {
	emptyPath := ""
	fmt.Printf("Creating directory metadata for source: %s\n", *source)
	sourceMetadata := createDirectoryMetadata(source, &emptyPath)
	fmt.Printf("Creating directory metadata for destination: %s\n", *destination)
	destinationMetadata := createDirectoryMetadata(destination, &emptyPath)

	fmt.Printf("Finding required operations")
	operations := findRequiredOperations(sourceMetadata, destinationMetadata)

	for _, op := range operations {
		fmt.Printf("%s %s %s", op.OpType, op.Source, op.Destination)
	}
}

func createDirectoryMetadata(directoryPath, relativePathToDir *string) *commons.DirectoryMetadata {
	log.Printf("Creating directory metadata: %s\n", *directoryPath)

	name := filepath.Base(*directoryPath)
	//hash := createDirectoryHash(directoryPath)

	subdirectories := make([]*commons.DirectoryMetadata, 0)
	files := make([]*commons.FileMetadata, 0)

	//fileNameToFile := make(map[*string]*commons.FileMetadata)
	//subDirectoryNameToSubDirectory := make(map[*string]*commons.DirectoryMetadata)

	currentRelativePathToDir := filepath.Join(*relativePathToDir, name)

	err := filepath.Walk(*directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Unable to read path %q: %v\n", path, err)
			return err
		}
		if path == *directoryPath {
			return nil
		}
		if info.IsDir() {
			metadata := createDirectoryMetadata(&path, &currentRelativePathToDir)
			//directoryName := filepath.Base(path)
			//subDirectoryNameToSubDirectory[&directoryName] = metadata
			subdirectories = append(subdirectories, metadata)
		} else {
			metadata := createFileMetadata(&path, &currentRelativePathToDir, info.ModTime(), info.Size())
			if metadata != nil {
				//fileName := filepath.Base(path)
				//fileNameToFile[&fileName] = metadata
				files = append(files, metadata)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", directoryPath, err)
	}

	return &commons.DirectoryMetadata{&name, directoryPath /*hash, fileNameToFile, subDirectoryNameToSubDirectory,*/, files, subdirectories}
}

func createFileMetadata(filePath, relativePathToDir *string, modTime time.Time, size int64) *commons.FileMetadata {
	name := filepath.Base(*filePath)
	hash := createFileHash(filePath)

	if hash == nil {
		return nil
	}

	return &commons.FileMetadata{&name, filePath, relativePathToDir, hash, modTime, size}
}

func createDirectoryHash(directoryPath *string) []byte {
	hash, err := hashdir.Create(*directoryPath, "sha256")

	if err != nil {
		log.Fatal(err)
	}

	return []byte(hash)
}

func createFileHash(filePath *string) []byte {
	file, err := os.Open(*filePath)
	if err != nil {
		log.Printf("Cannot open file %s: %s", *filePath, err.Error())
		return nil
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Fatal(err)
	}
	return hasher.Sum(nil)
}

type fileSearchParams struct {
	name    *string
	hash    []byte
	modTime time.Time
	size    int64
}

// Creates a mapping from fileSearchParams to FileMetadata for all files in the directory
func createInvertedFileMapping(directoryMetadata *commons.DirectoryMetadata) map[*fileSearchParams]*commons.FileMetadata {
	invertedMapping := make(map[*fileSearchParams]*commons.FileMetadata)

	addFilesToInvertedMapping(directoryMetadata, invertedMapping)

	return invertedMapping
}

func addFilesToInvertedMapping(directoryMetadata *commons.DirectoryMetadata, invertedMapping map[*fileSearchParams]*commons.FileMetadata) {
	for _, file := range directoryMetadata.Files {
		fileParam := createFileSearchParam(file)
		invertedMapping[fileParam] = file
	}

	for _, directory := range directoryMetadata.Subdirectories {
		addFilesToInvertedMapping(directory, invertedMapping)
	}
}

func createFileSearchParam(file *commons.FileMetadata) *fileSearchParams {
	return &fileSearchParams{file.Name, file.Hash, file.ModTime, file.Size}
}

func findRequiredOperations(sourceMetadata, destinationMetadata *commons.DirectoryMetadata) []*commons.Operation {
	invertedSourceMapping := createInvertedFileMapping(sourceMetadata)
	invertedDestinationMapping := createInvertedFileMapping(destinationMetadata)

	operations := make([]*commons.Operation, 0)

	// TODO: directory creation/deletion ops
	findAddMoveOperationsInDirectory(sourceMetadata, destinationMetadata, invertedDestinationMapping, operations)
	findDeleteOperationsInDirectory(destinationMetadata, invertedSourceMapping, operations)

	return operations
}

func findAddMoveOperationsInDirectory(sourceMetadata, destinationMetadata *commons.DirectoryMetadata,
	invertedDestinationMapping map[*fileSearchParams]*commons.FileMetadata, operations []*commons.Operation) {

	for _, sourceFile := range sourceMetadata.Files {
		param := createFileSearchParam(sourceFile)
		destinationFile, found := invertedDestinationMapping[param]

		if !found {
			// Add file addition operation
			op := commons.Operation{commons.OpAddFile, sourceFile.CompletePath, sourceFile.RelativePathToDir}
			operations = append(operations, &op)
		} else {
			// Add file move operation
			op := commons.Operation{commons.OpMoveFile, destinationFile.CompletePath, sourceFile.RelativePathToDir}
			operations = append(operations, &op)
		}
	}

	for _, subDirectory := range sourceMetadata.Subdirectories {
		findAddMoveOperationsInDirectory(subDirectory, destinationMetadata, invertedDestinationMapping, operations)
	}
}

func findDeleteOperationsInDirectory(destinationMetadata *commons.DirectoryMetadata,
	invertedSourceMapping map[*fileSearchParams]*commons.FileMetadata, operations []*commons.Operation) {

	for _, destinationFile := range destinationMetadata.Files {
		param := createFileSearchParam(destinationFile)
		_, found := invertedSourceMapping[param]

		if !found {
			// Add file deletion operation
			op := commons.Operation{commons.OpDeleteFile, destinationFile.CompletePath, nil}
			operations = append(operations, &op)
		}
	}

	for _, subDirectory := range destinationMetadata.Subdirectories {
		findDeleteOperationsInDirectory(subDirectory, invertedSourceMapping, operations)
	}
}
