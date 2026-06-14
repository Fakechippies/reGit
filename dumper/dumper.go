package dumper

import (
	"fmt"
	"io"
	"net/http"
)

func New(baseURL string) *Dumper {
	return &Dumper{
		BaseURL: baseURL,
	}
}

func (d *Dumper) testConn() error {
	resp, err := http.Get(d.BaseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, d.BaseURL)
	}

	return nil
}

func (d *Dumper) fetch(path string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", d.BaseURL, path))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (d *Dumper) fetchCommit(commit string) (*Commit, error) {
	body, err := d.fetchObject(commit)
	if err != nil {
		return nil, err
	}

	return parseCommit(string(body)), nil
}

func (d *Dumper) fetchTree(tree string) ([]TreeEntry, error) {
	body, err := d.fetchObject(tree)
	if err != nil {
		return nil, err
	}

	return parseTree(body), nil
}

func (d *Dumper) fetchBlob(blob string) ([]byte, error) {
	body, err := d.fetchObject(blob)
	if err != nil {
		return nil, err
	}

	return splitObjectData(body)
}

func (d *Dumper) fetchIndex() ([]IndexEntry, error) {
	resp, err := http.Get(fmt.Sprintf("%s/.git/index", d.BaseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, ".git/index")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseIndex(body), nil
}

func (d *Dumper) fetchObject(sha string) ([]byte, error) {
	if d.packsLoaded {
		if object, ok := d.packedObjects[sha]; ok {
			return object, nil
		}
	}

	path := fmt.Sprintf(".git/objects/%s", shaToPath(sha))
	resp, err := http.Get(fmt.Sprintf("%s/%s", d.BaseURL, path))
	if err != nil {
		return d.fetchPackedObject(sha, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return d.fetchPackedObject(sha, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return d.fetchPackedObject(sha, err)
	}

	return decodeObject(body)
}

func (d *Dumper) fetchPack(pack string) ([]byte, error) {
	path := fmt.Sprintf(".git/objects/pack/%s", pack)
	resp, err := http.Get(fmt.Sprintf("%s/%s", d.BaseURL, path))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	return io.ReadAll(resp.Body)
}

func (d *Dumper) fetchIdx(idx string) ([]string, error) {
	path := fmt.Sprintf(".git/objects/pack/%s", idx)
	resp, err := http.Get(fmt.Sprintf("%s/%s", d.BaseURL, path))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	idxData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseIdx(idxData), nil
}
