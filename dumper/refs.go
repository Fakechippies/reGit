package dumper

import "strings"

func (d *Dumper) fetchPackedRefs() ([]Ref, error) {
	data, err := d.fetch(".git/packed-refs")
	if err != nil {
		return nil, err
	}
	return parsePackedRefs(data), nil
}

var commonBranches = []string{
	"main", "master", "dev", "develop", "development",
	"staging", "production", "prod", "test", "testing",
	"release", "hotfix", "feature", "feat", "fix",
}

func (d *Dumper) bruteForceBranches() []Ref {
	var refs []Ref

	for _, branch := range commonBranches {
		data, err := d.fetch(".git/refs/heads/" + branch)
		if err != nil {
			continue
		}

		sha := strings.TrimSpace(data)
		if len(sha) < 40 {
			continue
		}

		refs = append(refs, Ref{
			Name: "refs/heads/" + branch,
			SHA:  sha,
		})
	}
	return refs
}

func (d *Dumper) fetchReflogs() ([]string, error) {
	data, err := d.fetch(".git/logs/HEAD")
	if err != nil {
		return nil, err
	}

	return parseReflog(data), nil
}
