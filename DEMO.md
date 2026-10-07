# Demo script: an agent ships a change, another agent tests it live and merges it

**Length:** about 12 minutes, two acts.

**What the audience sees:** a coding agent on the presenter's laptop opens a
pull request. GitHub Actions deploys that pull request to its own temporary copy
of the app on a Miren cluster. A second agent, running inside CI, tests the
live copy and decides whether to merge. In Act 1 the change is good and gets
merged and shipped with no human clicking anything. In Act 2 the change passes
every unit test but breaks the live app, and the agent refuses it.

The app is **Brewbar**, a one-page coffee-ordering site with a small JSON API.

## The pieces

| Piece | Where it runs | What it does |
|---|---|---|
| Author agent | Claude Code on the presenter's laptop | Writes the change and opens the pull request |
| `unit-tests` job | GitHub Actions | `go vet` and `go test` |
| `preview` job | GitHub Actions → Miren | Deploys the pull request as an ephemeral version named `pr-<number>` |
| `agent-review` job | GitHub Actions | A second Claude agent tests the preview with `curl` and writes a verdict |
| Merge step | GitHub Actions | Merges if the verdict says `merge` |
| `Deploy` workflow | GitHub Actions → Miren | Ships `main` to production after the merge |

An **ephemeral version** (Miren's term for a preview deploy) is a full, running
copy of the app built from the pull request's code. It gets its own web address,
runs beside production without touching it, and Miren deletes it on its own
after a set time, 24 hours here.

```
 laptop                GitHub Actions                         Miren cluster
 ──────                ──────────────                         ─────────────
 author agent ──PR──▶  unit-tests
                           │
                       preview ─────── miren deploy ────────▶ pr-12.brewbar…  (lives 24h)
                           │                                        ▲
                       agent-review ── curl, compare to contract ───┘
                           │
                       verdict.json ── merge ──▶ main ── Deploy ─▶ brewbar…  (production)
```

## Before you go on stage

Do the one-time setup in [README.md](README.md) first. Then, on the day:

- [ ] `main` is green and production is serving the starting version of the app.
- [ ] No open pull requests left over from a rehearsal.
- [ ] Browser tabs open: the repo's **Pull requests** page, the **Actions** page,
      and production Brewbar.
- [ ] A terminal in the repo with Claude Code started (`claude`).
- [ ] A second terminal showing `miren app versions -a brewbar`. You will
      re-run it to show the preview appear.
- [ ] Rehearse each act once on the day. Note how long the `agent-review` job
      takes so you know how much time to fill.

## Opening (1 minute)

**Say:** "Agents can write code fast. The hard part is trusting what they
write. Code review by reading the diff doesn't tell you whether the thing
actually works once it's deployed. So I want to show the agent's change
running on real infrastructure, tested by another agent against the real
running app, and merged only if it holds up."

**Show:** production Brewbar. Order two lattes with code `WELCOME10`. Point out
the total: **$8.81**.

**Show:** `docs/api.md`. "This is our API contract. The web page and a mobile
app depend on these field names. Adding things is fine. Renaming things breaks
customers."

## Act 1: a good change gets merged (5–6 minutes)

### 1. The author agent writes the change

**Do:** in Claude Code, paste:

> Add drink sizes to Brewbar. Each order item can take an optional `size`:
> `small` is 50 cents less than the menu price, `medium` is the menu price and
> the default, `large` is 75 cents more. Return the size on each quoted line.
> Add a size picker to the page, update `docs/api.md` and the tests, then commit
> on a new branch and open a pull request.

**Say, while it works:** "Nothing special about this agent. It's the same coding
agent I use every day. It runs the unit tests locally, and if they pass it
opens a pull request. Once that pull request exists, the agent on my laptop
is out of the picture."

### 2. CI deploys the pull request to its own preview

**Show:** the pull request, then click through to the **PR preview** run.

**Say:** "First the unit tests. Then this `preview` job deploys this exact
pull request to our Miren cluster as an ephemeral version called
`pr-<number>`. It logs in to the cluster with GitHub's short-lived identity
token, so there's no long-lived cluster password stored in GitHub."

**Do:** in the second terminal, re-run `miren app versions -a brewbar`.

**Show:** the new `pr-<number>` version next to production. Open the preview
link from the job summary. Pick a large latte to show the new picker works.

**Say:** "That's a real deployment with its own address, running next to
production. It deletes itself in 24 hours."

### 3. The CI agent tests the live preview

**Show:** the `agent-review` job log as it runs.

**Say:** "Now a second agent takes over. It isn't reading the code and guessing.
It reads the contract, works out the prices it expects by hand, sends real
requests to the preview, and compares. It also checks that every field the web
page reads is still in the responses. Then it tests the new feature the pull
request claims to add."

Things to point at in the log as they scroll by:

- It computing an expected total and then checking the response against it.
- It trying a `size: "large"` order on the preview.
- It writing `.agent/verdict.json`.

**Say:** "The agent decides, but it doesn't hold the merge button. It writes a
verdict file. The workflow reads that and does the merge. That keeps the
agent's permissions small and leaves an audit trail."

### 4. The merge and the production deploy

**Show:** the comment the workflow posted on the pull request: a summary and a
table of every check with what was sent and what came back.

**Show:** the pull request is merged. Open the **Deploy** run that started from
the merge.

**Show:** refresh production Brewbar. The size picker is live.

**Say:** "From prompt to production with no human approving anything. And
every step left a record: the preview, the agent's test results, the merge."

## Act 2: a change that passes tests but breaks the app (4–5 minutes)

### 1. The breaking change

**Say:** "That's the happy path. The interesting case is a change that looks
fine. Here's a tidy-up: rename our JSON money fields to camelCase to match our
JavaScript style. It compiles, and every unit test passes."

**Do:** run the scripted version so the break is guaranteed:

```bash
./demo/break-contract.sh
```

Or, if you want to do it live, give Claude Code this prompt. The live version
is less predictable, because a careful agent may update the page too, which
fixes the page but still breaks the mobile app:

> Rename the JSON fields in `orders.go` from snake_case to camelCase to match
> our JavaScript style. Commit on a new branch and open a pull request.

**Show:** the `unit-tests` job goes green.

**Say:** "Green tests. In a lot of setups, this merges."

### 2. The preview shows the problem

**Show:** open the preview for this pull request and request a quote.

**Show:** every price on the page reads **$NaN**. The page asks for
`total_cents` and the API now returns `totalCents`.

### 3. The CI agent refuses it

**Show:** the comment on the pull request. Expect a ❌ verdict, with failed
checks naming the renamed fields and the broken page.

**Show:** the pull request is still open and the check is red.

**Show:** production Brewbar is untouched.

**Say:** "Same pipeline, same agent. It caught a break that unit tests missed,
because it tested the running app against the promise we made to our users.
The bad version only ever lived in a throwaway preview."

## Close (1 minute)

**Say:**

- "Every pull request gets a real running copy of the app on Miren in about a
  minute, and it cleans itself up."
- "That makes it practical to let an agent test the deployed app, not just the
  code."
- "The agent makes the call, and the workflow carries it out, so you can see and
  audit every decision."

## If something goes wrong

| Problem | What to do |
|---|---|
| `preview` fails with an authentication error | The cluster's CI binding doesn't allow `pull_request` events. See step 3 of the README setup. |
| Deploy works but the job has no URL | The deploy action reads the URL from `miren deploy` output. Run `miren app versions -a brewbar` and open the preview by hand. |
| Agent says `reject` on the good change | Read its failed checks aloud. That is still a working demo of the gate. Then move on to Act 2. |
| Merge step fails with "not permitted" | `MERGE_TOKEN` is missing or lacks write access to contents and pull requests. |
| Merge works but production doesn't deploy | The merge used the built-in `GITHUB_TOKEN`, which can't trigger other workflows. Check `MERGE_TOKEN` is set. |
| Running long | Skip the live preview click-through in Act 1 step 2. The agent's report shows the same thing. |

## Resetting between runs

- Close any leftover Act 2 pull requests and delete their branches.
- Revert the Act 1 merge on `main` with a normal revert pull request, or
  rehearse in a fork you can throw away.
- Old previews expire on their own after 24 hours.
