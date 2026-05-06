# S3 Cache for GitHub Actions

GitHub Action that allows you to cache build artifacts to S3 using parallel, multi-core compression.

Fork from [leroy-merlin-br/action-s3-cache](https://github.com/leroy-merlin-br/action-s3-cache).

Changes:
- Configurable parallel compression (tar+gzip, zip, tar+zstd)
- Parallel read pipeline and write pool for tar backends
- Parallel deflate worker pool for zip
- Upgrade golang version

## Prerequisites

- An AWS account. [Sign up here](https://aws.amazon.com/resources/create-account/).
- AWS Access and Secret Keys. More info [here](https://aws.amazon.com/premiumsupport/knowledge-center/create-access-key/).
- An S3 bucket.

## Compression

All backends use **parallel CPU cores** for maximum throughput. Select via the optional `compression` input:

| Value | Default | Speed | Ratio | Format |
|-------|---------|-------|-------|--------|
| `tar+gzip` | ✓ | ★★★☆☆ | ★★★☆☆ | `.tar.gz` — universal, widely compatible |
| `zip` | | ★★★★☆ | ★★★☆☆ | `.zip` — universal |
| `tar+zstd` | | ★★★★★ | ★★★★☆ | `.tar.zst` — fastest; best ratio |

`tar+zstd` is recommended for large caches (e.g. `node_modules`). `tar+gzip` is the default for broad compatibility.

> **Important:** A cache saved with one compression type cannot be restored with another.
> If you change `compression`, update your cache `key` as well to avoid a stale-cache miss.

## Usage

Add your AWS credentials as repository secrets: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

### Save cache

```yml
- name: Save cache
  uses: everest/action-s3-cache@v4
  with:
    action: put
    aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
    aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    aws-region: us-east-1
    bucket: your-bucket
    key: ${{ hashFiles('yarn.lock') }}
    artifacts: |
      node_modules/
    # compression: tar+gzip  # default; also: zip, tar+zstd
    # s3-class: STANDARD      # default; also: ONEZONE_IA, INTELLIGENT_TIERING, GLACIER, etc.
```

### Restore cache

```yml
- name: Restore cache
  uses: everest/action-s3-cache@v4
  with:
    action: get
    aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
    aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    aws-region: us-east-1
    bucket: your-bucket
    key: ${{ hashFiles('yarn.lock') }}
    # compression: tar+gzip  # must match the value used when saving
```

### Delete cache

```yml
- name: Delete cache
  uses: everest/action-s3-cache@v4
  with:
    action: delete
    aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
    aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    aws-region: us-east-1
    bucket: your-bucket
    key: ${{ hashFiles('yarn.lock') }}
```

## Full example

```yml
- name: Checkout
  uses: actions/checkout@v4

- name: Restore cache
  uses: everest/action-s3-cache@v4
  with:
    action: get
    aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
    aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    aws-region: us-east-1
    bucket: your-bucket
    key: ${{ hashFiles('yarn.lock') }}
    compression: tar+zstd

- name: Install dependencies
  run: yarn

- name: Save cache
  uses: everest/action-s3-cache@v4
  with:
    action: put
    aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
    aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    aws-region: us-east-1
    bucket: your-bucket
    s3-class: STANDARD_IA
    key: ${{ hashFiles('yarn.lock') }}
    compression: tar+zstd
    artifacts: |
      node_modules/
```

## Contributing

This action uses pre-compiled Go binaries that are built and committed to version control automatically by CI.

- [ ] TODO Change to uploading built binaries to GitHub release assets instead of committing to version control (this would prevent bloating the repo size and history)

1. Create a branch and make your code changes to the Go source files
2. Open a PR targeting `develop`
3. Once merged, CI creates a release PR with built binaries for all platforms
4. Review and merge the release PR
5. This triggers tag and GitHub release creation (`v1`, `v2`, `v3`, etc.)
