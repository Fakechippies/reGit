package dumper

import (
	"net/http"
	"sync"
	"time"
)

const (
	defaultJobs    = 10
	defaultRetries = 3
	defaultTimeout = 3 * time.Second
)

type Options struct {
	Branches  []string
	Headers   map[string]string
	UserAgent string
	Proxy     string
	Jobs      int
	Retries   int
	Timeout   time.Duration
}

type Dumper struct {
	BaseURL       string
	client        *http.Client
	options       Options
	packsLoaded   bool
	packedObjects map[string][]byte
	packsMu       sync.Mutex
}

type Handler struct {
	dumper *Dumper
	writer *Writer
}

type ProgressEvent struct {
	Total      int
	Downloaded int
	Current    string
}

type ProgressReporter func(ProgressEvent)

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
