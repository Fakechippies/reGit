package dumper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func New(baseURL string) *Dumper {
	return NewWithOptions(baseURL, Options{})
}

func NewWithOptions(baseURL string, options Options) *Dumper {
	options = normalizeOptions(options)
	client := &http.Client{
		Timeout: options.Timeout,
	}
	if options.Proxy != "" {
		proxyURL, err := url.Parse(options.Proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	return &Dumper{
		BaseURL: baseURL,
		client:  client,
		options: options,
	}
}

func normalizeOptions(options Options) Options {
	if options.Jobs <= 0 {
		options.Jobs = defaultJobs
	}
	if options.Retries <= 0 {
		options.Retries = defaultRetries
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultTimeout
	}
	if options.Headers == nil {
		options.Headers = map[string]string{}
	}
	return options
}

func (d *Dumper) testConn() error {
	resp, err := d.request("")
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
	body, err := d.fetchBytes(path)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (d *Dumper) fetchBytes(path string) ([]byte, error) {
	resp, err := d.request(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	return io.ReadAll(resp.Body)
}

func (d *Dumper) request(path string) (*http.Response, error) {
	target := d.BaseURL
	if path != "" {
		target = fmt.Sprintf("%s/%s", d.BaseURL, path)
	}

	var lastErr error
	attempts := d.options.Retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		if d.options.UserAgent != "" {
			req.Header.Set("User-Agent", d.options.UserAgent)
		}
		for name, value := range d.options.Headers {
			req.Header.Set(name, value)
		}

		resp, err := d.client.Do(req)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("server returned %s", resp.Status)
		}
		if attempt+1 < attempts {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	return nil, lastErr
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
	body, err := d.fetchBytes(".git/index")
	if err != nil {
		return nil, err
	}

	return parseIndex(body), nil
}

func (d *Dumper) fetchObject(sha string) ([]byte, error) {
	if object, ok := d.lookupPackedObject(sha); ok {
		return object, nil
	}

	path := fmt.Sprintf(".git/objects/%s", shaToPath(sha))
	resp, err := d.request(path)
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

func (d *Dumper) lookupPackedObject(sha string) ([]byte, bool) {
	d.packsMu.Lock()
	defer d.packsMu.Unlock()

	if !d.packsLoaded {
		return nil, false
	}

	object, ok := d.packedObjects[sha]
	return object, ok
}

func (d *Dumper) fetchPack(pack string) ([]byte, error) {
	return d.fetchBytes(fmt.Sprintf(".git/objects/pack/%s", pack))
}

func (d *Dumper) fetchIdx(idx string) ([]string, error) {
	idxData, err := d.fetchBytes(fmt.Sprintf(".git/objects/pack/%s", idx))
	if err != nil {
		return nil, err
	}

	return parseIdx(idxData), nil
}
