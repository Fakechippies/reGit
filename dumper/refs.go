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
	branches := uniqueStrings(append(commonBranches, d.options.Branches...))
	var refs []Ref

	for _, branch := range branches {
		for _, refPath := range branchRefPaths(branch) {
			data, err := d.fetch(".git/" + refPath)
			if err != nil {
				continue
			}

			sha := strings.TrimSpace(data)
			if !validSHA(sha) {
				continue
			}

			refs = append(refs, Ref{
				Name: refPath,
				SHA:  sha,
			})
		}
	}
	return refs
}

func (d *Dumper) fetchReflogs() ([]string, error) {
	var shas []string
	seen := map[string]struct{}{}
	add := func(values []string) {
		for _, sha := range values {
			if !validSHA(sha) {
				continue
			}
			if _, ok := seen[sha]; ok {
				continue
			}
			seen[sha] = struct{}{}
			shas = append(shas, sha)
		}
	}

	for _, logPath := range d.reflogPaths() {
		data, err := d.fetch(".git/" + logPath)
		if err != nil {
			continue
		}
		add(parseReflog(data))
	}

	if len(shas) == 0 {
		return nil, errNotFound("reflogs")
	}
	return shas, nil
}

func (d *Dumper) fetchExtraSHAs() []string {
	var shas []string
	seen := map[string]struct{}{}
	add := func(values []string) {
		for _, sha := range values {
			if !validSHA(sha) {
				continue
			}
			if _, ok := seen[sha]; ok {
				continue
			}
			seen[sha] = struct{}{}
			shas = append(shas, sha)
		}
	}

	for _, file := range commonDiscoveryFiles(d.options.Branches) {
		data, err := d.fetch(".git/" + file)
		if err != nil {
			continue
		}
		add(parseSHAs(data))
		add(parseRefs(data))
	}

	return shas
}

func (d *Dumper) reflogPaths() []string {
	branches := uniqueStrings(append(commonBranches, d.options.Branches...))
	paths := []string{
		"logs/HEAD",
		"logs/refs/stash",
	}
	for _, branch := range branches {
		paths = append(paths,
			"logs/refs/heads/"+branch,
			"logs/refs/remotes/origin/"+branch,
		)
	}
	return uniqueStrings(paths)
}

func branchRefPaths(branch string) []string {
	return []string{
		"refs/heads/" + branch,
		"refs/remotes/origin/" + branch,
		"refs/wip/index/refs/heads/" + branch,
		"refs/wip/wtree/refs/heads/" + branch,
	}
}

func commonDiscoveryFiles(branches []string) []string {
	files := []string{
		"FETCH_HEAD",
		"ORIG_HEAD",
		"config",
		"info/refs",
		"packed-refs",
		"logs/HEAD",
		"logs/refs/stash",
	}
	for _, branch := range uniqueStrings(append(commonBranches, branches...)) {
		files = append(files,
			"refs/heads/"+branch,
			"refs/remotes/origin/"+branch,
			"refs/wip/index/refs/heads/"+branch,
			"refs/wip/wtree/refs/heads/"+branch,
			"logs/refs/heads/"+branch,
			"logs/refs/remotes/origin/"+branch,
		)
	}
	return uniqueStrings(files)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

type errNotFound string

func (e errNotFound) Error() string {
	return string(e) + " not found"
}
