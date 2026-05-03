# locationid

Language-neutral repository for the `LocationCode` data type.

See `CONTRIBUTING.md` for commit, PR, and release conventions.

## Layout

```text
spec/  canonical format spec and shared test vectors
go/    Go implementation
ts/    future TypeScript implementation
```

## Go

The Go module lives at:

```text
github.com/ruckstackapp/locationid/go
```

## TypeScript

The TypeScript package lives in `ts/`.

## Releases

Go releases are managed with `release-please`.

Because the Go module lives in `go/`, tags are created with the module path prefix:

```text
go/v0.1.0
go/v0.1.1
go/v1.0.0
```

Workflow behavior:

1. `CI` runs on pull requests and pushes to `main`.
2. `Release Please` runs on pushes to `main`.
3. When releasable changes have landed, it opens or updates a release PR.
4. Merging that release PR creates the Git tag and GitHub release.

This keeps normal merges separate from intentional public releases.
