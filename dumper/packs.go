package dumper

import "strings"

func (d *Dumper) fetchPacks() ([]string, error) {
	packs, err := d.fetchPackNames()
	if err != nil {
		return nil, err
	}

	var shas []string
	for _, pack := range packs {
		idx := strings.Replace(pack, ".pack", ".idx", 1)
		packSHAs, err := d.fetchIdx(idx)
		if err != nil {
			continue
		}
		shas = append(shas, packSHAs...)
	}

	return shas, nil
}

func (d *Dumper) fetchPackNames() ([]string, error) {
	data, err := d.fetch(".git/objects/info/packs")
	if err != nil {
		return nil, err
	}

	var packs []string
	for line := range strings.SplitSeq(data, "\n") {
		line = strings.TrimSpace(line)
		pack, ok := strings.CutPrefix(line, "P ")
		if !ok {
			continue
		}
		packs = append(packs, pack)
	}

	return packs, nil
}
