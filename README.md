# Brewbar: agent-reviewed pull requests on Miren previews

Brewbar is a small Go web app for pricing coffee orders. It exists to demo a
pipeline where every pull request is deployed to a temporary Miren preview, a
Claude agent tests that preview, and the pull request merges itself if the agent
approves. The presenter's script is in [DEMO.md](DEMO.md).

## What's here

| Path | What it is |
|---|---|
| `main.go`, `orders.go` | The app: a web page plus `GET /api/menu` and `POST /api/orders` |
| `orders_test.go` | Unit tests |
| `static/index.html` | The ordering page |
| `docs/api.md` | The API contract the CI agent tests against |
| `.miren/app.toml` | Miren app config (app name `brewbar`) |
| `.github/workflows/pr-preview.yml` | Unit tests → preview deploy → agent review → merge |
| `.github/workflows/deploy.yml` | Deploys `main` to production |
| `.github/agent/report.jq` | Turns the agent's verdict into the pull request comment |
| `demo/break-contract.sh` | Opens the Act 2 pull request: passes unit tests, breaks the live API |

## Run it locally

```bash
go test ./...
go run .            # http://localhost:3000
```

## One-time setup

You need the `miren` CLI logged in to the cluster you'll demo on, and the
`gh` CLI logged in to GitHub.

1. **Create the GitHub repo and push this code to `main`.**

2. **Create the app on Miren with a first deploy from your laptop.**

   ```bash
   miren deploy
   ```

3. **Let GitHub Actions deploy to the app.** This creates a binding that
   trusts GitHub's short-lived OIDC identity tokens (a signed note from GitHub
   saying which repo and event a job came from) instead of a stored password.
   Pull request previews need the `pull_request` event, which isn't allowed by
   default.

   ```bash
   miren auth ci add -a brewbar \
     --github OWNER/REPO \
     --allowed-events push,pull_request,workflow_dispatch \
     --description "Brewbar GitHub Actions"
   ```

4. **Add repository secrets** (**Settings → Secrets and variables → Actions**):

   | Secret | Value |
   |---|---|
   | `MIREN_CLUSTER` | Output of `miren cluster export-address` |
   | `ANTHROPIC_API_KEY` | API key for the CI agent |
   | `MERGE_TOKEN` | A fine-grained personal access token, or a GitHub App token, with **Contents** and **Pull requests** write access to this repo |

   `MERGE_TOKEN` exists because merges made with the built-in `GITHUB_TOKEN`
   don't start other workflows, so the production deploy would never run.

5. **Repo settings:** allow squash merging. If you protect `main`, require the
   `unit-tests` check and make sure the `MERGE_TOKEN` identity is allowed to
   merge without a human approval.

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
   squash-merges with `MERGE_TOKEN`. On `reject` it fails the check.

## Limits worth knowing

- Previews only run for pull requests from branches in this repo. Pull requests
  from forks don't get secrets, so they stop after the unit tests.
- The agent is allowed to `curl` any address. Tighten `--allowedTools` in
  `pr-preview.yml` if that matters for your setup.
- The agent reads code from the pull request, which an attacker could write to
  steer it. That's acceptable when only trusted people can push branches. Don't
  copy this setup to a repo that runs previews for untrusted contributors.
