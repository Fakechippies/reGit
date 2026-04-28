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

type Ref struct {
	Name string // refs/remote/origin/main OR refs/tags/v0.9.8
	SHA  string
}
