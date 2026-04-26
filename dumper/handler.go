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
	if err := h.dumper.testConn(); err != nil {
		return err
	}

	head, err := h.dumper.fetch(".git/HEAD")
	if err != nil {
		return err
	}

	branch := trimHEAD(head)
	latestCommit, err := h.dumper.fetch(fmt.Sprintf(".git/%s", branch))
	if err != nil {
		return err
	}

	latestCommit = strings.TrimSpace(latestCommit)
	if latestCommit == "" {
		return fmt.Errorf("empty commit SHA for branch %s", branch)
	}

	entries, err := h.dumper.fetchIndex()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		data, err := h.dumper.fetchBlob(entry.SHA)
		if err != nil {
			return fmt.Errorf("fetch blob %s (%s): %w", entry.Path, entry.SHA, err)
		}

		if err := h.writer.Write(entry.Path, data); err != nil {
			return fmt.Errorf("write %s: %w", entry.Path, err)
		}
	}

	return nil
}
