#!/usr/bin/env bash
# Opens the Act 2 pull request: a "tidy-up" that renames every *_cents JSON
# field to camelCase. The Go code compiles and the unit tests still pass, but
# the live API no longer matches docs/api.md and the web page shows $NaN.
#
# Use this when you want Act 2 to be deterministic instead of asking the local
# agent to make the change live.
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
