package memorymanager

import (
	"io"
)

type InputFile struct {
	Name   string
	Reader io.Reader
}

type MatchingChunk struct {
	File  File      `json:"file"`
	Chunk FileChunk `json:"chunk"`
	Score float32   `json:"score"`
}

type File struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
}

type FileChunk struct {
	Id         string `json:"id"`
	FileId     string `json:"file_id"`
	ChunkIndex int    `json:"chunk_index"`
	Content    string `json:"content"`
}
