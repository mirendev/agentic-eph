# Renders .agent/verdict.json as the PR comment. Run with: jq -r -f report.jq
def cell: tostring | gsub("\n"; " ") | gsub("\\|"; "\\|");
def mark: if . == "pass" then "✅" else "❌" end;

(if .decision == "merge" then "## ✅ Agent review: merging" else "## ❌ Agent review: not merging" end),
"",
.summary,
"",
"Tested against preview: \(env.PREVIEW_URL)",
"",
"| Check | Result | What happened |",
"|---|---|---|",
(.checks[] | "| \(.name | cell) | \(.result | mark) | \(.detail | cell) |")
