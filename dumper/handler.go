package dumper

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"sync"
)

func NewHandler(baseURL, baseDir string) (*Handler, error) {
	return NewHandlerWithOptions(baseURL, baseDir, Options{})
}

func NewHandlerWithOptions(baseURL, baseDir string, options Options) (*Handler, error) {
	writer, err := newWriter(baseDir)
	if err != nil {
		return nil, err
	}

	return &Handler{
		dumper: NewWithOptions(baseURL, options),
		writer: writer,
	}, nil
}

func (h *Handler) Run() error {
	return h.RunWithProgress(nil)
}

func (h *Handler) RunWithProgress(report ProgressReporter) error {
	if err := h.dumper.testConn(); err != nil {
		return err
	}

	head, err := h.dumper.fetch(".git/HEAD")
	if err != nil {
		return err
	}

	latestCommit, err := h.resolveLatestCommit(head)
	if err != nil {
		return err
	}

	latestCommit = strings.TrimSpace(latestCommit)
	if latestCommit == "" {
		return fmt.Errorf("empty commit SHA")
	}

	entries, err := h.dumper.fetchIndex()
	if err != nil {
		return err
	}

	entries = mergeEntries(entries, h.fetchDiscoveredEntries(latestCommit))

	total := len(entries)
	recovered := 0
	failed := 0
	completed := 0
	h.reportProgress(report, total, 0, "")

	type result struct {
		path string
		ok   bool
	}

	jobs := h.dumper.options.Jobs
	work := make(chan IndexEntry)
	results := make(chan result)
	var workers sync.WaitGroup

	for range jobs {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for entry := range work {
				data, err := h.dumper.fetchBlob(entry.SHA)
				if err != nil {
					results <- result{path: entry.Path}
					continue
				}

				if err := h.writer.Write(entry.Path, data); err != nil {
					results <- result{path: entry.Path}
					continue
				}

				results <- result{path: entry.Path, ok: true}
			}
		}()
	}

	go func() {
		for _, entry := range entries {
			work <- entry
		}
		close(work)
		workers.Wait()
		close(results)
	}()

	for res := range results {
		completed++
		if res.ok {
			recovered++
		} else {
			failed++
		}
		h.reportProgress(report, total, completed, res.path)
	}

	if recovered == 0 && failed > 0 {
		return fmt.Errorf("failed to recover any files")
	}

	h.writeSanitizedConfig()

	return nil
}

func (h *Handler) writeSanitizedConfig() {
	data, err := h.dumper.fetch(".git/config")
	if err != nil {
		return
	}

	_ = h.writer.Write(".git/config", []byte(sanitizeGitConfig(data)))
}

func (h *Handler) fetchDiscoveredEntries(latestCommit string) []IndexEntry {
	seeds := h.discoveredSHAs(latestCommit)
	seenObjects := map[string]struct{}{}
	seenEntries := map[string]struct{}{}
	var entries []IndexEntry

	type queuedObject struct {
		SHA      string
		Path     string
		Rooted   bool
		Writable bool
	}

	queue := make([]queuedObject, 0, len(seeds))
	for i, sha := range seeds {
		queue = append(queue, queuedObject{
			SHA:      sha,
			Writable: i == 0,
		})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if !validSHA(current.SHA) {
			continue
		}
		if _, ok := seenObjects[current.SHA]; ok {
			continue
		}
		seenObjects[current.SHA] = struct{}{}

		object, err := h.dumper.fetchObject(current.SHA)
		if err != nil {
			continue
		}

		switch objectType(object) {
		case "commit":
			commit := parseCommit(string(object))
			if commit == nil {
				continue
			}
			queue = append(queue, queuedObject{
				SHA:      commit.Tree,
				Rooted:   true,
				Writable: current.Writable,
			})
			for _, parent := range commit.Parents {
				queue = append(queue, queuedObject{SHA: parent})
			}

		case "tree":
			if !current.Rooted || !current.Writable {
				continue
			}
			for _, entry := range parseTree(object) {
				entryPath := path.Join(current.Path, entry.Name)
				if strings.HasPrefix(entry.Mode, "04") {
					queue = append(queue, queuedObject{
						SHA:      entry.SHA,
						Path:     entryPath,
						Rooted:   true,
						Writable: true,
					})
					continue
				}

				key := entryPath + "\x00" + entry.SHA
				if _, ok := seenEntries[key]; ok {
					continue
				}
				seenEntries[key] = struct{}{}
				entries = append(entries, IndexEntry{
					SHA:  entry.SHA,
					Path: entryPath,
				})
			}
		}
	}

	return entries
}

func (h *Handler) discoveredSHAs(latestCommit string) []string {
	seen := map[string]struct{}{}
	var shas []string

	add := func(sha string) {
		sha = strings.TrimSpace(sha)
		if !validSHA(sha) {
			return
		}
		if _, ok := seen[sha]; ok {
			return
		}
		seen[sha] = struct{}{}
		shas = append(shas, sha)
	}

	add(latestCommit)

	for _, ref := range h.dumper.bruteForceBranches() {
		add(ref.SHA)
	}

	if refs, err := h.dumper.fetchPackedRefs(); err == nil {
		for _, ref := range refs {
			add(ref.SHA)
		}
	}

	if refs, err := h.dumper.fetchReflogs(); err == nil {
		for _, ref := range refs {
			add(ref)
		}
	}

	for _, sha := range h.dumper.fetchExtraSHAs() {
		add(sha)
	}

	if packSHAs, err := h.dumper.fetchPacks(); err == nil {
		for _, sha := range packSHAs {
			add(sha)
		}
	}

	return shas
}

func mergeEntries(base, extra []IndexEntry) []IndexEntry {
	seenPaths := make(map[string]struct{}, len(base))
	for _, entry := range base {
		seenPaths[entry.Path] = struct{}{}
	}

	for _, entry := range extra {
		if _, ok := seenPaths[entry.Path]; ok {
			continue
		}
		seenPaths[entry.Path] = struct{}{}
		base = append(base, entry)
	}

	return base
}

func objectType(data []byte) string {
	nullIdx := bytes.IndexByte(data, 0)
	if nullIdx == -1 {
		return ""
	}

	header := string(data[:nullIdx])
	objectType, _, _ := strings.Cut(header, " ")
	return objectType
}

func (h *Handler) reportProgress(report ProgressReporter, total, downloaded int, current string) {
	if report == nil {
		return
	}

	report(ProgressEvent{
		Total:      total,
		Downloaded: downloaded,
		Current:    current,
	})
}

func (h *Handler) resolveLatestCommit(head string) (string, error) {
	head = strings.TrimSpace(head)
	if validSHA(head) {
		return head, nil
	}

	branch := trimHEAD(head)
	if branch != "" {
		latestCommit, err := h.dumper.fetch(fmt.Sprintf(".git/%s", branch))
		if err == nil {
			latestCommit = strings.TrimSpace(latestCommit)
			if validSHA(latestCommit) {
				return latestCommit, nil
			}
		}

		refs, err := h.dumper.fetchPackedRefs()
		if err == nil {
			for _, ref := range refs {
				if ref.Name == branch && validSHA(ref.SHA) {
					return ref.SHA, nil
				}
			}
		}
	}

	for _, ref := range h.dumper.bruteForceBranches() {
		if validSHA(ref.SHA) {
			return ref.SHA, nil
		}
	}

	refs, err := h.dumper.fetchPackedRefs()
	if err == nil {
		for _, ref := range refs {
			if validSHA(ref.SHA) {
				return ref.SHA, nil
			}
		}
	}

	refsFromLog, err := h.dumper.fetchReflogs()
	if err == nil {
		for i := len(refsFromLog) - 1; i >= 0; i-- {
			if validSHA(refsFromLog[i]) {
				return refsFromLog[i], nil
			}
		}
	}

	return "", fmt.Errorf("unable to resolve latest commit from HEAD, packed refs, or reflog")
}

func validSHA(sha string) bool {
	sha = strings.TrimSpace(sha)
	if len(sha) != 40 {
		return false
	}

	for _, ch := range sha {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}

	return true
}
