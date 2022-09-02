# sccdiff

[![Build Status](https://github.com/vearutop/sccdiff/workflows/test-unit/badge.svg)](https://github.com/vearutop/sccdiff/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/vearutop/sccdiff/branch/master/graph/badge.svg)](https://codecov.io/gh/vearutop/sccdiff)
[![Time Tracker](https://wakatime.com/badge/github/vearutop/sccdiff.svg)](https://wakatime.com/badge/github/vearutop/sccdiff)
![Code lines](https://sloc.xyz/github/vearutop/sccdiff/?category=code)
![Comments](https://sloc.xyz/github/vearutop/sccdiff/?category=comments)


A tool to show the stats of code changes grouped by language, based on [`scc`](https://github.com/boyter/scc).

[Blog post](https://dev.to/vearutop/improving-changed-lines-visibility-in-github-pull-requests-4m7a).

## Usage

```
Usage of sccdiff:
  -all
        Include unmodified records in report.
  -basedir string
        Base directory.
  -baseref string
        Base reference. (default "HEAD")
  -groupdirs string
        Group files from directories, can be used to combine vendor, can have multiple comma separated values.
  -version
        Show app version and exit.
```

If there are no flags provided, `sccdiff` will try to check the code changes against `git` `HEAD` revision.

Result is an ASCII formatted table, suitable for Markdown.

```
| Language  | Files  |   Lines    |    Code    | Comments |  Blanks  | Complexity |    Bytes     |
|-----------|--------|------------|------------|----------|----------|------------|--------------|
| Go        | 2 (+2) | 385 (+385) | 298 (+298) | 1 (+1)   | 86 (+86) | 51 (+51)   | 7.4K (+7.4K) |
| Go (test) | 2 (+1) | 78 (+75)   | 58 (+56)   | 0        | 20 (+19) | 1 (+1)     | 1.8K (+1.7K) |
| License   | 1      | 21         | 17         | 0        | 4        | 0          | 1.1K (+13B)  |
| Makefile  | 1      | 40 (+1)    | 29 (+1)    | 4        | 7        | 2          | 1.2K (+42B)  |
| Markdown  | 1      | 30 (+13)   | 24 (+12)   | 0        | 6 (+1)   | 0          | 1.7K (+759B) |
| Shell     | 0 (-1) | 0 (-22)    | 0 (-15)    | 0 (-2)   | 0 (-5)   | 0          | 0B (-764B)   |
| YAML      | 5      | 308 (+3)   | 267 (+3)   | 25       | 16       | 0          | 9.8K (+49B)  |
```

### GitHub Action

This is an example configuration to report code stats changes as pull request comment.

```yaml
name: cloc
on:
  pull_request:

# Cancel the workflow in progress in newer build is about to start.
concurrency:
  group: ${{ github.workflow }}-${{ github.head_ref || github.run_id }}
  cancel-in-progress: true

jobs:
  cloc:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v2
        with:
          path: pr
      - name: Checkout base code
        uses: actions/checkout@v2
        with:
          ref: ${{ github.event.pull_request.base.sha }}
          path: base
      - name: Count Lines Of Code
        id: loc
        run: |
          curl -sLO https://github.com/vearutop/sccdiff/releases/download/v1.0.3/linux_amd64.tar.gz && tar xf linux_amd64.tar.gz
          sccdiff_hash=$(git hash-object ./sccdiff)
          [ "$sccdiff_hash" == "ae8a07b687bd3dba60861584efe724351aa7ff63" ] || (echo "::error::unexpected hash for sccdiff, possible tampering: $sccdiff_hash" && exit 1)
          OUTPUT=$(cd pr && ../sccdiff -basedir ../base)
          echo "${OUTPUT}"
          OUTPUT="${OUTPUT//$'\n'/%0A}"
          echo "::set-output name=diff::$OUTPUT"

      - name: Comment Code Lines
        continue-on-error: true
        uses: marocchino/sticky-pull-request-comment@v2
        with:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          header: LOC
          message: |
            ### Lines Of Code

            ${{ steps.loc.outputs.diff }}

```

Example output: https://github.com/vearutop/sccdiff/pull/2.
