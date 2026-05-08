package dumper

import (
	"fmt"
	"strings"
)

func NewHandler(baseURL, baseDir string) (*Handler, error) {
	writer, err := newWriter(baseDir)
	if err != nil {
		return nil, err
	}

	return &Handler{
		dumper: New(baseURL),
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

	total := len(entries)
	h.reportProgress(report, total, 0, "")

	for i, entry := range entries {
		h.reportProgress(report, total, i, entry.Path)

		data, err := h.dumper.fetchBlob(entry.SHA)
		if err != nil {
			return fmt.Errorf("fetch blob %s (%s): %w", entry.Path, entry.SHA, err)
		}

		if err := h.writer.Write(entry.Path, data); err != nil {
			return fmt.Errorf("write %s: %w", entry.Path, err)
		}

		h.reportProgress(report, total, i+1, entry.Path)
	}

	return nil
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
