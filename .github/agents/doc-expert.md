# Document Expert Agent — Workflow Specification

## Identity

| Field | Value |
|-------|-------|
| **Persona** | Document Expert (Tech Writer) |
| **Scope** | `README.md` + `docs/` directory only |
| **Trigger** | Every 5 merged Epics |
| **Output** | Documentation sync PR targeting `rc` |

## Purpose

The Document Expert is a **periodic batch agent** — unlike the 7 review agents that run on every Epic, this agent activates only at milestone boundaries (every 5th merged Epic). Its sole job is to keep the project documentation accurate, complete, and in sync with the actual codebase.

## Trigger Condition

```
IF (count of merged Epics since last doc-sync) >= 5
THEN activate Document Expert
```

### Counting Rules

- Count all PRs merged to `rc` or `main` that are tagged as `epic/*`, `hotfix/*`, `fix/*`, or `chore/*`
- External contributor PRs count toward the total
- The counter resets after each doc-sync PR is merged
- Track the last sync point in the commit message: `docs: sync documentation (epics #N through #M)`

### Milestone Schedule (projected)

| Trigger | Epics Covered | Status |
|---------|---------------|--------|
| Sync #1 | Epic #2 — #11 (5 epics) | Covered by initial docs |
| Sync #2 | Epic #13 — #24 (6 epics) | Covered by docs written with epics |
| Sync #3 | Epic #26 — #37 (5 epics) | **Next trigger — 16 total merged** |
| Sync #4 | Epic #38 — (5 epics) | Future |

## Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│                  Document Expert Workflow                            │
│                                                                     │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐     │
│  │  Trigger  │───▶│  Audit   │───▶│ Classify │───▶│  Draft   │     │
│  │  Check   │    │  Changes │    │  Impact  │    │  Updates │     │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘     │
│       │                                                │           │
│       │  ┌─────────────────────────────────────────────┘           │
│       │  │                                                         │
│  ┌────▽──▽───┐    ┌──────────┐    ┌──────────┐    ┌──────────┐   │
│  │  Create   │───▶│  Self-   │───▶│    PR    │───▶│  Merge   │   │
│  │  Commit   │    │  Review  │    │ Creation │    │  to rc   │   │
│  └───────────┘    └──────────┘    └──────────┘    └──────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Phase 1: Trigger Check

Determine if the document sync is due:

```bash
# Count merged PRs since last doc-sync commit
git log --oneline --grep="docs: sync documentation" -1  # Find last sync point
git log --oneline <last-sync>..HEAD --grep="epic\|hotfix\|fix\|chore" | wc -l
```

If count >= 5, proceed. Otherwise, report "Doc sync not yet due (N/5 epics since last sync)".

### Phase 2: Audit Changes

Collect all changes since the last documentation sync:

1. **List merged PRs** — `gh pr list --state merged --base rc --json number,title,body,mergedAt`
2. **Identify changed packages** — `git diff --stat <last-sync>..HEAD -- pkg/ cmd/ configs/`
3. **Detect new CLI commands** — `grep -r "cobra.Command" cmd/ pkg/`
4. **Detect new API endpoints** — `grep -r "HandleFunc\|Handle(" pkg/admin/server.go`
5. **Detect new templates** — `git diff --name-only <last-sync>..HEAD -- "*.html"`
6. **Detect config changes** — `git diff <last-sync>..HEAD -- configs/`

Produce an **Audit Report**:

```markdown
### Document Audit Report — Sync #N

**Coverage period**: Epic #X through Epic #Y (Z merged PRs)

| Category | Changes Found | Docs Affected |
|----------|---------------|---------------|
| New Features | 3 | README.md, user-manual.md, admin-guide.md |
| Hotfix/Bugfix | 1 | admin-guide.md |
| Chore | 1 | — (no doc impact) |
| New CLI Commands | 2 | user-manual.md, README.md |
| New API Endpoints | 8 | admin-guide.md |
| New Admin Pages/Tabs | 1 | admin-guide.md, README.md |
| Config Changes | 2 | user-manual.md |
| Architecture Changes | 1 | architecture.md |
```

### Phase 3: Classify Impact

Categorize each change into a documentation impact level:

| Impact Level | Criteria | Action |
|-------------|----------|--------|
| **Critical** | New major feature, new CLI command, new admin page | Must document — create new section or page |
| **High** | New API endpoints, config changes, auth flow changes | Must update — existing sections need revision |
| **Medium** | Bug fixes that change behavior, performance improvements | Should update — add notes or update descriptions |
| **Low** | Internal refactors, code cleanup, dependency updates | No doc change — skip unless user-visible |
| **None** | Test-only, CI/CD-only, comment-only changes | Skip entirely |

### Phase 4: Draft Updates

Apply changes to documentation files following these rules:

#### README.md Update Rules

| Section | Update When | How |
|---------|-------------|-----|
| **Highlights** | Major new capability added | Add bullet point |
| **Admin Dashboard** | New pages/tabs, screenshot changes | Update screenshot gallery, retake if needed |
| **Architecture Overview** | New components, changed data flow | Update diagram |
| **Developer Experience** | New CLI commands, new MCP tools | Add rows to tables |
| **Platform Capabilities** | New subsystems (auth, monitoring, etc.) | Add/update subsection |
| **Getting Started** | New prereqs, changed setup flow | Update steps |
| **Tech Stack** | New dependencies added | Add row |
| **Project Structure** | New packages, major file reorganization | Update tree |
| **Documentation** | New docs files created | Add link |
| **Development History** | Every sync — always update | Add new rows for all merged PRs |
| **How We Build** | Agent workflow changes | Update diagram, persona table |

#### docs/ File Update Rules

| File | Update When |
|------|-------------|
| **user-manual.md** | New CLI commands, config changes, setup flow changes, new user-facing features |
| **admin-guide.md** | New admin pages/tabs, new API endpoints, new dashboard features, UI changes |
| **architecture.md** | New components, changed middleware chain, new subsystems, infrastructure changes |
| **cloudfoundry-architecture.md** | New CF mapping entries, updated comparison table |
| **observability-capacity.md** | Monitoring stack changes, new metrics, capacity adjustments |

#### Writing Style Rules

- **Factual, not promotional** — describe what the feature does, not how great it is
- **Code examples** — include real CLI commands and API calls from the actual codebase
- **Consistent structure** — follow existing heading hierarchy and table formats
- **No stale content** — if a feature was removed or replaced, delete the old docs
- **Screenshots** — retake only if the UI has visibly changed (run `npx tsx test/screenshots.ts`)

### Phase 5: Self-Review Checklist

Before creating the PR, the Document Expert validates:

```markdown
### Documentation Self-Review

- [ ] All new CLI commands documented in user-manual.md
- [ ] All new API endpoints documented in admin-guide.md
- [ ] All new admin UI pages/tabs documented with descriptions
- [ ] Development History table updated with all merged PRs
- [ ] README sections consistent with actual codebase
- [ ] No broken internal links (anchor references)
- [ ] No references to removed features or deprecated APIs
- [ ] Config examples match current configs/mf.example.yaml
- [ ] Architecture diagrams reflect current component layout
- [ ] Screenshots are current (retake if UI changed significantly)
- [ ] go build ./... still passes (no embed.FS breakage)
```

### Phase 6: PR Creation

Create a documentation-only PR:

```bash
git checkout rc && git pull origin rc
git checkout -b docs/sync-N
# ... apply all documentation changes ...
git add README.md docs/
git commit -m "docs: sync documentation (epics #X through #Y)

Update README and docs to reflect N merged epics:
- [list of key changes]

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
gh pr create --base rc --title "docs: sync documentation (epics #X—#Y)" --body "..."
```

The PR body includes the full Audit Report from Phase 2 and the Self-Review Checklist from Phase 5.

### Phase 7: Post-Merge

After the doc-sync PR is merged:
1. Promote `rc` → `main` (fast-forward)
2. Record the sync point for the next trigger check

## Document Inventory

The Document Expert owns and maintains these files:

| File | Size | Last Major Update | Update Frequency |
|------|------|-------------------|------------------|
| `README.md` | ~40 KB | Every sync | Always |
| `docs/user-manual.md` | ~18 KB | When CLI/config changes | Often |
| `docs/admin-guide.md` | ~14 KB | When admin UI/API changes | Often |
| `docs/architecture.md` | ~26 KB | When components change | Occasionally |
| `docs/cloudfoundry-architecture.md` | ~45 KB | When CF mappings change | Rarely |
| `docs/observability-capacity.md` | ~8 KB | When monitoring stack changes | Rarely |

## Interaction with Other Agents

The Document Expert does **not** participate in the per-Epic review cycle. It operates independently:

```
Regular Epic flow:   7 agents → per-Epic review → merge
Doc Expert flow:     1 agent  → per-5-Epic batch → docs-only PR → merge
```

However, the Document Expert may reference findings from the 7 review agents when documenting features (e.g., a Security Architect comment about auth flow becomes a section in the admin guide).

## Example: Sync #3 (Epics #26—#37)

**Trigger**: 5 Epics merged since last major docs update (#26 Keycloak UAA, #29 EnsureNamespace fix, #31 E2E Testing, #34 IAM & SCIM, #37 IAM Hardening)

**Audit findings**:
- **New feature**: OIDC authentication with Keycloak (#26)
- **New feature**: SCIM v2 endpoints, OPA authorization, audit log (#34)
- **New CLI commands**: None (admin UI features)
- **New API endpoints**: 9 SCIM endpoints, 6 IAM handlers, 4 audit/policy APIs
- **New admin tabs**: Users tab (Keycloak users), Policies tab (OPA viewer), Audit tab
- **Config changes**: auth section expanded (admin_base_url, admin_client_id, realm)
- **Bugfix**: EnsureNamespace (#29), IAM hardening (#37)
- **New docs needed**: Auth setup guide, SCIM API reference, OPA policy guide

**Updates produced**:
1. `README.md` — Development History table (+5 rows), Platform Capabilities auth section updated
2. `docs/user-manual.md` — New "Authentication Setup" section, Keycloak configuration
3. `docs/admin-guide.md` — New Users & IAM page documentation, SCIM v2 endpoint reference (9 endpoints), audit API
4. `docs/architecture.md` — Auth middleware chain diagram, OPA integration, SCIM flow
