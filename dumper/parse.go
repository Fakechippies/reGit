package dumper

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

func decodeObject(body []byte) ([]byte, error) {
	bodyBytes := bytes.NewReader(body)
	reader, err := zlib.NewReader(bodyBytes)
	if err != nil {
		return nil, err
	}

	finalBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return finalBytes, nil
}

func splitObjectData(data []byte) ([]byte, error) {
	nullIdx := bytes.IndexByte(data, 0)
	if nullIdx == -1 {
		return nil, fmt.Errorf("invalid git object data")
	}

	return data[nullIdx+1:], nil
}

func parseIndex(data []byte) []IndexEntry {
	count := int(binary.BigEndian.Uint32(data[8:12]))
	i := 12

	var entries []IndexEntry
	for range count {
		i += 40
		sha := hex.EncodeToString(data[i : i+20])
		i += 20

		length := binary.BigEndian.Uint16(data[i : i+2])
		pathLen := int(length & 0x0FFF)
		i += 2

		path := string(data[i : i+pathLen])
		i += pathLen + 1

		entryLen := 62 + pathLen + 1
		pad := (8 - entryLen%8) % 8
		i += pad

		entries = append(entries, IndexEntry{
			SHA:  sha,
			Path: path,
		})
	}

	return entries
}

func parseTree(data []byte) []TreeEntry {
	nullIdx := bytes.IndexByte(data, 0)
	if nullIdx == -1 {
		return nil
	}
	data = data[nullIdx+1:]

	var trees []TreeEntry
	i := 0
	for i < len(data) {
		null := bytes.IndexByte(data[i:], 0)
		if null == -1 {
			break
		}

		entry := data[i : i+null]
		mode, name, found := bytes.Cut(entry, []byte(" "))
		if !found {
			break
		}

		i += null + 1
		sha := hex.EncodeToString(data[i : i+20])
		i += 20

		trees = append(trees, TreeEntry{
			Mode: string(mode),
			Name: string(name),
			SHA:  sha,
		})
	}

	return trees
}

func parseCommit(data string) *Commit {
	nullIdx := strings.IndexByte(data, 0)
	if nullIdx == -1 {
		return nil
	}

	content := data[nullIdx+1:]
	commit := &Commit{}

	for line := range strings.SplitSeq(content, "\n") {
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "tree ") {
			commit.Tree = strings.TrimSpace(strings.TrimPrefix(line, "tree "))
			continue
		}

		if strings.HasPrefix(line, "parent ") {
			commit.Parents = append(commit.Parents, strings.TrimSpace(strings.TrimPrefix(line, "parent ")))
		}
	}

	return commit
}

func parsePackedRefs(data string) []Ref {
	var refs []Ref

	for line := range strings.SplitSeq(data, "\n") {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		refs = append(refs, Ref{
			SHA:  parts[0],
			Name: strings.TrimSpace(parts[1]),
		})
	}
	return refs
}

func parseReflog(data string) []string {
	var refs []string

	for line := range strings.SplitSeq(data, "\n") {
		parts := strings.Split(line, " ")
		if len(parts) < 2 || parts[1] == "0000000000000000000000000000000000000000" {
			continue
		}
		refs = append(refs, parts[1])
	}
	return refs
}

func parseIdx(data []byte) []string {
	b := data[1032:]
	count := int(binary.BigEndian.Uint32(data[1028:1032]))

	shas := make([]string, count)
	for i := 0; i < count; i++ {
		sha := hex.EncodeToString(b[i*20 : i*20+20])
		shas[i] = sha
	}
	return shas
}

func shaToPath(sha string) string {
	return fmt.Sprintf("%s/%s", sha[:2], sha[2:])
}

func trimHEAD(head string) string {
	return strings.TrimSpace(strings.TrimPrefix(head, "ref: "))
}
