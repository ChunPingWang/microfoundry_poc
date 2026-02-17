# MicroFoundry — Claude Code Project Rules

## Branch Strategy: Release-Candidate Flow

```
main (stable)
  └── rc (release-candidate, integration branch)
        ├── epic/feature-a  →  PR targets rc
        ├── epic/feature-b  →  PR targets rc
        └── epic/feature-c  →  PR targets rc
```

### Rules

1. **`main`** is the stable release branch. Only `rc` merges into `main` when validated.
2. **`rc`** is the integration branch. All Epic PRs target `rc`, not `main`.
3. **`epic/*`** branches are created from `rc` (never from `main`).
4. When `rc` accumulates a validated set of features, it is merged to `main` and optionally tagged as a release.
5. Never stack PRs by merging one epic branch into another. Each epic branch is independent and based on `rc`.

### Creating a New Epic Branch

```bash
git checkout rc
git pull origin rc
git checkout -b epic/new-feature
```

### Merging Flow

```
epic/feature → PR → rc (merge) → validate → PR → main (merge + tag)
```

---

## Agent Workflow Rules

### Analyzer Agent — Dependency Check (MANDATORY)

Before planning any new Epic, the Analyzer agent MUST:

1. **Check for open PRs** — Run `gh pr list --state open` and identify any unmerged PRs.
2. **Identify dependencies** — Determine if the new Epic relies on code from any open PR (shared files, APIs, models, routes, templates).
3. **If dependencies exist** — STOP and ask the user to merge the dependent PRs into `rc` first. Do NOT proceed with implementation until dependencies are cleared.
4. **Report dependency status** in the Analyzer report:

```markdown
### Dependency Check
| PR | Title | Status | Required? |
|----|-------|--------|-----------|
| #26 | UAA Keycloak | Open | YES — new Epic uses auth middleware |

**Action Required**: Merge PR #26 into `rc` before starting this Epic.
```

5. **New Epic branches MUST be based on `rc`** — never on `main` or another epic branch.

### All Agents — Branch Awareness

- When creating a new branch, always branch from `rc`.
- When the `rc` branch does not exist yet, ask the user to create it first.
- PR base branch must always be `rc` (use `gh pr create --base rc`).

---

## Project Conventions

- **Language**: Go 1.25+
- **CLI Framework**: cobra
- **Config**: viper + YAML (`configs/mf.yaml`)
- **K8s Client**: client-go with multi-cluster ClientManager
- **Admin UI**: Go templates + HTMX + Tailwind CSS (CDN)
- **Templates**: embed.FS with clone pattern (one clone per page)
- **Auth**: OIDC via coreos/go-oidc/v3 + Keycloak
- **Monitoring**: Prometheus + Loki + Grafana + Beyla
- **Settings storage**: K8s ConfigMap/Secret (not file-based)
- **No new dependencies** unless absolutely necessary — prefer stdlib

## Build & Verify

```bash
go build ./...          # Must pass
go vet ./...            # Must pass
go test ./...           # Must pass
```
