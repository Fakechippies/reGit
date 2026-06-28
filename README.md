# reGit

`reGit` is a small Go CLI for reconstructing a working tree from an exposed `.git` directory on a web server. It fetches the remote repository index, downloads referenced Git objects, and writes the recovered files into a local directory.

This project is inspired by [`arthaud/git-dumper`](https://github.com/arthaud/git-dumper), a more complete repository dumping tool.

## Features

- Recovers files from `.git/index` (loose and packed objects)
- Walks commit trees to discover files not in the index
- Pack-file support with full delta resolution (OFsDelta and REF_Delta)
- Branch brute-force (`main`, `master`, `dev`, `develop`, `staging`, etc.)
- User-specified branch discovery
- Packed-refs parsing
- Reflog parsing (`logs/HEAD`, stash, branch, and remote branch logs)
- Extra SHA discovery from `FETCH_HEAD`, `ORIG_HEAD`, `info/refs`, `packed-refs`, config, and common ref files
- Concurrent downloads with configurable jobs, retries, timeout, headers, user-agent, and HTTP proxy
- Sanitized recovered `.git/config`
- Progress TUI (bubbletea) in terminals
- Single static binary, no runtime dependencies

## Requirements

- Go `1.26.1` or newer to build from source
- Network access to the target web server

## Build

```bash
go build -o reGit .
```

```bash
go run . <url> <output-dir>
```

## Usage

```bash
reGit http://target/.git dump-output
```

Useful options:

```bash
reGit -b staging -b production -j 20 -r 5 -t 5s \
  -u "Mozilla/5.0" \
  -H "Authorization: Bearer TOKEN" \
  --proxy http://127.0.0.1:8080 \
  http://target/.git dump-output
```

| Flag | Description |
|---|---|
| `-b`, `--branch` | Extra branch name to try in heads, remotes, logs, and wip refs |
| `-j`, `--jobs` | Concurrent file downloads, default `10` |
| `-r`, `--retries` | Retry count per request, default `3` |
| `-t`, `--timeout` | HTTP request timeout, default `3s` |
| `-u`, `--user-agent` | Custom HTTP user-agent |
| `-H`, `--header` | Custom HTTP header as `Name: value`; may be repeated |
| `--proxy` | HTTP proxy URL |

## How It Works

1. Tests connectivity to the target.
2. Reads `.git/HEAD` and resolves the current branch/commit.
3. Downloads `.git/index` and extracts file paths + blob SHAs.
4. Walks the commit tree to discover additional files not in the index.
5. Gathers more SHA seeds from packed-refs, reflogs, user/common branch refs, extra Git metadata files, and pack indexes.
6. Fetches loose objects from `.git/objects/`; falls back to pack files on 404.
7. Decompresses (or delta-resolves) each object and writes recovered files concurrently.
8. Writes a sanitized `.git/config` if the remote config is available.

## Comparison to git-dumper

`git-dumper` (Python) remains more feature-complete:

| Feature | reGit | git-dumper |
|---|---|---|
| Index recovery | ✓ | ✓ |
| Loose objects | ✓ | ✓ |
| Pack files (idx + delta resolution) | ✓ | ✓ (via dulwich) |
| Branch brute-force | ✓ | ✓ |
| Packed-refs | ✓ | ✓ |
| Reflog | ✓ | ✓ |
| Commit tree walking | ✓ | ✓ |
| Recursive object discovery | ✓ | ✓ |
| Directory listing (recursive wget) | ✗ | ✓ |
| Concurrent downloads | ✓ | ✓ (multiprocessing) |
| User-specified branches | ✓ (`-b` flag) | ✓ (`-b` flag) |
| Proxy support | HTTP proxy | HTTP, SOCKS4, SOCKS5 |
| Client certificates | ✗ | ✓ |
| Retry / timeout config | ✓ | ✓ |
| Custom headers / user-agent | ✓ | ✓ |
| Sanitize `.git/config` | ✓ | ✓ |
| `git checkout .` final step | ✗ | ✓ |
| Tag object support | ✗ | ✓ |
| FETCH_HEAD / ORIG_HEAD / stash / wip refs | ✓ | ✓ |
| Progress TUI | ✓ | ✗ |
| Single static binary | ✓ | ✗ (requires Python + deps) |

## In Progress

- [x] packed-refs + branch brute-force -> more SHAs to dump
- [x] pack discovery (`info/packs`) -> get objects that 404 as loose
- [x] reflog (`logs/HEAD`) -> orphaned commits, deleted branches
- [x] user branches, retries, timeout, headers, user-agent, HTTP proxy, and concurrent downloads
- [ ] submodules (`.gitmodules`) -> recursive dumps
- [ ] LFS -> large assets

## License

MIT License. See [LICENSE](https://github.com/Fakechippies/reGit/blob/main/LICENSE).
