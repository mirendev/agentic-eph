# Agent-reviewed pull requests on Miren previews

This repo is an example of a pipeline where an AI agent decides whether a pull
request merges, based on testing the change while it's running.

Every pull request is deployed to its own temporary copy of the app on a
[Miren](https://miren.md) cluster. A Claude agent running in GitHub Actions
tests that copy against the app's API contract. If the agent approves, the
workflow merges the pull request, and `main` deploys to production. If it
rejects, the check fails and the agent's findings are posted on the pull
request.

The app is **Brewbar**, a one-page coffee-ordering site in Go with a small JSON
API and no outside dependencies.

```
 pull request ──▶ unit-tests
                      │
                  preview ─────── miren deploy --ephemeral ──▶ pr-12.brewbar…  (lives 24h)
                      │                                              ▲
                  agent-review ── curl, compare to docs/api.md ──────┘
                      │
                  verdict ─────── merge ──▶ main ── Deploy ──▶ brewbar…  (production)
```

An **ephemeral version** is Miren's name for a preview deploy. It's a full,
running copy of the app built from the pull request's code. It gets its own web
address, runs beside production without touching it, and Miren deletes it after
a set time, 24 hours here.

## Why test the running app

Unit tests check the code you thought to test. A preview checks what users
would actually get. `scripts/open-breaking-pr.sh` shows the difference: it
renames the API's JSON fields to camelCase. The code compiles and every unit
test passes, but the web page now shows `$NaN` for every price, and any other
client using the API breaks too. The agent catches this by comparing the live
responses with `docs/api.md` and the fields the page reads.

## What's here

| Path | What it is |
|---|---|
| `main.go`, `orders.go` | The app: a web page plus `GET /api/menu` and `POST /api/orders` |
| `orders_test.go` | Unit tests |
| `static/index.html` | The ordering page |
| `docs/api.md` | The API contract the agent tests against |
| `.miren/app.toml` | Miren app config (app name `brewbar`) |
| `.github/workflows/pr-preview.yml` | Unit tests → preview deploy → agent review → merge |
| `.github/workflows/deploy.yml` | Deploys `main` to production |
| `.github/agent/report.jq` | Turns the agent's verdict into the pull request comment |
| `scripts/open-breaking-pr.sh` | Opens a pull request that passes unit tests but breaks the live API |

## Run it locally

```bash
go test ./...
go run .            # http://localhost:3000
```

## Set it up in your own repo

You need the `miren` CLI logged in to your cluster, and the `gh` CLI logged in
to GitHub.

1. **Copy this repo into your own GitHub repo** and push it to `main`. If you
   change the app name in `.miren/app.toml`, change `app: brewbar` in both
   workflows to match.

2. **Create the app on Miren with a first deploy from your laptop.**

   ```bash
   miren deploy
   ```

3. **Let GitHub Actions deploy to the app.** This creates a binding that
   trusts GitHub's short-lived OIDC identity tokens (a signed note from GitHub
   saying which repo and event a job came from) instead of a stored password.
   Previews need the `pull_request` event, which isn't allowed by default.

   ```bash
   miren auth ci add -a brewbar \
     --github OWNER/REPO \
     --allowed-events push,pull_request,workflow_dispatch \
     --description "GitHub Actions"
   ```

4. **Add repository secrets** (**Settings → Secrets and variables → Actions**):

   | Secret | Value |
   |---|---|
   | `MIREN_CLUSTER` | Output of `miren cluster export-address` |
   | `ANTHROPIC_API_KEY` | API key for the agent |
   | `MERGE_TOKEN` | A fine-grained personal access token, or a GitHub App token, with **Contents** and **Pull requests** write access to the repo |

   `MERGE_TOKEN` exists because merges made with the built-in `GITHUB_TOKEN`
   don't start other workflows, so the production deploy would never run. If
   the repo belongs to an organization that requires approval for fine-grained
   tokens, an org owner must approve the token before it can merge.

5. **Repo settings:** allow squash merging. If you protect `main`, require the
   `unit-tests` check and make sure the `MERGE_TOKEN` identity is allowed to
   merge without a human approval.

## Try it

**A change that should merge.** Ask a coding agent, such as Claude Code, for an
additive feature, then let it open the pull request. For example:

> Add an optional `milk` field to order items: `whole` (default, no charge) or
> `oat` (60 cents more). Return it on each quoted line, add a picker to the
> page, and update `docs/api.md` and the tests. Commit on a new branch and open
> a pull request.

Within a few minutes the pull request gets a preview, the agent's report, and a
merge, and the **Deploy** workflow ships it.

**A change that should be rejected.**

```bash
./scripts/open-breaking-pr.sh
```

The unit tests pass. The agent's report shows failed checks for the renamed
fields and the broken page, the check goes red, and production is untouched.

## How the gate works

1. `unit-tests` runs `go vet` and `go test`.
2. `preview` runs `miren deploy --ephemeral pr-<number> --ttl 24h` through
   `mirendev/actions/deploy`, then waits for `/healthz` to answer.
3. `agent-review` runs Claude Code through `anthropics/claude-code-action`. The
   agent can read the repo, run read-only `git` commands, and run `curl` and
   `jq`. It can't edit files, push, comment, or merge.
4. The agent returns its verdict as structured output (the action's
   `--json-schema` option), so the workflow gets checked JSON like this:

   ```json
   {
     "decision": "merge",
     "summary": "Adds drink sizes. Live prices match the contract.",
     "checks": [
       { "name": "large latte", "result": "pass", "detail": "1 large latte: total_cents 571 (525 + 46 tax)" }
     ]
   }
   ```

5. A follow-up step lists any tool calls the agent tried but wasn't allowed
   to make, in the job summary. Check there first if a run goes wrong.
6. The workflow posts the verdict as a pull request comment. On `merge` it
   squash-merges with `MERGE_TOKEN`, pinned to the commit the agent tested, so
   a push after testing makes the merge fail. On `reject` it fails the check.

The agent decides, but the workflow holds the merge permission. That keeps the
agent's access small and leaves a record of every decision.

## Limits worth knowing

- Previews only run for pull requests from branches in this repo. Pull requests
  from forks don't get secrets, so they stop after the unit tests.
- The agent is allowed to `curl` any address. Tighten `--allowedTools` in
  `pr-preview.yml` if that matters for your setup.
- The agent reads code from the pull request, which an attacker could write to
  steer it. That's acceptable when only trusted people can push branches. Don't
  copy this setup to a repo that runs previews for untrusted contributors.
- The agent's step-by-step work isn't shown in the Actions log. The action
  hides it by default because tool output can contain secrets. Its report on
  the pull request is the record.
