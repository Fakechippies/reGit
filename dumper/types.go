package dumper

import "sync"

type Dumper struct {
	BaseURL string
}

type Handler struct {
	dumper *Dumper
	writer *Writer
}

type Writer struct {
	BaseDir string
	sync.Mutex
}

type Commit struct {
	Tree    string
	Parents []string
}

type TreeEntry struct {
	Mode string
	Name string
	SHA  string
}

type IndexEntry struct {
	SHA  string
	Path string
}
