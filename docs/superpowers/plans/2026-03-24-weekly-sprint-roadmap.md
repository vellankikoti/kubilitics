# Weekly Sprint Roadmap — Kubilitics v0.3.x

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship 22 features over 8 weekly sprints, closing 78+ competitor gaps identified in Aptakube/Headlamp analysis, using incremental releases (v0.3.0–v0.3.7).

**Architecture:** Cherry-pick proven code from PR #22 (`feat/competitor-gap-closure`) for Weeks 1–4. Build net-new features for Weeks 5–8. Each week is a feature branch → PR → merge → release cycle.

**Tech Stack:** React + TypeScript + Vite (frontend), Go (backend), Monaco Editor, xterm.js, Zustand stores, Tauri 2.0 (desktop)

**Workflow per week:**
1. Create feature branch from `main` (e.g., `feat/week-1-yaml-editor`)
2. Cherry-pick or develop the feature
3. Run: `tsc --noEmit`, `npm run build`, `npm run test`, `madge --circular`
4. Raise PR with test plan
5. Review for security issues and merge conflicts
6. Merge to `main`, test manually, tag release (v0.3.x)

---

## Week 1 — YAML Editor Enhancements (v0.3.0)

**Source:** PR #22 commit `ae34d8d` (P0-3)
**Branch:** `feat/week-1-yaml-editor`
**Risk:** Low — isolated to 2 files, no backend changes

### Task 1: Cherry-pick YAML Editor changes

**Files:**
- Modify: `kubilitics-frontend/src/components/resources/YamlEditorDialog.tsx`

- [ ] **Step 1: Create feature branch**
```bash
git checkout main && git pull
git checkout -b feat/week-1-yaml-editor
```

- [ ] **Step 2: Cherry-pick the YAML editor commit**
```bash
git cherry-pick ae34d8d --no-commit
```

- [ ] **Step 3: Resolve conflicts**
Keep only changes to `YamlEditorDialog.tsx`. Discard any unrelated files:
```bash
git checkout HEAD -- <any-unrelated-files>
```

- [ ] **Step 4: Verify the specific features work**
Checklist:
- Fold All / Unfold All buttons appear in toolbar
- Managed Fields toggle hides/shows `metadata.managedFields`
- Font size S/M/L buttons change Monaco editor font
- Diff view shows side-by-side comparison of initial vs edited YAML
- Dialog stays open on apply failure (doesn't lose edits)

- [ ] **Step 5: Run full verification**
```bash
cd kubilitics-frontend
npx tsc --noEmit
npm run build
npm run test
npx madge --circular --extensions ts,tsx src/
```

- [ ] **Step 6: Commit and push**
```bash
git add -A
git commit -m "feat: YAML editor — fold/unfold, managed fields toggle, font size, diff view"
git push origin feat/week-1-yaml-editor
```

- [ ] **Step 7: Create PR**
```bash
gh pr create --title "feat: YAML Editor Enhancements" --body "Cherry-picked from PR #22 (P0-3)..."
```

- [ ] **Step 8: Merge and tag**
```bash
gh pr merge --squash
git checkout main && git pull
# bump versions, commit, tag v0.3.0
```

---

## Week 2 — Advanced Log Viewer Engine (v0.3.1)

**Source:** PR #22 commits `d090bad` + `6eb79ee` (P0-1)
**Branch:** `feat/week-2-log-viewer`
**Risk:** Medium — touches LogViewer (heavily used), adds new store + hook

### Task 1: Cherry-pick Log Viewer changes

**Files:**
- Modify: `kubilitics-frontend/src/components/resources/LogViewer.tsx`
- Create: `kubilitics-frontend/src/hooks/useLogFilterHistory.ts`
- Create: `kubilitics-frontend/src/stores/logFilterStore.ts`
- Modify: `kubilitics-frontend/src/lib/logParser.ts`

- [ ] **Step 1: Create feature branch**
```bash
git checkout main && git pull
git checkout -b feat/week-2-log-viewer
```

- [ ] **Step 2: Cherry-pick log viewer commits**
```bash
git cherry-pick d090bad --no-commit
git cherry-pick 6eb79ee --no-commit
```

- [ ] **Step 3: Resolve conflicts**
LogViewer.tsx will have significant conflicts with current main. Key features to preserve:
- JSON detection and prettification with syntax highlighting
- Regex filter toggle with error handling
- Context lines around filtered results (grep -A -B mode)
- Inverse filter (exclude mode)
- Filter history with pin/save (useLogFilterHistory hook)
- Persistent filters across navigation (logFilterStore Zustand store)
- Auto-reconnect with exponential backoff
- Alternating line shading
- Hide terminated container logs toggle

- [ ] **Step 4: Verify features**
Checklist:
- Open any pod logs → JSON lines are syntax-highlighted
- Toggle regex mode → enter regex → verify filter works
- Context lines slider → shows N lines around matches
- Inverse filter → excludes matching lines
- Filter history dropdown → shows recent filters
- Navigate away and back → filters persist
- Disconnect WebSocket → reconnecting indicator appears → auto-reconnects
- Toggle alternating shading → rows alternate bg color
- Hide terminated containers toggle works

- [ ] **Step 5: Run full verification**
```bash
npx tsc --noEmit && npm run build && npm run test && npx madge --circular --extensions ts,tsx src/
```

- [ ] **Step 6: Commit, push, PR, merge, tag v0.3.1**

---

## Week 3 — Node Operations (v0.3.2)

**Source:** PR #22 commit `ab6e404` (P0-2 partial — node ops)
**Branch:** `feat/week-3-node-ops`
**Risk:** Medium — backend change (new Go endpoint), requires cluster to test

### Task 1: Cherry-pick Node Operations backend

**Files:**
- Create: `kubilitics-backend/internal/api/rest/node_operations.go`
- Modify: `kubilitics-backend/internal/api/rest/handler.go` (register routes)

- [ ] **Step 1: Create feature branch**
- [ ] **Step 2: Cherry-pick node_operations.go from PR #22**
- [ ] **Step 3: Register routes in handler.go**
Routes:
```
POST /clusters/{clusterId}/resources/nodes/{name}/cordon
POST /clusters/{clusterId}/resources/nodes/{name}/uncordon
POST /clusters/{clusterId}/resources/nodes/{name}/drain
```
- [ ] **Step 4: Build and test backend**
```bash
cd kubilitics-backend && go build ./cmd/server && go test ./... -count=1
```

### Task 2: Cherry-pick Node Operations frontend

**Files:**
- Modify: `kubilitics-frontend/src/pages/NodeDetail.tsx`
- Modify: `kubilitics-frontend/src/pages/Nodes.tsx`

- [ ] **Step 5: Cherry-pick NodeDetail + Nodes page changes**
Features:
- Cordon/Uncordon button on NodeDetail page (calls real API)
- Drain button with progress indicator
- Visual unschedulable indicator on Nodes list

- [ ] **Step 6: Run full verification (backend + frontend)**
- [ ] **Step 7: Commit, push, PR, merge, tag v0.3.2**

---

## Week 4 — Bulk Actions (v0.3.3)

**Source:** PR #22 commit `ab6e404` (P0-2 — bulk actions)
**Branch:** `feat/week-4-bulk-actions`
**Risk:** High — touches 30+ list pages, needs careful integration

### Task 1: Cherry-pick BulkActionToolbar

**Files:**
- Create: `kubilitics-frontend/src/components/list/BulkActionToolbar.tsx`
- Modify: 30+ list pages to add selection checkboxes

- [ ] **Step 1: Create feature branch**
- [ ] **Step 2: Cherry-pick BulkActionToolbar component**
- [ ] **Step 3: Integrate into priority list pages first**
Start with:
1. Pods list — bulk delete
2. Deployments list — bulk restart, scale, delete
3. StatefulSets list — bulk restart, scale
4. DaemonSets list — bulk restart
5. Nodes list — bulk cordon/uncordon
- [ ] **Step 4: Add row selection (checkbox) to ResourceList component**
- [ ] **Step 5: Wire bulk operations to real backend endpoints**
- [ ] **Step 6: Integrate into remaining list pages (20+)**
- [ ] **Step 7: Run full verification**
- [ ] **Step 8: Commit, push, PR, merge, tag v0.3.3**

---

## Week 5 — File Transfer / kubectl cp (v0.3.4)

**Source:** New development (not in PR #22)
**Branch:** `feat/week-5-file-transfer`
**Risk:** Medium — new backend endpoint + new UI component

### Task 1: Backend — File transfer endpoint

**Files:**
- Create: `kubilitics-backend/internal/api/rest/file_transfer.go`

Endpoints:
```
POST /clusters/{clusterId}/resources/{namespace}/{pod}/upload — multipart file → kubectl cp
GET  /clusters/{clusterId}/resources/{namespace}/{pod}/download?path= — kubectl cp → stream
POST /clusters/{clusterId}/resources/{namespace}/{pod}/ls?path= — list directory
```

### Task 2: Frontend — File Transfer UI

**Files:**
- Create: `kubilitics-frontend/src/components/resources/FileTransferDialog.tsx`
- Modify: `kubilitics-frontend/src/pages/PodDetail.tsx` (add button)

Features:
- Upload: drag-and-drop + file picker → POST to backend
- Download: browse container filesystem → click to download
- Directory listing with breadcrumbs
- Progress indicator for large files

---

## Week 6 — Debug Containers (v0.3.5)

**Source:** New development
**Branch:** `feat/week-6-debug-containers`

### Task 1: Backend — Ephemeral container endpoint

**Files:**
- Create: `kubilitics-backend/internal/api/rest/debug_container.go`

Endpoint: `POST /clusters/{clusterId}/resources/{namespace}/{pod}/debug`

### Task 2: Frontend — Debug Container UI

**Files:**
- Create: `kubilitics-frontend/src/components/resources/DebugContainerDialog.tsx`
- Modify: `kubilitics-frontend/src/pages/PodDetail.tsx`

Features:
- Image selector (busybox, alpine, nicolaka/netshoot, custom)
- Target container selector
- Auto-open terminal after container starts
- Node debugging via `kubectl debug node/`

---

## Week 7 — Cluster Colors & Env Badges (v0.3.6)

**Source:** New development
**Branch:** `feat/week-7-cluster-ux`

### Task 1: Cluster color/badge configuration

**Files:**
- Modify: `kubilitics-frontend/src/stores/clusterStore.ts`
- Create: `kubilitics-frontend/src/components/settings/ClusterAppearance.tsx`
- Modify: `kubilitics-frontend/src/components/layout/Header.tsx`

Features:
- Per-cluster color picker (8 preset colors)
- Environment badge (prod/staging/dev/custom)
- Production warning banner (red header stripe)
- Cluster aliases (short names)
- Color-coded cluster selector in header

---

## Week 8 — Search & Filter Power Features (v0.3.7)

**Source:** New development
**Branch:** `feat/week-8-search-power`

### Task 1: Enhanced global search

**Files:**
- Modify: `kubilitics-frontend/src/components/layout/GlobalSearch.tsx`
- Create: `kubilitics-frontend/src/hooks/useSearchHistory.ts`

Features:
- Sidebar search (Cmd+K → full resource search)
- Virtual scrolling for 50K+ namespaces
- ConfigMap/Secret content search
- Namespace switching from search results
- Search history with recent items

---

## Version Release Schedule

| Week | Version | Feature | Competitor Issues Closed |
|------|---------|---------|-------------------------|
| 1 | v0.3.0 | YAML Editor | 5 |
| 2 | v0.3.1 | Log Viewer | 14 |
| 3 | v0.3.2 | Node Operations | 6 |
| 4 | v0.3.3 | Bulk Actions | 8 |
| 5 | v0.3.4 | File Transfer | 3 |
| 6 | v0.3.5 | Debug Containers | 4 |
| 7 | v0.3.6 | Cluster UX | 6 |
| 8 | v0.3.7 | Search Power | 7 |
| **Total** | | **8 features** | **53 issues** |

## Post-Week-8 Backlog (P2/P3)

These remain for future sprints after the core 8 weeks:
- Keyboard shortcut customization
- Desktop window state persistence
- Port forward improvements
- i18n expansion (6 languages)
- CRD ecosystem integrations (Karpenter, Argo, Crossplane)
- Proxy support (HTTP/SOCKS5)
- GPU monitoring
- Split view / multi-panel
- Saved views & workspaces
- PVC file browser
