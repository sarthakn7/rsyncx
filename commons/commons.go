package commons

import "time"

type FileMetadata struct {
	Name              *string
	CompletePath      *string // Complete path on disk
	RelativePathToDir *string // Path relative to source/destination directory
	Hash              []byte
	ModTime           time.Time
	Size              int64
}

type DirectoryMetadata struct {
	Name *string
	Path *string
	//Hash                           []byte
	//FileNameToFile                 map[*string]*FileMetadata
	//SubDirectoryNameToSubDirectory map[*string]*DirectoryMetadata
	Files          []*FileMetadata
	Subdirectories []*DirectoryMetadata
}

type OpType int

// Possible file operation types
const (
	OpAddFile OpType = iota + 1
	OpDeleteFile
	OpMoveFile
	OpCreateDirectory
	OpDeleteDirectory
)

type Operation struct {
	OpType      OpType
	Source      *string
	Destination *string
}
