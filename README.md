# Pixiv Follower Downloader (Go)

[中文](README.zh-CN.md)

A command-line Pixiv image downloader that:

- Reads a target user's following list
- Downloads original images from followed users
- Stores metadata in SQLite
- Cleans up local files that are not tracked in the database

## Source-Based Overview

This project is implemented in:

- `pixiv.go`: CLI entrypoint and download loop
- `lib/lib.go`: Pixiv API calls, image download, SQLite operations, cleanup logic

Current flow in code:

1. Prompt for `homeid` and `cookie`.
2. Initialize local DB/filesystem state.
3. Clean orphan/invalid local files (`DeleteBadImageFromRootfs`).
4. Fetch followed users from:
   - `GET /ajax/user/{homeid}/following?offset=0&limit=99&rest=show`
5. Fetch each followed user's illustration IDs from:
   - `GET /ajax/user/{uid}/profile/all`
6. For each illustration:
   - Skip if in blacklist DB.
   - Skip if already in SQL DB.
   - Fetch metadata from `GET /ajax/illust/{pid}`.
   - Download original image.
   - Save metadata into `pixiv.db` table `imgs`.

## Build

```bash
./build.sh
# or
go build -ldflags "-s -w"
```

## Run

```bash
./pixiv
```

Then input:

- `homeid` (Pixiv user ID)
- `cookie` (Pixiv cookie string)

## CLI Flags (Declared in Source)

- `-onlyBad` (bool): only run cleanup and exit
- `-debug` (bool): debug mode switch
- `-proxyType` (string): proxy type, `none` or `http`
- `-proxy` (string): proxy address
- `-thread` (int): max parallel workers (default `10`)

Important: `flag.Parse()` is not called in current source, so these flags currently stay at defaults unless code is updated.

## Output and Data

- Images: `img/<author_uid>/<illust_id>.jpg`
- Main DB: `pixiv.db`
  - Tables: `imgs`, `star`
- Blacklist DB: `Blacklist.db`
  - Table: `bad`

Saved metadata fields in `imgs`:

- `name`
- `id`
- `Author`
- `Authorid`
- `R18`
- `createDate`
- `tags`
- `size`

## Notes and Caveats

- Following list fetch currently requests only one page (`limit=99`).
- Network failures use recursive retry loops.
- Cookie input uses `fmt.Scanln`, which may break if spaces are present in the cookie string.
- Ensure your usage complies with Pixiv Terms of Service and local laws.

## License

No license file was found in the repository. Add one if you plan to distribute this project.

