#!/usr/bin/env bash
# Opens a pull request that the agent review should reject: a "tidy-up" that
# renames every *_cents JSON field to camelCase. The Go code compiles and the
# unit tests still pass, but the live API no longer matches docs/api.md and the
# web page shows $NaN for every price.
set -euo pipefail

cd "$(dirname "$0")/.."

git fetch origin main
branch="tidy/camelcase-json-$(date +%s)"
git switch -c "$branch" origin/main

perl -pi -e 's/json:"([a-z]+)_cents"/json:"$1Cents"/g' orders.go

go test ./...

git commit -am "Use camelCase for JSON money fields

Matches the JavaScript naming style used across our other services."
git push -u origin "$branch"

gh pr create --fill --head "$branch"
