package dumper

import (
	"bufio"
	"os"
	"path/filepath"
)

func newWriter(baseDir string) (*Writer, error) {
	err := os.MkdirAll(baseDir, 0o755)
	if err != nil {
		return nil, err
	}

	return &Writer{
		BaseDir: baseDir,
	}, nil
}

func (w *Writer) Write(path string, data []byte) error {
	w.Lock()
	defer w.Unlock()

	fullPath := filepath.Join(w.BaseDir, path)
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	if err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bufio.NewWriter(f)
	if _, err := buf.Write(data); err != nil {
		return err
	}

	return buf.Flush()
}
