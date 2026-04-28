# reGit

`reGit` is a small Go CLI for reconstructing a working tree from an exposed `.git` directory on a web server. It fetches the remote repository index, downloads referenced Git objects, and writes the recovered files into a local directory.

This project is inspired by [`arthaud/git-dumper`](https://github.com/arthaud/git-dumper), a more complete repository dumping tool. `reGit` focuses on a smaller Go-based workflow centered on recovering files from `.git/index` and loose objects.

## Features

- Simple CLI with two required arguments: target URL and output directory
- Rebuilds files directly from `.git/index` and the referenced blob objects
- Preserves nested paths when writing recovered files
- Single static binary workflow with minimal runtime overhead

## Requirements

- Go `1.26.1` or newer to build from source
- Network access to the target web server
- A target that exposes `.git/index` and the corresponding object files under `.git/objects/`

## Build

```bash
go build -o reGit .
```

You can also run it without building an explicit binary:

```bash
go run . <url> <output-dir>
```

## Usage

```bash
reGit <url> <output-dir>
```

Example:

```bash
reGit http://localhost:8000/ dump-1
```

## How It Works

`reGit` performs the following steps:

1. Connects to the target URL and confirms it is reachable.
2. Reads `.git/HEAD` and the current branch reference.
3. Downloads `.git/index` from the target.
4. Extracts file paths and blob SHAs from the index.
5. Fetches the corresponding objects from `.git/objects/`.
6. Decompresses each object and writes the recovered file into the output directory.

## Limitations

- The current implementation reconstructs files from the Git index; it does not rebuild history, branches, tags, or a full repository structure.
- Success depends on the remote server exposing both `.git/index` and the required object files.
- If objects are missing or access is blocked, recovery will be partial or fail.

## Inspiration

If you need broader recovery coverage, including additional ref discovery and full repository reconstruction strategies, see [`git-dumper`](https://github.com/arthaud/git-dumper).

## Run

```bash
reGit http://IP:PORT DIR
```

## License

This project is licensed under the MIT License. See [LICENSE](/home/chips/Projects/reGit/LICENSE).

## In Progress

- [x] packed-refs + branch brute-force -> more SHAs to dump
- [ ] pack discovery (`info/packs`) -> get objects that 404 as loose
- [x] reflog (`logs/HEAD`) -> orphaned commits, deleted branches
- [ ] submodules (`.gitmodules`) -> recursive dumps
- [ ] LFS -> large assets
