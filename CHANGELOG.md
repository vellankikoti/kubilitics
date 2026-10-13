# Changelog

All notable changes to Kubilitics will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.4] - 2026-10-11


### Added

- Add APIService, VerticalPodAutoscaler, and DRA caching by @vellankikoti ([71a600f](https://github.com/vellankikoti/kubilitics/commit/71a600fb648a74fa544869ba95231c10dd841420))


### Documentation

- Add standalone CODE_OF_CONDUCT.md, close .gitignore gaps by @vellankikoti ([b1242d6](https://github.com/vellankikoti/kubilitics/commit/b1242d694e91812909685d91b6abbca7eb7175e0))
- Correct KNOWN-ISSUES.md re: ProjectDetailPage Connect fix completeness by @vellankikoti ([455761f](https://github.com/vellankikoti/kubilitics/commit/455761f8db8e98a887ed81bba126b687e0a28ad0))
- Correct stale upstream-sync claim in RELEASE-PROCESS.md by @vellankikoti ([d752b3f](https://github.com/vellankikoti/kubilitics/commit/d752b3feb556db3ea21794b9c0e8b68a66c854c1))
- Add stabilization plan from critical-review pass by @vellankikoti ([eed9d11](https://github.com/vellankikoti/kubilitics/commit/eed9d11b82290b229af6feab4bad674c8780ec7c))
- Record Phase 3 benchmark results and Phase 4 closure by @vellankikoti ([50ce5ea](https://github.com/vellankikoti/kubilitics/commit/50ce5ea41ff4d3843f589ba28f5e4b5ecac78188))
- Record cluster-detection reliability fixes from Headlamp comparison by @vellankikoti ([5263f69](https://github.com/vellankikoti/kubilitics/commit/5263f691d5bc6b748d09a7a5f6f212b712198610))
- Record the unreachable-cluster identity-collision freeze root cause by @vellankikoti ([8bb4ca2](https://github.com/vellankikoti/kubilitics/commit/8bb4ca2e27f09ced26b65d05ee1ead989d37b857))
- Correct mis-filed flaky-test entries, record actual fixes by @vellankikoti ([1012747](https://github.com/vellankikoti/kubilitics/commit/1012747f5ea459489cddcd062f81c38a8874e4d8))
- Close out STABILIZATION-PLAN.md — all 5 phases verified done by @vellankikoti ([e0939fe](https://github.com/vellankikoti/kubilitics/commit/e0939fe3e40b70414461efa02ffc990c2fea9b45))
- Add standalone CODE_OF_CONDUCT.md, close .gitignore gaps by @vellankikoti ([13ea9cc](https://github.com/vellankikoti/kubilitics/commit/13ea9cc0d0e5845945a4d51058ad7fcc349639fa))
- V1.2.4 real-artifact validation report by @vellankikoti ([93a1e99](https://github.com/vellankikoti/kubilitics/commit/93a1e99aa14725263b31b7f3b00e5d8853a90790))


### Fixed

- Changelog workflow reuses one branch instead of one per tag by @vellankikoti ([bd71d91](https://github.com/vellankikoti/kubilitics/commit/bd71d9126078319ca12e493a1e1ddd3c76e96c70))
- Restore missing Apache-2.0 APPENDIX section by @vellankikoti ([1e3e6d7](https://github.com/vellankikoti/kubilitics/commit/1e3e6d72fa77889806f0a3d6af59670654ddfd0d))
- Close review gaps from Batch 1's informer-cache rollout by @vellankikoti ([50f280f](https://github.com/vellankikoti/kubilitics/commit/50f280faba81c24918648ee25bf89d23e2306560))
- Scope resource-topology collection to the resource's own namespace by @vellankikoti ([a421170](https://github.com/vellankikoti/kubilitics/commit/a421170d5ad7d4aeedccd9581d74bff82a5ea303))
- Clamp memory/CPU percent to 100; fix misleading network metric tooltip by @vellankikoti ([e3b6710](https://github.com/vellankikoti/kubilitics/commit/e3b67103ef75210e7e253beb2ecc02fde94afa9e))
- Real match navigation, JSON-mode highlighting, light-mode contrast by @vellankikoti ([40dcfb5](https://github.com/vellankikoti/kubilitics/commit/40dcfb58ff5c9f1d22cda7371c67ca5e02141829))
- Recover automatically from the graph engine's cold start instead of giving up forever by @vellankikoti ([f97338e](https://github.com/vellankikoti/kubilitics/commit/f97338e8bcd5cc936d75d0c8a88e709f5fd4c690))
- Cap unbounded directory listing; stream downloads instead of buffering by @vellankikoti ([b9fbad5](https://github.com/vellankikoti/kubilitics/commit/b9fbad5fd335803f22cb4606f351b70136eeecf8))
- Migrate vitest 2→4 + vite 5→6 to clear 2 critical dev-only CVEs by @vellankikoti ([751f299](https://github.com/vellankikoti/kubilitics/commit/751f299f5733de0be1577faf21eb38876e79c660))
- Close project-page status-badge gap; expand informer-cache coverage by @vellankikoti ([c342ce3](https://github.com/vellankikoti/kubilitics/commit/c342ce3978acf6634c0baaa44d3d3847d46a5260))
- ProjectDetailPage Connect was still a no-op under metadata-fetch failure by @vellankikoti ([33e56da](https://github.com/vellankikoti/kubilitics/commit/33e56da06fcd648b347d35e2fb7bbe583d8df7af))
- Add missing panic recovery to informer cache's background retry loop by @vellankikoti ([cb9ccd9](https://github.com/vellankikoti/kubilitics/commit/cb9ccd9b4ed895be9b6d744a3139e5e58daff740))
- Critical pre-release hardening — panic recovery, lint gate, CRD sync decoupling by @vellankikoti ([ffbc0f3](https://github.com/vellankikoti/kubilitics/commit/ffbc0f3551149174079a5706b7d564e801b8a67f))
- Critical pre-release hardening — wrong-cluster bug, dead subscriber dedup, silent errors by @vellankikoti ([d90a4b9](https://github.com/vellankikoti/kubilitics/commit/d90a4b9df4830504ce241fc296af3502119921c5))
- Namespace filter usability — search box, select-all, URL persistence by @vellankikoti ([3956f8c](https://github.com/vellankikoti/kubilitics/commit/3956f8c660a1ff77d33dda506bb9db9c900b7123))
- Standardize namespace filter UI — extract shared hook, migrate Deployments by @vellankikoti ([91a9581](https://github.com/vellankikoti/kubilitics/commit/91a95814677bbdb20037e889c0921183912f18c3))
- Standardize namespace filter UI across all 27 remaining list pages by @vellankikoti ([a25962e](https://github.com/vellankikoti/kubilitics/commit/a25962e63042228ee76ad5a4025848ef517de500))
- Close Secret-redaction case bypass and add RBAC to Events/OTel routes by @vellankikoti ([c77de17](https://github.com/vellankikoti/kubilitics/commit/c77de17f9a0df84f209295fcfc776b936bf3499b))
- Close remaining Phase 0 RBAC gaps — file browser, scanner, report schedules by @vellankikoti ([2f01011](https://github.com/vellankikoti/kubilitics/commit/2f01011bde6c34aec9ac1fa0e27f07ad94eec8cd))
- Reset restart_count on success; Reconnect escalates to a real process restart by @vellankikoti ([4903db3](https://github.com/vellankikoti/kubilitics/commit/4903db3f23de9d3bb74f79dd4093d4b539328c20))
- Add health-monitor loop for the AI/brain sidecar by @vellankikoti ([a5ac7b6](https://github.com/vellankikoti/kubilitics/commit/a5ac7b64c6c9abc4a566adebaedbe449497ef932))
- Require destructive-action confirmation for node drain; add test coverage by @vellankikoti ([2d9dbd2](https://github.com/vellankikoti/kubilitics/commit/2d9dbd20de82c2cc3a6aba78310000f29c3659d9))
- OIDC nonce replay protection + fix broken SAML ACS flow by @vellankikoti ([91f14e9](https://github.com/vellankikoti/kubilitics/commit/91f14e923ea857d0ab125231a6c3ce415f1351f9))
- Gate VerticalPodAutoscaler on discovery, not just DRA by @vellankikoti ([89fc568](https://github.com/vellankikoti/kubilitics/commit/89fc56831bc49d669773f6fe28c84243a88d22a8))
- Wire emitClusterSwitch into the one actual choke point by @vellankikoti ([dffae8c](https://github.com/vellankikoti/kubilitics/commit/dffae8c37e6e4263c437878adc35a112d8241e3f))
- Populate Connected from the reachability checker, not empty by @vellankikoti ([54f9141](https://github.com/vellankikoti/kubilitics/commit/54f91413c2695a9d1f67ecfeafd50401ee0940f8))
- Refresh discovery state before forwarding a watch event by @vellankikoti ([9a56bbb](https://github.com/vellankikoti/kubilitics/commit/9a56bbb95d7b68eb9164a51a351e2d0c0078230d))
- Seed ServerURL from kubeconfig so unreachable clusters get one stable identity by @vellankikoti ([a9f7393](https://github.com/vellankikoti/kubilitics/commit/a9f7393ae9cff4a25e9a042f6bacd68f4575614f))
- Distinguish checking from unreachable; fix flaky/broken tests found along the way by @vellankikoti ([ae4304c](https://github.com/vellankikoti/kubilitics/commit/ae4304c48838d92ca69b0de055dde1acd55517e4))
- Wire AggregatePods in; fix its backwards edge-direction assumption by @vellankikoti ([3caf04a](https://github.com/vellankikoti/kubilitics/commit/3caf04a23fd37f153551f6df9d440521e6ef05c7))
- Add 15s request-level timeout watchdog by @vellankikoti ([5d5f576](https://github.com/vellankikoti/kubilitics/commit/5d5f5763387f4d77e325c6acc086e8fa32ffa75a))
- Close path-traversal arbitrary read/write in Tauri commands by @vellankikoti ([688dd00](https://github.com/vellankikoti/kubilitics/commit/688dd006bc68481ef90e3c0dfa2a7905b1e47fea))
- Scope shell:allow-execute to no args for both sidecars by @vellankikoti ([518139d](https://github.com/vellankikoti/kubilitics/commit/518139d4da1feba9af5e21cdfedd577879ef9025))
- Guard divide-by-zero in ResourceMetricsChart trend calc by @vellankikoti ([ebcc2e7](https://github.com/vellankikoti/kubilitics/commit/ebcc2e7327fc6ee0cdfb492b735f09a2c7c60105))
- Surface fetch errors instead of silent empty/zero states by @vellankikoti ([5722df3](https://github.com/vellankikoti/kubilitics/commit/5722df35b5362661b9e9dc583df9d65e43d25e9d))
- Gate Sentry init behind analytics consent by @vellankikoti ([c923bf3](https://github.com/vellankikoti/kubilitics/commit/c923bf3472677d9cd82f809b589648419916457c))
- Add panic recovery to ~40 unrecovered production goroutines by @vellankikoti ([78123d3](https://github.com/vellankikoti/kubilitics/commit/78123d3ec070e129a479834c8c29d41bd1d6b9ee))
- Restore missing Apache-2.0 APPENDIX section by @vellankikoti ([89837ce](https://github.com/vellankikoti/kubilitics/commit/89837ce4220ca255389fde24c6d92c83b752ec1a))
- Remove dead shortestPath/buildFailurePath, kept alive only by tests by @vellankikoti ([38823fc](https://github.com/vellankikoti/kubilitics/commit/38823fc0785100725f43acf838fae6d61ef9e54a))
- Eliminate contention-driven flakiness in full-suite vitest runs by @vellankikoti ([2adfdd1](https://github.com/vellankikoti/kubilitics/commit/2adfdd17f34a4fdc2ac81e655b7cd8bd28121e49))
- P0 — app crashed on every launch via brain health-monitor by @vellankikoti ([2fe793d](https://github.com/vellankikoti/kubilitics/commit/2fe793de805832a85bc85349c7a94bb9e3a1a1f2))


### Misc

- Backfill missing v1.2.0, v1.2.2, v1.2.3 sections by @vellankikoti ([22f3cea](https://github.com/vellankikoti/kubilitics/commit/22f3ceac4c9b9ab09d62b48066b8e756e2103a05))
- Update go.work.sum checksums picked up by test runs by @vellankikoti ([3dbdd08](https://github.com/vellankikoti/kubilitics/commit/3dbdd0873c756745acc6f66919478ae418f9a778))
- Delete 12 confirmed-dead, unrouted page components by @vellankikoti ([3a3ad7f](https://github.com/vellankikoti/kubilitics/commit/3a3ad7f523e9eece4e886bcdd866ced8e5d8d504))
- Regenerate capabilities schema after args:false change by @vellankikoti ([bfe5ef9](https://github.com/vellankikoti/kubilitics/commit/bfe5ef90acb193fe5929b9a790bb91b9c736c395))
- Bump version to 1.2.4 by @vellankikoti ([f4e2a05](https://github.com/vellankikoti/kubilitics/commit/f4e2a05f2ed748b616bed615b08bcbff98410363))


### Performance

- Route cluster-summary, fleet-overview, and consumers through the informer cache by @vellankikoti ([8b0031f](https://github.com/vellankikoti/kubilitics/commit/8b0031f08530364f2ae83a37cb79e79805ff80e4))
- Add namespace indexer + incremental pod-metrics to the informer cache (Batch 2, part 1) by @vellankikoti ([612fd69](https://github.com/vellankikoti/kubilitics/commit/612fd69136507e1df3a7f686e871a7d97d62dcd6))
- Shard OverviewCache's lock per cluster (Batch 2, part 2) by @vellankikoti ([df5ff88](https://github.com/vellankikoti/kubilitics/commit/df5ff889d6e481ca73743d603eafc2b54f8d3538))
- Index-based topology matchers + parallel node-stats fetch (Batch 2, part 3) by @vellankikoti ([33d70b0](https://github.com/vellankikoti/kubilitics/commit/33d70b0b13766895b3d0b8cd6fb48e6bdd92281b))
- Memoize resizable-table context + debounce persistence; add select() passthrough (Batch 3, part 1) by @vellankikoti ([5a77cdc](https://github.com/vellankikoti/kubilitics/commit/5a77cdc97b5975d80796b93eb41fbcd8bb99c150))
- Gate ELK relayout on an actual shape change (Batch 3, part 2) by @vellankikoti ([e512a4e](https://github.com/vellankikoti/kubilitics/commit/e512a4ef765c8db2acf68326882cb97d98e02891))
- Move topology layout off the main thread into a Web Worker (Batch 3, part 3) by @vellankikoti ([e057774](https://github.com/vellankikoti/kubilitics/commit/e0577740f929c82e2e296610a96b7e704d5963ab))
- Route topology collection through the informer cache (closes Batch 1 gap) by @vellankikoti ([2feef9f](https://github.com/vellankikoti/kubilitics/commit/2feef9f303afc061e56a54da9169fcb2ab9f851f))
- Route the V1 engine (main Topology tab) through the informer cache by @vellankikoti ([34b7a28](https://github.com/vellankikoti/kubilitics/commit/34b7a28fa198e2a4dd64ef12ad5e682f62499c85))
- Debounce namespace-keyed side queries on Deployments/StatefulSets by @vellankikoti ([9cab509](https://github.com/vellankikoti/kubilitics/commit/9cab509b7f4c78d450c242e8f11b9d5e05917bd8))
- O(1) pod/edge lookups, one shared BFS instead of one-per-node by @vellankikoti ([ebb5740](https://github.com/vellankikoti/kubilitics/commit/ebb5740eea0544fdfb2acd9157e85a4907285bff))


### Tests

- Real coverage for the generic resource mutation path by @vellankikoti ([3012e83](https://github.com/vellankikoti/kubilitics/commit/3012e83b3cf6039f579f054f2524ab09f7fded49))
- Cover Login's MFA branch — zero coverage before, auth-bypass-risk path by @vellankikoti ([cd2a99d](https://github.com/vellankikoti/kubilitics/commit/cd2a99d9df0a3035e4bdf4a9ba5f0332d245600f))
- Cover the 3 highest-fan-out shared list-page hooks; fix a real crash bug found along the way by @vellankikoti ([e1782d9](https://github.com/vellankikoti/kubilitics/commit/e1782d9bed8b131b16d99127170c4d902bc93f5d))
- Add real destructive-action test — ConfigMap delete to completion by @vellankikoti ([c99b664](https://github.com/vellankikoti/kubilitics/commit/c99b6641b48aaa9841af3476fbbda3bbf1873298))
- Real coverage for CronJob trigger and Job retry handlers by @vellankikoti ([c6d71c3](https://github.com/vellankikoti/kubilitics/commit/c6d71c3628459b8a082614b4a0a2b27c0737b5b3))
- Real coverage for pod exec, shell, PTY stream, port-forward by @vellankikoti ([c8b4b5f](https://github.com/vellankikoti/kubilitics/commit/c8b4b5fb2ad5e96cd40ff2f0b0e221dec2e3c867))
- Cover Pods/Deployments delete, bulk-delete, and scale flows by @vellankikoti ([bbfc682](https://github.com/vellankikoti/kubilitics/commit/bbfc68268db0d82d81ed68521ba635fb338d9b0f))
- Fix flaky hangingSource race under -race, stress-verified by @vellankikoti ([b5d4761](https://github.com/vellankikoti/kubilitics/commit/b5d4761581a380421bd0561c009fbd02d9a55747))
- Fix mock pointing at the wrong module specifier by @vellankikoti ([cc789ec](https://github.com/vellankikoti/kubilitics/commit/cc789ec207ef494eac6ff6003aaf1113ab007637))
- Close remaining resources.go mutation-path gaps by @vellankikoti ([4833ecb](https://github.com/vellankikoti/kubilitics/commit/4833ecb5c323b53148a741a56f3e6ec789a2c3eb))

## [1.2.3] - 2026-10-06


### Documentation

- UI reliability audit findings and release plan (Oct 2026) by @vellankikoti ([5cda11c](https://github.com/vellankikoti/kubilitics/commit/5cda11ce09394248e7c9133c93d93bde76751057))
- Round 2 audit findings + grounded Headlamp comparison by @vellankikoti ([9bad44f](https://github.com/vellankikoti/kubilitics/commit/9bad44f2b59e41575ce3bcbc53078782052de929))
- Issue-count accounting toward the 50-per-release bar (46/50) by @vellankikoti ([1eb4d18](https://github.com/vellankikoti/kubilitics/commit/1eb4d18815eacfc3da77caea70c64f07117e2f33))
- 50-issue floor reached for this release by @vellankikoti ([caa9ca9](https://github.com/vellankikoti/kubilitics/commit/caa9ca9f1492b9e39ca4f798c2f4b03de11b59bb))
- V1.2.3 real-artifact validation report by @vellankikoti ([887b40f](https://github.com/vellankikoti/kubilitics/commit/887b40f1689ce884e497ea57e0f261a259d00892))


### Fixed

- Surface silent data-fetch failures instead of rendering 0 by @vellankikoti ([e3109db](https://github.com/vellankikoti/kubilitics/commit/e3109db44bfa1ad2a2d41a19d49c499c2c36f09b))
- Harden array-spread sites against non-array API data ("X is not iterable") by @vellankikoti ([88f755b](https://github.com/vellankikoti/kubilitics/commit/88f755bd8b4c13e96878ad0edc1d4dc3753809c8))
- Batch 2 — dashboard health score, simulation/health-issue crashes, 4 more silent-failure hooks by @vellankikoti ([bb1b60b](https://github.com/vellankikoti/kubilitics/commit/bb1b60b79aa2b80b9e8f2bd3d658cedd23d28266))
- Panic recovery on all 18 unprotected goroutines + silent-failure fixes by @vellankikoti ([68dec4a](https://github.com/vellankikoti/kubilitics/commit/68dec4a6bc90118c66df3e5f7d154fd572cf494d))
- Batch 3 — 4 Dashboard widgets silently showed 0/stable on fetch failure by @vellankikoti ([5382875](https://github.com/vellankikoti/kubilitics/commit/53828751890c844133907d426326da1ebc19eed7))
- Round 2 batch 1 — pod status, metrics, project dialogs by @vellankikoti ([6cc8458](https://github.com/vellankikoti/kubilitics/commit/6cc84585fd2861791501d666365c08dd41965317))
- Round 2 batch 2 — 8 more goroutine panic sites + audit-event logging by @vellankikoti ([4784894](https://github.com/vellankikoti/kubilitics/commit/4784894d8cd91f26658cad549c9ea9d392cb7f7e))
- Harden clusterPresenceStore against non-array snapshot fields by @vellankikoti ([12a4860](https://github.com/vellankikoti/kubilitics/commit/12a4860e1327b1d1ceb94291e618bf7b177ec7c8))
- Round 2 batch 3 — scanner panic recovery + 3 silent-failure fixes by @vellankikoti ([3238253](https://github.com/vellankikoti/kubilitics/commit/3238253ffb2b5f2cd6e4e99795aea902f698b86d))
- Goroutine + HTTP connection leaks on client disconnect / retry by @vellankikoti ([45da7de](https://github.com/vellankikoti/kubilitics/commit/45da7dedbf1e4166702bcf44062e1e7e05bdb060))
- Scope keepPreviousData to the same cluster (cluster-switch race) by @vellankikoti ([d571816](https://github.com/vellankikoti/kubilitics/commit/d571816469ebde7ac566b3a3e5235ea01d1c5389))
- Bound workloads.go's live-API pods fallback to 5000 by @vellankikoti ([1359d62](https://github.com/vellankikoti/kubilitics/commit/1359d6256f1bdc71f8d30100f9ed1e5796c1af37))
- Suppress staticcheck SA1019 for intentional corev1.Endpoints usage by @vellankikoti ([7a441f3](https://github.com/vellankikoti/kubilitics/commit/7a441f301225ca1b3c4c3c521fc9bb8bb0fa1f8a))

## [1.2.2] - 2026-10-05


### Added

- Phase 1 complete — tool catalog with 127 tools audited by @vellankikoti ([5b5b5c0](https://github.com/vellankikoti/kubilitics/commit/5b5b5c01da0d3081ed863ed4285fc498d8ca1725))
- Certification engine — grade all 157 tools, gate destructive execution by @vellankikoti ([fe5673d](https://github.com/vellankikoti/kubilitics/commit/fe5673d034fc1998f66b29df5dc4e48133bab7d8))
- Execution layer hardening — idempotency, timeouts, envelope, 92% coverage by @vellankikoti ([3950ee1](https://github.com/vellankikoti/kubilitics/commit/3950ee1a8f40ef4d8bc740fc109820a37d89b57d))
- AI guardrails — redaction, injection scanner, tool call budget by @vellankikoti ([9e782e5](https://github.com/vellankikoti/kubilitics/commit/9e782e552acc34a8dec2b1cf06c40e0a0bd32dd2))
- Observability endpoints, Prometheus metrics, dynamic guardrail config by @vellankikoti ([12e140d](https://github.com/vellankikoti/kubilitics/commit/12e140de1db3343cd3576938cde2dd5b48d9e9ac))
- Hardening benchmark suite — 20 benchmarks across all safety layers by @vellankikoti ([c3069a8](https://github.com/vellankikoti/kubilitics/commit/c3069a882f6c27c47100cae79ac10f6f18d06ee0))
- 157-tool MCP safety stack — cert gate, guardrails, observability, benchmarks by @vellankikoti ([24d8b2d](https://github.com/vellankikoti/kubilitics/commit/24d8b2d763c4869f1dd161532aa4e0321c47b92b))
- Add lazy cluster and engine lifecycle management by @vellankikoti ([b37c373](https://github.com/vellankikoti/kubilitics/commit/b37c3739662b1c350381a55b7b3b53daadfd24ae))


### Documentation

- Promote desktop app to primary install path by @vellankikoti ([573ae38](https://github.com/vellankikoti/kubilitics/commit/573ae383b0229b444eb24fd2780aa02e250902fe))
- Fix Windows install + add verified AI tools cheatsheet by @vellankikoti ([104ac35](https://github.com/vellankikoti/kubilitics/commit/104ac35b546f592d3fff4366e6fcdbcf64e54dc7))
- Color logo, SVG architecture diagram, 8-screen app showcase by @vellankikoti ([a61de75](https://github.com/vellankikoti/kubilitics/commit/a61de759cd950c43a156e7b51ed491da11f400a9))
- Fix app-screens link to render via htmlpreview.github.io by @vellankikoti ([972c283](https://github.com/vellankikoti/kubilitics/commit/972c283b9acbf78a177218c8d650255483329f41))
- Replace fake mockups with 8 real app screenshots by @vellankikoti ([70e33cd](https://github.com/vellankikoti/kubilitics/commit/70e33cd82044f4af5260dba9e55c5438a76fe2a9))
- 157-tool MCP reference with validation status and example usage by @vellankikoti ([a8d10be](https://github.com/vellankikoti/kubilitics/commit/a8d10be9ff78ee6ee3e3d7caf5ec6902e431ea6f))
- Add vision, architecture diagrams, and per-tool user query examples by @vellankikoti ([3d9fca1](https://github.com/vellankikoti/kubilitics/commit/3d9fca1ec4a62746ce52fefb7fa99fb43a2d1973))
- Add critical gap-analysis report for terminal feature by @vellankikoti ([5fe238b](https://github.com/vellankikoti/kubilitics/commit/5fe238bb5cc41a6ec450229bf4772991826eff0b))
- Add cluster-terminal rebuild spec by @vellankikoti ([50e8e62](https://github.com/vellankikoti/kubilitics/commit/50e8e62dc1332e4aff540b1004b020c36fef90d3))
- Record commit SHA in v1.2.1 release report by @vellankikoti ([5e22555](https://github.com/vellankikoti/kubilitics/commit/5e2255506133b777a3b64de71ee0ba867abb899d))
- R2 — 10K topology correctness & VALID-07 re-verification by @vellankikoti ([a996080](https://github.com/vellankikoti/kubilitics/commit/a99608074c67cd58e05c8f824347c63169b04e44))
- Record commit SHA in v1.2.2 release report by @vellankikoti ([029fd50](https://github.com/vellankikoti/kubilitics/commit/029fd506cdca16198d969c11205c2e007bfdcd9e))
- R3 — 10K lifecycle/informer stability re-verification by @vellankikoti ([21eefe8](https://github.com/vellankikoti/kubilitics/commit/21eefe8b08be452f7ed7a7e4df431cb28306ff1a))
- Record commit SHA in v1.2.3 release report by @vellankikoti ([08c8a9b](https://github.com/vellankikoti/kubilitics/commit/08c8a9be5ab244a596e28b5369ffbb40a2db226f))
- Record commit SHA in v1.3.0 release report by @vellankikoti ([706a4fa](https://github.com/vellankikoti/kubilitics/commit/706a4fa7ecba2f906547bba4e963b971aa493999))
- V1.3.0 acceptance cycle — STABLE LOCAL RELEASE by @vellankikoti ([8205f95](https://github.com/vellankikoti/kubilitics/commit/8205f95494ce5c3b9ef3ad7059914f7c9dba3651))
- Investigation and implementation records for v1.2.1 by @vellankikoti ([63221dd](https://github.com/vellankikoti/kubilitics/commit/63221dd2f39d2f644530c6ac35e94f3aad64fc89))
- V1.2.1 consolidated release report by @vellankikoti ([f51fa13](https://github.com/vellankikoti/kubilitics/commit/f51fa130397f7e18cac8376f687f2a537551c369))


### Fixed

- List_problems rejects 'all' filter and stops stealing 'list pods' by @vellankikoti ([1ddfecb](https://github.com/vellankikoti/kubilitics/commit/1ddfecba859759a7048f053eb8885021e4fcbd82))
- Pass GITHUB_TOKEN to git-cliff so it can fetch commit metadata by @vellankikoti ([93604d2](https://github.com/vellankikoti/kubilitics/commit/93604d2808c417f983de3866e2852063a4716aeb))
- Suppress no-control-regex on ANSI strip regex in logParser by @vellankikoti ([117525f](https://github.com/vellankikoti/kubilitics/commit/117525fc0c4ce6ba7f4b7dd1d8e4887fa08e217c))
- Migrate golangci-lint to v2.12.2 + v2 config schema by @vellankikoti ([340ed6b](https://github.com/vellankikoti/kubilitics/commit/340ed6b206b112a24919f291e3f9c62a9783c671))
- Remove gosimple linter (merged into staticcheck in golangci-lint v2) by @vellankikoti ([b2c459c](https://github.com/vellankikoti/kubilitics/commit/b2c459c5ceefa527c3f7a3e861dd7a8a67b7e87c))
- Scope staticcheck to SA*+S1* only — exclude new QF*/ST* in v2 by @vellankikoti ([f8da17e](https://github.com/vellankikoti/kubilitics/commit/f8da17e7b9deb4bd15f86ee8910d995fae19e4f6))
- Budget config, fanOut recovery, load-test correctness by @vellankikoti ([59dd5a5](https://github.com/vellankikoti/kubilitics/commit/59dd5a58f8c4051033c92ca4fa34a72f013352ca))
- Resolve 179 GitHub Security alerts — CVE upgrades + code fixes by @vellankikoti ([13c6e82](https://github.com/vellankikoti/kubilitics/commit/13c6e8270198473358b1d35c385cbd812a67e556))
- Suppress 4 remaining semgrep responsewriter alerts in brain/server.go by @vellankikoti ([f5a113b](https://github.com/vellankikoti/kubilitics/commit/f5a113b1129e533d6b1b13485baf1c4ef964e426))
- Permanently fix all 89 scanner findings — code changes, not suppressions by @vellankikoti ([a5ee040](https://github.com/vellankikoti/kubilitics/commit/a5ee040c5fa34e2759fef5243feb52d48359c306))
- Surface missing default StorageClass instead of a silent stuck pod by @vellankikoti ([5280be5](https://github.com/vellankikoti/kubilitics/commit/5280be54137e7f84c3e809dc7f0bc8d592598b2d))
- Auto-fallback to emptyDir when no default StorageClass exists by @vellankikoti ([8612cad](https://github.com/vellankikoti/kubilitics/commit/8612cada3cffb62c8ea62cfc2de861115e5118f2))
- Optimize paginated cache conversion for 10K-scale concurrency by @vellankikoti ([d28a1d9](https://github.com/vellankikoti/kubilitics/commit/d28a1d9a5be796327a2814c23726113d9b9a687f))
- Eliminate N+1 fan-out and bound multi-cluster concurrency by @vellankikoti ([ffeb650](https://github.com/vellankikoti/kubilitics/commit/ffeb650d90ebbccd4de2555c5fa18e7750ca0008))
- Correct storage N+1, bound node counts, fix V2 force-refresh by @vellankikoti ([eb8f775](https://github.com/vellankikoti/kubilitics/commit/eb8f775d8f3b7c2a46cec2161bd973af74304285))
- Bound blocking I/O, fix cross-cluster WebSocket isolation by @vellankikoti ([0a7761a](https://github.com/vellankikoti/kubilitics/commit/0a7761a09f2565373dcaadd7e7153e196890a444))
- Cluster connect flow, resource quantity parsing, health display by @vellankikoti ([b049c5a](https://github.com/vellankikoti/kubilitics/commit/b049c5a0886ca0757c6a5b4c3ca8f6406d62ae13))
- Pin cosign-installer to v3 (v4 does not exist) by @vellankikoti ([a1cff6b](https://github.com/vellankikoti/kubilitics/commit/a1cff6b967b1bcd3e5267f74694758773d60d154))


### Misc

- Bump all GitHub Actions to latest stable versions by @vellankikoti ([c7d4f83](https://github.com/vellankikoti/kubilitics/commit/c7d4f83a1e9ac43cfdc35cb64dea5ac17936ae8f))
- Ignore certify-tools binary by @vellankikoti ([2e06855](https://github.com/vellankikoti/kubilitics/commit/2e06855a02f53b787afe6c0cc7c740fb8e2f3f23))


### Tests

- Live tool validation harness — 148/148 tools PASS by @vellankikoti ([6a2ddd2](https://github.com/vellankikoti/kubilitics/commit/6a2ddd2e1d915e3ca92d090452b5fe7dfb85b606))
- Tool test harness — contracts, chaos, load, e2e + observation coverage boost by @vellankikoti ([7840344](https://github.com/vellankikoti/kubilitics/commit/78403444d34aa81c462132c45fd0098486ef9ac1))

## [1.2.0] - 2026-05-22


### Documentation

- Record Session 2 root cause — allow-unsigned-executable-memory blocked on macOS 26 by @vellankikoti ([48106f3](https://github.com/vellankikoti/kubilitics/commit/48106f34f6077b84197dcc96be843adb772fec2d))


### Fixed

- Harden macOS signing pipeline — restore inside-out codesign, fix CLI version by @vellankikoti ([36211ec](https://github.com/vellankikoti/kubilitics/commit/36211ec91c2b93581ad557857ac79aff5324f9ab))
- Rewrite Carbon.framework → HIToolbox for macOS 26 launch by @vellankikoti ([46a4692](https://github.com/vellankikoti/kubilitics/commit/46a4692bfc056aeb1e9b3ad196099b84508f1fc9))
- Carbon→HIToolbox rewrite for macOS 26 (Tahoe) compatibility by @vellankikoti ([2e3667f](https://github.com/vellankikoti/kubilitics/commit/2e3667fb30d8dc8c65bd9c648b20ed75a3126db6))
- Remove allow-unsigned-executable-memory — blocked on macOS 26 by @vellankikoti ([9169303](https://github.com/vellankikoti/kubilitics/commit/9169303ff4c5da2b5a7f0ed99cfeb91412e970fa))
- Remove allow-unsigned-executable-memory — blocked on macOS 26 by @vellankikoti ([b91d62a](https://github.com/vellankikoti/kubilitics/commit/b91d62ac8efa3f631807afd759918b83d159c914))
- Remove keychain-access-groups — blocks macOS 26 launch by @vellankikoti ([f9195f1](https://github.com/vellankikoti/kubilitics/commit/f9195f1e413c52581adc7551bd1d982157327640))
- Correct resource count + remove macOS keychain ACL prompt by @vellankikoti ([f2f961d](https://github.com/vellankikoti/kubilitics/commit/f2f961d610abe06278844d56c9fe2f6ad0209d8c))

## [1.4.0] - 2026-05-21


### Added

- Add OTel spans to AI execution paths by @vellankikoti ([7cf9edb](https://github.com/vellankikoti/kubilitics/commit/7cf9edb7aa48bac1620d7d7b4c7a83e5451af49c))
- PostgreSQL + Redis config plumbing and schema verification by @vellankikoti ([d83cabc](https://github.com/vellankikoti/kubilitics/commit/d83cabc51e84874a7416f3de39c8a534d19bdbf1))
- Execution plane classification — thinking vs execution metrics by @vellankikoti ([94a5daf](https://github.com/vellankikoti/kubilitics/commit/94a5daf686d8f080b1bf80965fea86bd1e6b4851))
- Token counts + deployment health status in AI responses by @vellankikoti ([6e113af](https://github.com/vellankikoti/kubilitics/commit/6e113af8fdec5bf212e2f3252c43626471d82424))
- Persist past sessions — History dropdown in ChatHeader by @vellankikoti ([4b6aa0e](https://github.com/vellankikoti/kubilitics/commit/4b6aa0e5ac39edaacaf7f029edd13f0348ffef7c))
- Headlamp-style AddClusterDialog — detected contexts as primary path by @vellankikoti ([5fe69d4](https://github.com/vellankikoti/kubilitics/commit/5fe69d47f0c8f95cb9114f1721cdf7549830a839))
- Two-pane cluster onboarding — list left, add-cluster right by @vellankikoti ([db6835b](https://github.com/vellankikoti/kubilitics/commit/db6835bf14ffaf5c82c588c83812666f2561a717))
- Auto-hotwire active LLM profile on brain start by @vellankikoti ([7b16403](https://github.com/vellankikoti/kubilitics/commit/7b164039e1afce388f0e425e2994ccc7775f90fd))
- Prominent API key error banner above input by @vellankikoti ([4731ceb](https://github.com/vellankikoti/kubilitics/commit/4731cebdd2014afa1192e7797a0ee96d5db833ac))


### Changed

- Extract k8siface narrow interfaces to reduce coupling by @vellankikoti ([338b05c](https://github.com/vellankikoti/kubilitics/commit/338b05c78a141560cf7a1a0ae23c30397865ab96))


### Documentation

- CHANGELOG + AI-CAPABILITIES updated for release by @vellankikoti ([7bdc0a7](https://github.com/vellankikoti/kubilitics/commit/7bdc0a72ab9e30265cc86356ff08fde2481a704a))


### Fixed

- Add causal_chains migration + wire EnsureTables on startup by @vellankikoti ([2563f29](https://github.com/vellankikoti/kubilitics/commit/2563f29845ae5b79c27bcb93bb353b758a28b37a))
- Dynamic brain gRPC port — fall back when 50061 is occupied by @vellankikoti ([0f6260c](https://github.com/vellankikoti/kubilitics/commit/0f6260c9ba1c747d01aaf9fedc90ffc64fa8f866))
- Silence dead_code warning on save_ai_config_inner by @vellankikoti ([6f3292d](https://github.com/vellankikoti/kubilitics/commit/6f3292d08c8dc051da1267238529d80cfd98d5fc))
- Permanently suppress transient brain-unreachable errors from UI by @vellankikoti ([4be7b13](https://github.com/vellankikoti/kubilitics/commit/4be7b133dc0cbb7620a24242155c42f8997e29f4))
- Commit headers before Flush to prevent streaming response corruption by @vellankikoti ([e622152](https://github.com/vellankikoti/kubilitics/commit/e622152a7f789edbf155987da654010713dc65d3))
- Plug WriteHeader forward path + add regression tests by @vellankikoti ([73ac48f](https://github.com/vellankikoti/kubilitics/commit/73ac48f71537c06fad27b3dd10835b96b0c4b8f7))
- Brain auto-discovers backend port after Tauri health-monitor restart by @vellankikoti ([d26faee](https://github.com/vellankikoti/kubilitics/commit/d26faee50075d949cf8cf88c697db9a85fb1f160))
- Correct namespace labels, token counts, and text selection by @vellankikoti ([fc26b1b](https://github.com/vellankikoti/kubilitics/commit/fc26b1bce684d4c4526c704245c8481ae3395f61))
- Deployment status + namespace label root causes by @vellankikoti ([16e5ab4](https://github.com/vellankikoti/kubilitics/commit/16e5ab4eb821135de8be00220ba49efa7d4553f8))
- Namespace label reads from data + header offset fixes blue line by @vellankikoti ([7c2ed8f](https://github.com/vellankikoti/kubilitics/commit/7c2ed8f24a9356f0ab342bd07e905d4cdb91ae23))
- Merge duplicate style props — restores drag-resize width by @vellankikoti ([d2d85d7](https://github.com/vellankikoti/kubilitics/commit/d2d85d77cd7e5757132747968676d92ec0a6de89))
- Eliminate Unknown status for all K8s resource kinds by @vellankikoti ([af0eb1d](https://github.com/vellankikoti/kubilitics/commit/af0eb1da8c00076c9cdd9b86506acb8b0a4f1152))
- Route 'list services' to list_resources, not observe_services_by_filter by @vellankikoti ([28e6333](https://github.com/vellankikoti/kubilitics/commit/28e63333f12a9ab5fb4d85e1581be8f4faea74c7))
- Stop LLM completer hallucinating namespace in all-namespace results by @vellankikoti ([8b9ea50](https://github.com/vellankikoti/kubilitics/commit/8b9ea50b81128618afde2276e19227dadde243af))
- Route describe/inspect queries to inspect_<kind> tools, not list_resources by @vellankikoti ([4d4fc26](https://github.com/vellankikoti/kubilitics/commit/4d4fc261e7a028009f91e33d8c7edf8264117469))
- Remove action_* tools from LLM registry — use execution tools instead by @vellankikoti ([1a048c2](https://github.com/vellankikoti/kubilitics/commit/1a048c2a02a3a1e63ef4b1d0f84268c623d05254))
- Observe_services_by_filter adds summary field so LLM distinguishes healthy vs missing by @vellankikoti ([92bb6f7](https://github.com/vellankikoti/kubilitics/commit/92bb6f76a57f33391eac960a5dd61fba72a7ae28))
- Replace raw-data dumps with compact findings in Tier-1 analysis tools by @vellankikoti ([4668562](https://github.com/vellankikoti/kubilitics/commit/4668562c742b97d6d2e6833c9848d324923a5c4e))
- Add 'explain' and other query patterns to inspect routing; forbid get_resource for user queries by @vellankikoti ([d911317](https://github.com/vellankikoti/kubilitics/commit/d9113179ab4c224c6de823729e09eb8a53ee26a6))
- Plan_* tools return compact findings instead of raw K8s JSON dumps by @vellankikoti ([30e4277](https://github.com/vellankikoti/kubilitics/commit/30e42773958e78b309a0afb57fdd8acc98619d35))
- Narrate_* tools return compact summaries instead of raw K8s JSON dumps by @vellankikoti ([860855b](https://github.com/vellankikoti/kubilitics/commit/860855b22670c0ddda6aef14e360f7774315fe20))
- Recommend_*/security_*/cost_*/automation_* replace raw K8s dumps with compact findings by @vellankikoti ([0f5e9f6](https://github.com/vellankikoti/kubilitics/commit/0f5e9f63b61a239cb4e2ee4eca49ded8a0166c19))
- Data_source field + execution tools rewired to REST by @vellankikoti ([a1b5838](https://github.com/vellankikoti/kubilitics/commit/a1b5838661396f344837efff825f41ae50030a9c))
- 5 production-readiness fixes from systematic audit by @vellankikoti ([0ccd659](https://github.com/vellankikoti/kubilitics/commit/0ccd6595994d73b531d2b2f37d7e8ce563d48703))
- Cluster picker — back button + auto-open add dialog from nav by @vellankikoti ([74e18e5](https://github.com/vellankikoti/kubilitics/commit/74e18e5e409a3daa4e28017d7e4d218bed980b4a))
- Remove redundant overlay — picker page IS the add-cluster screen by @vellankikoti ([462daf0](https://github.com/vellankikoti/kubilitics/commit/462daf0996257c8932a8b1f0cd7858d678bb1236))
- Cluster picker — equal panes, bottom action bar, no top back button by @vellankikoti ([0724ce1](https://github.com/vellankikoti/kubilitics/commit/0724ce11547705744335ed02bdb93e270088cf77))
- Logo size, prominent tab switcher, rename Paste KubeConfig by @vellankikoti ([40d0157](https://github.com/vellankikoti/kubilitics/commit/40d0157b4fbbab6c8ddae35ea2edf9b4746026fb))
- Action bar buttons — red Cancel, blue Add Cluster, equal size by @vellankikoti ([42b54e3](https://github.com/vellankikoti/kubilitics/commit/42b54e394809ae27fe77f1546c31321700ff5ac9))
- Add Cluster always enabled on upload tab — click opens file picker by @vellankikoti ([12c0b1b](https://github.com/vellankikoti/kubilitics/commit/12c0b1b7e89a37053bf68e40328a7bc8bedd1396))
- Cancel button — outline variant using theme border color by @vellankikoti ([036ffd2](https://github.com/vellankikoti/kubilitics/commit/036ffd20057d2fc71e9f59d893b1d37c63970be5))
- Consistent multi-container tab switching and new sessions by @vellankikoti ([72eb62b](https://github.com/vellankikoti/kubilitics/commit/72eb62be338755dceb3dfadb2a10da0ebac473fe))
- Preserve history across container switches; fix AI panel z-index by @vellankikoti ([c786d5c](https://github.com/vellankikoti/kubilitics/commit/c786d5c1557a10a00a1a34e80ae539d279e6935c))
- Remove duplicate container bar, fix + picker clipping by @vellankikoti ([2eafae5](https://github.com/vellankikoti/kubilitics/commit/2eafae5fbac606bec756aa27d6e618256a8986f4))
- Garbled text after Terminal↔FileExplorer tab switch by @vellankikoti ([bb7488e](https://github.com/vellankikoti/kubilitics/commit/bb7488e07016302f7cb1ce949e44db3c28e7f676))
- E2e audit — timer leaks, auto-reconnect, connecting timeout by @vellankikoti ([68e8014](https://github.com/vellankikoti/kubilitics/commit/68e8014347c155cb2dd90b4303e2041ba9c46322))
- Url joining, missing headers, cleaner error display by @vellankikoti ([a8cde1d](https://github.com/vellankikoti/kubilitics/commit/a8cde1d00310c34caabc08dc7b372340f6f08e9e))
- Detect and auto-fix OpenRouter misconfigured as Anthropic by @vellankikoti ([95f472a](https://github.com/vellankikoti/kubilitics/commit/95f472a85cbd8200cfbe9a9f9ec8fc8f76604f94))
- Three bugs causing the blink+reconnect loop by @vellankikoti ([c96de65](https://github.com/vellankikoti/kubilitics/commit/c96de65bb4899c385afd073639a0c07f57331dfa))
- Refresh directory when switching from Terminal tab by @vellankikoti ([b71b7a4](https://github.com/vellankikoti/kubilitics/commit/b71b7a44503cf8ea50f2027ff7b3652c0a011d8e))
- Connect only when visible — eliminates double prompt by @vellankikoti ([7cceda8](https://github.com/vellankikoti/kubilitics/commit/7cceda80633d07b28f9d4d3327bff6155f0c5903))
- Session staleness after brain restart + cleaner error messages by @vellankikoti ([a7bc3c9](https://github.com/vellankikoti/kubilitics/commit/a7bc3c956df604600782557c43bd8f1d40999519))
- Stop reporting AI Ready when API key is invalid (401/403) by @vellankikoti ([05cb2b9](https://github.com/vellankikoti/kubilitics/commit/05cb2b9575d236ef3d36d8e52b8cffccdb5116bf))
- Show all pods — slim list items, 16KB cap, total_count from backend by @vellankikoti ([3fdbb06](https://github.com/vellankikoti/kubilitics/commit/3fdbb062ba91dade7ea11ab05d0f94c1e858561d))
- Install create-dmg before Tauri macOS DMG build by @vellankikoti ([c109ad2](https://github.com/vellankikoti/kubilitics/commit/c109ad2f225a2a970f75833933188247d7983ad5))
- Update Tauri 2.10.3→2.11.2, wry→0.55.1, tao→0.35.2 for macOS 26 support by @vellankikoti ([f3d5e84](https://github.com/vellankikoti/kubilitics/commit/f3d5e841c1bee254f99d08f0cc784cc52b0a3bfe))
- Remove LSRequiresCarbon — launchd POSIX 163 spawn failure on macOS 26 by @vellankikoti ([9f61d21](https://github.com/vellankikoti/kubilitics/commit/9f61d21775f8565c576da24cb6a489f28f34ed55))
- Correct macOS DMG build — pin macos-15, build .app-only, create DMG ourselves by @vellankikoti ([6aefb6d](https://github.com/vellankikoti/kubilitics/commit/6aefb6d1f8b142ece191edbf37f3b7440f341467))
- Fix YAML parse error — heredoc Python must be indented in run blocks by @vellankikoti ([f696805](https://github.com/vellankikoti/kubilitics/commit/f69680556cd7faffbd35485e438f10e7ef33dfdb))
- Remove infoPlistData — not in Tauri CLI 2.10.1 schema by @vellankikoti ([f882dcb](https://github.com/vellankikoti/kubilitics/commit/f882dcb74902e301f230519ea54fb4541e0125e3))
- Sign DMG with codesign before notarize; use --deep re-sign by @vellankikoti ([c2825dc](https://github.com/vellankikoti/kubilitics/commit/c2825dc19c4d7c83a6c7836808eca6375d106471))


### Misc

- Delete dead createActionHandler/routeActionTool/CategoryAction code by @vellankikoti ([47148de](https://github.com/vellankikoti/kubilitics/commit/47148de8c2a50d841f8bbeafcae4397b82c37341))
- Add tool catalog + desktop launch script by @vellankikoti ([afd2cd4](https://github.com/vellankikoti/kubilitics/commit/afd2cd4f496ecc6b6e7d30652c217d465d6e230e))


### Tests

- Add behavior coverage for recommend, cost, narrate, plan tools by @vellankikoti ([92fa72a](https://github.com/vellankikoti/kubilitics/commit/92fa72a0184b0521caf0100d3909e52c66a61366))
- Add health-analysis tool tests for statefulset/job/cronjob/service/ingress/daemonset/replicaset by @vellankikoti ([a35438b](https://github.com/vellankikoti/kubilitics/commit/a35438b50e592f0ff2cb1338b68931b77fb5c3df))
- Fix stale blast-radius test — scope inference changed behavior by @vellankikoti ([e4fceb9](https://github.com/vellankikoti/kubilitics/commit/e4fceb9b86743cd50601d96f3444cb99cce1e528))

## [1.2.0-rc.11] - 2026-04-30


### Misc

- Systematic cleanup — dead code, redundancy, back-and-forths by @vellankikoti ([5de7ffa](https://github.com/vellankikoti/kubilitics/commit/5de7ffa1e6f594d32e21cacddf106bcb2324ec4e))

## [1.2.0-rc.10] - 2026-04-30


### Fixed

- Dynamic brain port — fall back when 8081 is occupied by @vellankikoti ([3e52bc3](https://github.com/vellankikoti/kubilitics/commit/3e52bc37647b8b8e88d2ea27229e63ac36ee0e38))

## [1.2.0-rc.9] - 2026-04-30


### Fixed

- Drain stdout/stderr receivers to stop binary garbage in logs by @vellankikoti ([3aaf9ea](https://github.com/vellankikoti/kubilitics/commit/3aaf9ea9f9318d2748f209e483b487d2cca76fd8))

## [1.2.0-rc.8] - 2026-04-29


### Fixed

- Add EnforcementPolicy to kubilitics-backend gRPC server by @vellankikoti ([6e7f3b9](https://github.com/vellankikoti/kubilitics/commit/6e7f3b9de514612c49f23c6615a4f6ad6f3acb7f))

## [1.2.0-rc.7] - 2026-04-29


### Added

- UseAIReady — single source of truth for AI gating by @vellankikoti ([05f35fc](https://github.com/vellankikoti/kubilitics/commit/05f35fcf29a546052cb008f61f5135de42971276))
- UseInvalidateAI — awaitable refetch of all AI queries by @vellankikoti ([05ea1d9](https://github.com/vellankikoti/kubilitics/commit/05ea1d9ae3599cd29de3fca935c7668c4348ae7a))
- UseBrainStatusListener — invalidate on Tauri brain-status events by @vellankikoti ([60b0e94](https://github.com/vellankikoti/kubilitics/commit/60b0e946dd24b1a683b88eee250e1efffa11d492))
- Mount useBrainStatusListener at App root by @vellankikoti ([fceec48](https://github.com/vellankikoti/kubilitics/commit/fceec48cf91a5169077fa0844b01ffeba2d84568))
- Check-frontend-discipline scanner — AI single-source rule + rename by @vellankikoti ([1794ccb](https://github.com/vellankikoti/kubilitics/commit/1794ccba3480a53baa7565915c6c2941d9f9fd03))
- Add Profile struct + uuid/chrono deps by @vellankikoti ([a1ccc31](https://github.com/vellankikoti/kubilitics/commit/a1ccc311237851b3afcb473b6030f8fde3f55e91))
- Atomic journal load/save with tempfile+rename by @vellankikoti ([41dfe13](https://github.com/vellankikoti/kubilitics/commit/41dfe135107b6f6cfb03dd808ba0893d1500a963))
- First-run migration from legacy YAML + ghost cleanup by @vellankikoti ([f8f414a](https://github.com/vellankikoti/kubilitics/commit/f8f414aec2168e6be0867d67d252cac9b887b15a))
- UUID-keyed keychain helpers by @vellankikoti ([4d2a30b](https://github.com/vellankikoti/kubilitics/commit/4d2a30b619c70a4402ad575b55d2b1559d765201))
- List/save/update/delete Tauri commands with rollback by @vellankikoti ([e497f31](https://github.com/vellankikoti/kubilitics/commit/e497f31e6331164e456f4ceb1299f110690a1e9a))
- Activate_profile + test_profile via brain hot-wire by @vellankikoti ([2fd3069](https://github.com/vellankikoti/kubilitics/commit/2fd3069adb3c7e15f520782a18239f1d670f85dd))
- Wire commands + migration into Tauri startup; drop brain config path by @vellankikoti ([86d2863](https://github.com/vellankikoti/kubilitics/commit/86d2863511ec3420624a56e8484b930ccceab78b))
- Replace /ai/config with /ai/active-profile (delete ghost YAML reader) by @vellankikoti ([0b811bc](https://github.com/vellankikoti/kubilitics/commit/0b811bc76302add3eb05958aaf8131f8465b2aeb))
- UseAIProfiles hook (list + active via Tauri invoke) by @vellankikoti ([958a825](https://github.com/vellankikoti/kubilitics/commit/958a8259a8dfdd1067c3e8e12f5e832df0ddcca3))
- UseActiveProfile derived hook by @vellankikoti ([b5996ab](https://github.com/vellankikoti/kubilitics/commit/b5996ab25a0d87c5c70f475a9f04455037d4ce02))
- Wire AISettingsPage Save/Test/QuickConnect to new Tauri commands by @vellankikoti ([c425935](https://github.com/vellankikoti/kubilitics/commit/c4259356491adb76def9c61a1feeb84094b9ae35))
- Profile list UI with switch/new/edit/delete actions by @vellankikoti ([24ef8e5](https://github.com/vellankikoti/kubilitics/commit/24ef8e5551048024bdbb5fe2e3d660ef6da09b28))
- One-time migration toast prompting key re-entry by @vellankikoti ([fdf6c89](https://github.com/vellankikoti/kubilitics/commit/fdf6c89e371acd2536034c6fc7fe864c90e2a5e2))
- Scanner — block direct invoke('save_profile' | ...) outside helpers by @vellankikoti ([e6814fd](https://github.com/vellankikoti/kubilitics/commit/e6814fdd771d2c798b1d02b710420aa07d654307))
- LLM-as-translator — render structured tool data, not LLM prose (#96) by @vellankikoti ([8ef8d44](https://github.com/vellankikoti/kubilitics/commit/8ef8d4452eace77afb11afa094a2d8f2f328a761))
- Rewrite LLM stub for present-tense visibility + no-kubectl-suggestion (#97) by @vellankikoti ([611abe4](https://github.com/vellankikoti/kubilitics/commit/611abe430f129c403bd56f0818c00767fed247f4))
- Register all 27 inspect_<kind> tools as Deterministic+yaml_block (#98) by @vellankikoti ([bd3e4d4](https://github.com/vellankikoti/kubilitics/commit/bd3e4d442d89e246e77af7d6790eda1ba1c149d3))
- Wire LLM-backed summary completer with deterministic fallback (#99) by @vellankikoti ([61b566f](https://github.com/vellankikoti/kubilitics/commit/61b566fc70a07d60d2a7d132f8eaba79c296ce4c))
- LogBlock — line-oriented render type for get_logs (Phase 2 #4) (#101) by @vellankikoti ([22e9eab](https://github.com/vellankikoti/kubilitics/commit/22e9eabb5db9a7e17272e16029a6f8ab5e4fdfde))
- Per-turn query flow diagram with token counts by @vellankikoti ([45d9feb](https://github.com/vellankikoti/kubilitics/commit/45d9feb71795255021e9613c53ca6d9d0a540d55))


### Changed

- UseAIReady reads from useActiveProfile; delete useAIUserConfig + aiConfigStore by @vellankikoti ([2a0f85a](https://github.com/vellankikoti/kubilitics/commit/2a0f85a6261611c7ff14b2d768845a97f168fcbf))


### Documentation

- AiConfigStore — header comment forbidding UI gating by @vellankikoti ([4643581](https://github.com/vellankikoti/kubilitics/commit/4643581e0268fe03dc131854c8aec5ebd2ec8b67))


### Fixed

- Pass submitted key to post-save Test — stop the misleading 'missing key' error by @vellankikoti ([b031668](https://github.com/vellankikoti/kubilitics/commit/b031668811ee72ff9f6a72533a63c8c88d820fde))
- Hide Quick Connect once a key is configured by @vellankikoti ([2de78d2](https://github.com/vellankikoti/kubilitics/commit/2de78d2aef84a419460272c9bdf9453201967bf6))
- Trim API key on save and test — defeat clipboard whitespace by @vellankikoti ([7758280](https://github.com/vellankikoti/kubilitics/commit/77582805cbc03589090b213ad05315c8f2ad1e9c))
- Restore Quick Connect + warn on key/provider prefix mismatch by @vellankikoti ([f2ab279](https://github.com/vellankikoti/kubilitics/commit/f2ab279ff251c2627d5b823a01999a8f33c7f83d))
- Non-fatal /models probe in Custom client + 15s timeout by @vellankikoti ([74f2e72](https://github.com/vellankikoti/kubilitics/commit/74f2e724133c5ecf9f1d655f40b3b77b7f08f1c8))
- Fail loud on port 8190 conflict + dev preflight kill by @vellankikoti ([d2f5aea](https://github.com/vellankikoti/kubilitics/commit/d2f5aeadb33d5e21b592c848b6a74ccdedb1d585))
- Cross-platform port preflight + Windows-aware PID lookup by @vellankikoti ([0ccaa17](https://github.com/vellankikoti/kubilitics/commit/0ccaa17024d083db6c536dc512a4eb0fcde5b707))
- Option B — drop Vite proxy, route every fetch via apiUrl() by @vellankikoti ([545f361](https://github.com/vellankikoti/kubilitics/commit/545f361100bcbd5d812984169f984528d6865154))
- UseAICapabilities — lockstep refetch with useAIStatus by @vellankikoti ([e9017cb](https://github.com/vellankikoti/kubilitics/commit/e9017cb60909d69314195deb781b7da596a08db3))
- UseAIUserConfig — lockstep refetch with useAIStatus by @vellankikoti ([acd7214](https://github.com/vellankikoti/kubilitics/commit/acd7214e9507cf02726f10213c178c66c3a08e57))
- Migrate ChatStatusPill + ChatPanel to useAIReady; delete decodeNotReady by @vellankikoti ([807b27d](https://github.com/vellankikoti/kubilitics/commit/807b27d2c2db37087922c496ba484362b56a2644))
- AISettingsPage.handleSave awaits useInvalidateAI by @vellankikoti ([c341957](https://github.com/vellankikoti/kubilitics/commit/c341957a24f73b1c14fa87b3bfd9e18e87c8065b))
- Test_llm_connection — fall back to provider-specific env vars by @vellankikoti ([8169158](https://github.com/vellankikoti/kubilitics/commit/8169158aa4caa627de544c7c6e988e6274e56818))
- UseAIReady — accept empty status.state when capabilities is ready by @vellankikoti ([4d8c7dc](https://github.com/vellankikoti/kubilitics/commit/4d8c7dc039fbc0869c1d36f48f053ffdd2e0b1c4))
- Delete_profile restores in-memory on disk fail; drop unused BrainHotwire variant by @vellankikoti ([4f7d12f](https://github.com/vellankikoti/kubilitics/commit/4f7d12fc48d38e12c9946e5025b687448010971e))
- Overlay drawer pattern — main content no longer squeezed when chat opens by @vellankikoti ([ab934ea](https://github.com/vellankikoti/kubilitics/commit/ab934eab735beb7adb85c163237cdcfc8c0cc918))
- Replace global migration toast with inline banner on AI Settings by @vellankikoti ([656fdc4](https://github.com/vellankikoti/kubilitics/commit/656fdc4ebd42e7dbff53ae502239490364e470a8))
- Test button uses typed key + tighten 'configured' for hosted custom by @vellankikoti ([3351afe](https://github.com/vellankikoti/kubilitics/commit/3351afe684b2510982b4a1875198f3b7df4f1650))
- Cold-start hot-wire active profile on Tauri startup by @vellankikoti ([b8661bc](https://github.com/vellankikoti/kubilitics/commit/b8661bcc93d75bc37b47b619317da14ba65bc151))
- Surface active.last_error as brain_error + lock 4 regression guards by @vellankikoti ([cb1ef57](https://github.com/vellankikoti/kubilitics/commit/cb1ef575bd2df86d90026f48d002ee1c8a857f02))
- Silence ENHANCE_YOUR_CALM — disable idle pings, add server enforcement policy by @vellankikoti ([8ffe270](https://github.com/vellankikoti/kubilitics/commit/8ffe270c1f9efbd6badee633a0c2d2dc592554d5))


### Tests

- Expand hallucination probes 10 → 30 with axis tags (#100) by @vellankikoti ([eda54ca](https://github.com/vellankikoti/kubilitics/commit/eda54ca8ee737210bb97280467c579219095eb85))

## [1.2.0-rc.6] - 2026-04-24


### Added

- Real token counts + cost + finish-state in every turn footer by @vellankikoti ([abb10eb](https://github.com/vellankikoti/kubilitics/commit/abb10ebf1c314fe0d0422f6b5d7b4691adf0750a))
- Add macOS entitlements with keychain-access-groups by @vellankikoti ([6687bcb](https://github.com/vellankikoti/kubilitics/commit/6687bcb8958ba213685c3f55039c5dc93f8aa030))
- Wire entitlements.plist into tauri.conf macOS bundle by @vellankikoti ([d4b2305](https://github.com/vellankikoti/kubilitics/commit/d4b2305c8ae2891c9a7dbfa4d251f8e17b04b09e))
- Regression guard for macOS signing config by @vellankikoti ([4c43ba5](https://github.com/vellankikoti/kubilitics/commit/4c43ba5391b75e15d8e2637f231b01f23972ce52))


### Documentation

- Add macOS signing + notarization runbook by @vellankikoti ([f77ac3c](https://github.com/vellankikoti/kubilitics/commit/f77ac3c51d6fb5adf90fe932294a7e1a9b01591b))
- Note macOS signing + notarization in v1.2.0 by @vellankikoti ([15570ce](https://github.com/vellankikoti/kubilitics/commit/15570ce27be8e58309391cb36f41a833fcc4a8be))


### Fixed

- Three systemic bugs that made local-dev "AI Unreachable" permanent by @vellankikoti ([4adb49f](https://github.com/vellankikoti/kubilitics/commit/4adb49fe071b30c01871a280585f1b7f5014dbb7))
- Kill the SECOND keychain probe on AI Settings page load by @vellankikoti ([04f6c21](https://github.com/vellankikoti/kubilitics/commit/04f6c219dccaab76c5d1ecdd556eb23ead148899))
- Remove keychain migration — no prompts on page open, ever by @vellankikoti ([10e141a](https://github.com/vellankikoti/kubilitics/commit/10e141ad086a33352408c5542ef87931891a1192))
- Test connection for Ollama hits /api/tags, not /chat/completions by @vellankikoti ([788227d](https://github.com/vellankikoti/kubilitics/commit/788227de61b21a05023df079079cb9b9bafae4d3))
- Ollama model list is dynamic — fetch from /api/tags, auto-select by @vellankikoti ([d49fa7d](https://github.com/vellankikoti/kubilitics/commit/d49fa7d73e364c5bbe8c90a65cd6e4465b6e437e))
- Reconcile the "Connected" / "AI Unreachable" paradox by @vellankikoti ([6c9691f](https://github.com/vellankikoti/kubilitics/commit/6c9691fc79eea7cf89ea8678216cafee2849fc93))
- Never fatal on LLM init failure — hot-wire works even from cold start by @vellankikoti ([af8f2f7](https://github.com/vellankikoti/kubilitics/commit/af8f2f7ac132685a10282130b86374dde636d224))
- Surface real stream errors instead of masquerading them as 0-token done by @vellankikoti ([6492401](https://github.com/vellankikoti/kubilitics/commit/6492401ff5e68666b0fb36753d39dec8fe59949e))
- Enable tool router by default for Ollama (small-model prompt overflow fix) by @vellankikoti ([83555c7](https://github.com/vellankikoti/kubilitics/commit/83555c72491d9cf03744d96f714d18b5fc88b30d))
- Fail-closed notarization — no silent un-notarized ships by @vellankikoti ([1ac6b79](https://github.com/vellankikoti/kubilitics/commit/1ac6b7935de5a7eb97359da84a28d6c60c235cdc))
- Honest readiness — adapter probe drives ready flag end-to-end by @vellankikoti ([cf3d7a9](https://github.com/vellankikoti/kubilitics/commit/cf3d7a93359f7856a685e28ec633f95ace01325a))
- Skip beforeBuildCommand in CI — frontend dist is prebuilt by @vellankikoti ([88e3088](https://github.com/vellankikoti/kubilitics/commit/88e3088ebba9809a011a28ce5a46d3f784444933))
- Portable beforeBuildCommand — extract to bash script by @vellankikoti ([6186cdf](https://github.com/vellankikoti/kubilitics/commit/6186cdfa909cabb8e19135bfde61c383d7346102))


### Misc

- Call macOS signing-config guard by @vellankikoti ([821acd8](https://github.com/vellankikoti/kubilitics/commit/821acd8e688ddda63accd0647dcb7949637876f2))

## [1.2.0-rc.4] - 2026-04-24


### Fixed

- Dynamic backend port + never piggyback — kills the 1000th-time orphan-UI regression by @vellankikoti ([c9320bb](https://github.com/vellankikoti/kubilitics/commit/c9320bbea3242837aef602624b4674595eea282e))

## [1.2.0-rc.3] - 2026-04-24


### Added

- Setup-ollama.sh — one-shot bootstrap for throwaway bench VM by @vellankikoti ([e85008c](https://github.com/vellankikoti/kubilitics/commit/e85008c77a66d68102b3f92fca7909d79154f5c6))
- Implement KOTG.AI local AI agent — Phase 1 foundation by @claude ([35233be](https://github.com/vellankikoti/kubilitics/commit/35233beb8ff3934e15922eceb5a87cbe65122cb7))
- Add EC2 development machine setup guide and bootstrap script by @claude ([f019987](https://github.com/vellankikoti/kubilitics/commit/f019987889d8420af3f7c9c3dfec4a6501bd3fad))
- Add EC2 launch and SSH tunnel management scripts by @claude ([31a0f27](https://github.com/vellankikoti/kubilitics/commit/31a0f2757f4c97dae0f495e6f0fe2fa002289be3))
- Add kubilitics-ai module from kubilitics-os-emergent by @vellankikoti ([a37ced9](https://github.com/vellankikoti/kubilitics/commit/a37ced9b803a7aa85c630c80a154d10e39f613bb))
- Add agent_runtime.proto for backend→AI runtime contract by @vellankikoti ([4e90ff7](https://github.com/vellankikoti/kubilitics/commit/4e90ff7d82daa446f245126adca4b07a5dd667be))
- Minimal AgentRuntimeService stub (LLM-direct only) for subproject 3a by @vellankikoti ([21a31f4](https://github.com/vellankikoti/kubilitics/commit/21a31f4e40d11912bd51e35cb626b24bb71eeaac))
- Register AgentRuntimeService gRPC + /status HTTP endpoint by @vellankikoti ([1fea6a1](https://github.com/vellankikoti/kubilitics/commit/1fea6a1dd6603a8eb9216effb014e881743ed8d6))
- Introduce Router + Engine abstraction (subproject 3b) by @vellankikoti ([c09761d](https://github.com/vellankikoti/kubilitics/commit/c09761d755bb3b1566ae560984ad613e81afbf4d))
- Add kagent + python multi-agent Engine skeletons (3c + 3d) by @vellankikoti ([c5c94d5](https://github.com/vellankikoti/kubilitics/commit/c5c94d563beb76cfe3dd6085c390336d995acc4c))
- Wrap Router with v1 Safety wrapper (subproject 3e) by @vellankikoti ([37ee083](https://github.com/vellankikoti/kubilitics/commit/37ee08312a83c0977610ed453c23153269eb85c7))
- Kubilitics-ai chart for one-command in-cluster install (subproject 3f) by @vellankikoti ([b163037](https://github.com/vellankikoti/kubilitics/commit/b163037c22dae093be97de63974290127d964804))
- Wide-event LLM benchmarking harness for kubilitics-ai (Ollama/OpenAI/Anthropic) by @vellankikoti ([fded192](https://github.com/vellankikoti/kubilitics/commit/fded192fa5d66610b65ea3b6eca19986eaccf852))
- V1.5 wire-level engine + audit bridge by @vellankikoti ([4409bff](https://github.com/vellankikoti/kubilitics/commit/4409bffb9d67ee3951dc92f3401327a7761f77e7))
- Wire CompleteWithTools so MCP tools fire from LLM-direct path by @vellankikoti ([94a9853](https://github.com/vellankikoti/kubilitics/commit/94a985338bebd5d1cadda79c5ef301a99dd02696))
- Comprehensive 498-case AI tool coverage bench (auto-generated from taxonomy) by @vellankikoti ([1f4a5a3](https://github.com/vellankikoti/kubilitics/commit/1f4a5a35f595f413c50c20cba0708d6c5f38e7e9))
- Expand aliases (50+), fix content:null bug, support autonomy=3 by @vellankikoti ([b40a409](https://github.com/vellankikoti/kubilitics/commit/b40a409280853626e7164d33a2aa2c1776bda78c))
- Extract + harden chat system prompt by @vellankikoti ([1486a80](https://github.com/vellankikoti/kubilitics/commit/1486a807b36ca5eb94bb97532a1d0f666a31ea19))
- Extract summarizer + add 8KB hard cap by @vellankikoti ([ec879cd](https://github.com/vellankikoti/kubilitics/commit/ec879cdf9c5c799a7fbab2d3af430edf6bcc27dc))
- Cap every tool dispatch output at 8KB by @vellankikoti ([abc88c5](https://github.com/vellankikoti/kubilitics/commit/abc88c5091db3bd1d9a57452b14a5e7726504d21))
- Add end-to-end chat quality harness by @vellankikoti ([6d71d37](https://github.com/vellankikoti/kubilitics/commit/6d71d37c62f051c04325f7dd364a4a9c6ccca750))
- Per-turn routing recorder (JSONL) by @vellankikoti ([0262bcc](https://github.com/vellankikoti/kubilitics/commit/0262bccf2acef4c81487c3663d1b817e3d3a3890))
- Per-turn token tally + USD from price table by @vellankikoti ([796295e](https://github.com/vellankikoti/kubilitics/commit/796295e606d21f52f2f307ad169391f6dfd28488))
- Inject routing stages at every chat boundary by @vellankikoti ([16a1c7a](https://github.com/vellankikoti/kubilitics/commit/16a1c7a984d917c28104e5dbf9bb5f1ff55ee2f5))
- --trace-dir wires per-turn routing JSONL by @vellankikoti ([504e62c](https://github.com/vellankikoti/kubilitics/commit/504e62cdd49550141cdbec2bfda579e41efd69b4))
- Self-contained HTML + inline SVG renderer by @vellankikoti ([ddbaf92](https://github.com/vellankikoti/kubilitics/commit/ddbaf92783ca32d5c9131885b20cdca62405c034))
- AgentStreamEvent carries optional TokenUsage by @vellankikoti ([a1da588](https://github.com/vellankikoti/kubilitics/commit/a1da588953cc1cccc24dc2ba29710a14248f6f72))
- Surface prompt_eval_count + eval_count to AgentStreamEvent by @vellankikoti ([80c9eb8](https://github.com/vellankikoti/kubilitics/commit/80c9eb8b698bc1f6e6aebe8f681178d09ba78bf4))
- Emit done + cost stages at end of each chat turn by @vellankikoti ([a0efadc](https://github.com/vellankikoti/kubilitics/commit/a0efadc65c166c18706abbcbddc561fa9927ec0d))
- Tool-catalog extractor + 20 seed plain-english entries by @vellankikoti ([48398c3](https://github.com/vellankikoti/kubilitics/commit/48398c3456d382e68665955c169e182c984f73b4))
- V2 interactive template with tool explorer by @vellankikoti ([061b820](https://github.com/vellankikoti/kubilitics/commit/061b820bcbd4f2e75de1c4ea7096bcd27874f680))
- Realistic incident-scenarios suite + walkthrough journey cards by @vellankikoti ([5bf438b](https://github.com/vellankikoti/kubilitics/commit/5bf438b31e2a0e9c9a307ed5058fa5a475d856dd))
- Topic-aware tool selection — 163 tools → top 30 per query by @vellankikoti ([3d00e3a](https://github.com/vellankikoti/kubilitics/commit/3d00e3a707ab86060c409295199b1f519a2f833a))
- Together.ai provider config + API-only orchestrator by @vellankikoti ([d08c078](https://github.com/vellankikoti/kubilitics/commit/d08c07886c64209b7b98bfb75041839241884b25))
- Capture first 4KB of assistant text in llm_text_out by @vellankikoti ([d19dfec](https://github.com/vellankikoti/kubilitics/commit/d19dfec5036a3ec480d65c115f3a1bb0d840c2b4))
- Inspect_<kind> composites — fold detailed+events+ownership into one call per kind by @vellankikoti ([28b3751](https://github.com/vellankikoti/kubilitics/commit/28b3751744d7848579d00a4a030115d9abd96f29))
- Close gap 6 — analyze_blast_radius schema (scope-aware) by @vellankikoti ([a0061b3](https://github.com/vellankikoti/kubilitics/commit/a0061b32b677028bc5f8804317a0f764d77c43f4))
- Close gap 2 — resolve_resource name-hint -> namespace by @vellankikoti ([30ed412](https://github.com/vellankikoti/kubilitics/commit/30ed412c662ca3ce17a24873f6da3d97f6ed202b))
- Close gap 5 — log-streaming REST fallback by @vellankikoti ([abf0b74](https://github.com/vellankikoti/kubilitics/commit/abf0b74d2f57333e4b98bcf19fd363167cc17714))
- Close gap 4 — observe_recent_changes timeline fusion by @vellankikoti ([73a0089](https://github.com/vellankikoti/kubilitics/commit/73a00899a7cb830afcab4dc5795f51abbb145ee4))
- Close gap 3 — who_can_do RBAC aggregator by @vellankikoti ([435d483](https://github.com/vellankikoti/kubilitics/commit/435d48399f135f60d066568cd325a169a6c06145))
- Close gap 1 — live metrics adapter (pod/node/top-N) by @vellankikoti ([7c6b71f](https://github.com/vellankikoti/kubilitics/commit/7c6b71f7e8469ecf6e7e5bc2e6397ae122d2f76c))
- Self-redirecting tool descriptions — aggregators claim their scope by @vellankikoti ([9335250](https://github.com/vellankikoti/kubilitics/commit/93352509e68f53b9abf8224a4e4b8c1bc6ebe523))
- Validate tool args — reject fabricated wildcard params with redirect hints by @vellankikoti ([9fe3f8c](https://github.com/vellankikoti/kubilitics/commit/9fe3f8ca65ebb3e1a17d2ee463e5c09caf31b964))
- 3 real-world aggregators — services_by_filter, secrets_usage, ingresses_by_tls_expiry by @vellankikoti ([b4ba134](https://github.com/vellankikoti/kubilitics/commit/b4ba134bec3027b85ee57a54c61d6054e8138d81))
- Bias aggregator + composite tools to the top of the filtered list by @vellankikoti ([55bb54d](https://github.com/vellankikoti/kubilitics/commit/55bb54dbb268f00fbcdae18a2dda3b4beb138711))
- Retry with exp-backoff on 5xx by @vellankikoti ([bc97c77](https://github.com/vellankikoti/kubilitics/commit/bc97c770ac0e7487e698f83cc007730860929a89))
- Honor standard Retry-After header on 429 by @vellankikoti ([284d5f6](https://github.com/vellankikoti/kubilitics/commit/284d5f6405205fc90cef253530e5adab50406bdc))
- SQLite session persistence by @vellankikoti ([231e911](https://github.com/vellankikoti/kubilitics/commit/231e911dd79408d05d3b7e03e779ec56277f8130))
- Multi-cluster session isolation by @vellankikoti ([8c5b7ee](https://github.com/vellankikoti/kubilitics/commit/8c5b7ee0fe06b7d98191c7b731aea2f6e441d383))
- Pre-spend Gate primitive for live enforcement by @vellankikoti ([6002ffd](https://github.com/vellankikoti/kubilitics/commit/6002ffde4958154cbdfab6db4225ecd92d4486e3))
- Wire chat.SessionStore into grpc Send path by @vellankikoti ([0e50d95](https://github.com/vellankikoti/kubilitics/commit/0e50d956817fcbb682734c9545d80f8b5ea7a44a))
- Wire Gate into Send path + admin HTTP + reset by @vellankikoti ([2915f60](https://github.com/vellankikoti/kubilitics/commit/2915f60783a89455fa69326c66b77d34266ec1a2))
- Phase 3 C1 — 10 observability aggregators by @vellankikoti ([c42c041](https://github.com/vellankikoti/kubilitics/commit/c42c0417d7317441de862e2b586d2283b2364ef5))
- Phase 3 C2 — 10 root-cause diagnostics by @vellankikoti ([ad5e369](https://github.com/vellankikoti/kubilitics/commit/ad5e3694b44ac758f30017ad83ab68a3752dad37))
- Phase 3 C3 — 10 planning tools by @vellankikoti ([6ff55c1](https://github.com/vellankikoti/kubilitics/commit/6ff55c1adaa4e6658a075f96d4458fc444bc2df8))
- Phase 3 C4 — 10 compliance checks by @vellankikoti ([9b092a3](https://github.com/vellankikoti/kubilitics/commit/9b092a3b2fa01cff86feedf8e820b6860a40fb6a))
- Phase 3 C5 — 10 narrative tools (Phase 3 complete) by @vellankikoti ([ed9eb98](https://github.com/vellankikoti/kubilitics/commit/ed9eb98b885a0230e8952f53bbc951d037249534))
- LLM-as-judge for chat-quality bench (Phase 4 D) by @vellankikoti ([151cad6](https://github.com/vellankikoti/kubilitics/commit/151cad6c9c741138a12281c07b6a2017e5729cfd))
- Phase 4 D.3 — render LLM-as-judge scores by @vellankikoti ([1e65666](https://github.com/vellankikoti/kubilitics/commit/1e6566688c3b3ae7e22af4d7e6e570fb70f9d154))
- KUBILITICS_AI_GRPC_PORT env overrides :50051 default by @vellankikoti ([2f93c21](https://github.com/vellankikoti/kubilitics/commit/2f93c21f89e2390744e9370f205f998c9a460ac8))
- Honor KUBILITICS_LLM_API_KEY / _BASE_URL as post-yaml override by @vellankikoti ([0dad4fa](https://github.com/vellankikoti/kubilitics/commit/0dad4fafdc59572ab3bb5f3c8d1a7e16b48ac540))
- Add severity-scoring primitives (ScorePod, ScoreNode, ScoreEvent) by @vellankikoti ([49267a6](https://github.com/vellankikoti/kubilitics/commit/49267a67e056b1e9225135c981102e465c3745e7))
- Add RankCluster composition over pod/node/event scores by @vellankikoti ([306c771](https://github.com/vellankikoti/kubilitics/commit/306c77187a29d3cb488bebff6151eb118bd66a4d))
- Add RankProblems with filter-enum predicates by @vellankikoti ([b47a715](https://github.com/vellankikoti/kubilitics/commit/b47a7153ef3d86b8d7480d40801f5a955c931e8a))
- Extract with ordered regex-template strip rules by @vellankikoti ([08eb6fd](https://github.com/vellankikoti/kubilitics/commit/08eb6fd190592b6c95edc8e0859af126aea9803f))
- Cluster aggregates lines by extracted template by @vellankikoti ([da7e282](https://github.com/vellankikoti/kubilitics/commit/da7e28253f08e80f58b295db2cbeead8a52905ed))
- Add buildComposableResult envelope builder by @vellankikoti ([b6c8872](https://github.com/vellankikoti/kubilitics/commit/b6c8872192a24f5088be44be50765daec66638e9))
- HandleTriageCluster composite with triage.RankCluster by @vellankikoti ([ba4f708](https://github.com/vellankikoti/kubilitics/commit/ba4f7085a4dbf9b4f4e1ad48288db205c9bbe404))
- HandleListProblems typed-filter enumerator by @vellankikoti ([99fb9e4](https://github.com/vellankikoti/kubilitics/commit/99fb9e4c086acbaf6261c71996db6e76d974d7b9))
- HandleSearchLogs with logpattern.Cluster aggregation by @vellankikoti ([c7619c2](https://github.com/vellankikoti/kubilitics/commit/c7619c2152b6f9482c1be523b1db369ff2111682))
- +triage_cluster +list_problems +search_logs by @vellankikoti ([005a55a](https://github.com/vellankikoti/kubilitics/commit/005a55a509f9908c3973fc9f01c1a597af31ce96))
- Retire 25 folded observe_* tools; add retirement-guard test by @vellankikoti ([c5fe831](https://github.com/vellankikoti/kubilitics/commit/c5fe83180efb4e4fd2e1f71c2285cfabe7ecceea))
- ResilientResponse envelope type by @vellankikoti ([2551df7](https://github.com/vellankikoti/kubilitics/commit/2551df7a100681ee30f8c99c4cadbc50c12ac378))
- Thread-safe LRU cache with entry timestamps by @vellankikoti ([8242cf2](https://github.com/vellankikoti/kubilitics/commit/8242cf2442daa7333e1bd58fb276bbd490aabf93))
- IsTransientClusterError classifier by @vellankikoti ([94df785](https://github.com/vellankikoti/kubilitics/commit/94df785eeea14875d13edea8afb6a67aa0e8cdb4))
- WrapClusterHandler middleware by @vellankikoti ([f2a19ce](https://github.com/vellankikoti/kubilitics/commit/f2a19ce66c2f8d7ac055fb6ba084c617d6a1491c))
- LogicalIdentity \u2014 (name, serverUrl) replaces UUID by @vellankikoti ([529b8df](https://github.com/vellankikoti/kubilitics/commit/529b8dfa9173c032d1e3860d42be787854dcfdf4))
- GET /api/v1/presence endpoint stub by @vellankikoti ([3058fe4](https://github.com/vellankikoti/kubilitics/commit/3058fe432aa02676ba27b027c2b39bfd3029bba8))
- Wire presence endpoint into router + feature flag by @vellankikoti ([6d9c982](https://github.com/vellankikoti/kubilitics/commit/6d9c98282bd236522f552d0352dddc7f4b54ccf7))
- DiscoverySource interface + event types by @vellankikoti ([cc92a12](https://github.com/vellankikoti/kubilitics/commit/cc92a129ac737e56f6b787bac71be3073c7c7e8b))
- KubeconfigFileSource Enumerate by @vellankikoti ([86d1534](https://github.com/vellankikoti/kubilitics/commit/86d1534117a0f17e43ff79796f17801674948199))
- Fsnotify-backed kubeconfig live watching by @vellankikoti ([d52ef8f](https://github.com/vellankikoti/kubilitics/commit/d52ef8f14f337d19979ddbaa39dc63ea3139cf61))
- KubernetesSecretSource implements DiscoverySource by @vellankikoti ([6141083](https://github.com/vellankikoti/kubilitics/commit/6141083d43d182a66d07e87b20bc33960a53d0af))
- ManualSource — sqlite-backed + pub/sub by @vellankikoti ([8589b27](https://github.com/vellankikoti/kubilitics/commit/8589b272f2e3f5d17df11bd401aee8f1a872a28b))
- Manager composes sources + fans in events by @vellankikoti ([51214cb](https://github.com/vellankikoti/kubilitics/commit/51214cb4a951ae119845880508f02230bc1fbbb6))
- Real Manager wired + SSE event stream by @vellankikoti ([9052c46](https://github.com/vellankikoti/kubilitics/commit/9052c46bf0db4dac8673f068588047c016453084))
- Feature flag reader for onboarding-v2 by @vellankikoti ([c9d4bca](https://github.com/vellankikoti/kubilitics/commit/c9d4bca187b4bb058c61fc9aabcbe5f68cf9e4dc))
- LogicalIdentity + ResilientResponse types by @vellankikoti ([32c9c14](https://github.com/vellankikoti/kubilitics/commit/32c9c1404dcefc423c72bbd7938f8fbdcafced34))
- ClusterPresenceStore — single source of truth by @vellankikoti ([27e32ae](https://github.com/vellankikoti/kubilitics/commit/27e32ae35d72c2e37ea3c755ab013d391a7241c0))
- UseClusterPresence SSE subscription by @vellankikoti ([51a61f3](https://github.com/vellankikoti/kubilitics/commit/51a61f3bd5e5d3912f2a37cbd38bbb4780bb6af1))
- UseResilientQuery hook — generalizes 91d6970 by @vellankikoti ([00336b6](https://github.com/vellankikoti/kubilitics/commit/00336b69aae603acdaca35a9f6d1806091308bc0))
- ClusterUnreachableBoundary component by @vellankikoti ([6f46aca](https://github.com/vellankikoti/kubilitics/commit/6f46aca3f9dd3064a6860297c5191c0508675ed5))
- ClusterPickerPage — new /clusters landing by @vellankikoti ([903e899](https://github.com/vellankikoti/kubilitics/commit/903e8995286317cde8f015ef591c648449645bf5))
- WelcomePage zero-state for new users by @vellankikoti ([8650fe5](https://github.com/vellankikoti/kubilitics/commit/8650fe5e4e65f8ede640688108b54ecbf2f9140f))
- Flag-gated routing to /clusters + /welcome by @vellankikoti ([147d8fa](https://github.com/vellankikoti/kubilitics/commit/147d8fab611b909b65ccf9548911fe56e94a7f7f))
- FEATURE_PRESENCE_V2 default on — onboarding v2 live by @vellankikoti ([9a6c4d0](https://github.com/vellankikoti/kubilitics/commit/9a6c4d00aaa827517c807173ce2cd3375e4f014b))
- Deprecation banner on /connect page by @vellankikoti ([50feed0](https://github.com/vellankikoti/kubilitics/commit/50feed09f75767fb38d3ceb3e85b8b8ad8260b8d))
- Wire session_id + provider through to ConnectedCluster by @vellankikoti ([06004e8](https://github.com/vellankikoti/kubilitics/commit/06004e8aaa6460ec1653011e82d2952ea0d00444))
- AddClusterDialog — reusable modal extracted from /connect page by @vellankikoti ([82119a3](https://github.com/vellankikoti/kubilitics/commit/82119a3d7bcc7f6bc6fdcc2a4eeacfe470446649))
- Integrate AddClusterDialog into ClusterPickerPage + WelcomePage by @vellankikoti ([43e8c72](https://github.com/vellankikoti/kubilitics/commit/43e8c722f66a367a4f7c36cfa477ae23d2364e17))


### Changed

- Name threshold constants + wire ScoreEvent into RankCluster by @vellankikoti ([d282837](https://github.com/vellankikoti/kubilitics/commit/d282837597a51aaf09cb88b5f66239ecb910815a))
- Per-filter tests + harden image_pull/unhealthy predicates by @vellankikoti ([aace2ca](https://github.com/vellankikoti/kubilitics/commit/aace2ca5b64c5c4366ad045619a424c93f5294b5))
- Extract shared types to break import cycle by @vellankikoti ([a5e0122](https://github.com/vellankikoti/kubilitics/commit/a5e0122ea62d900362ea5f34f22820f20cd58c60))
- Migrate /summary onto resilient.WrapClusterHandler by @vellankikoti ([5f3a264](https://github.com/vellankikoti/kubilitics/commit/5f3a264ffcb4781b7be6a9f4cbc9c5afd50faee3))
- UseResourceCounts rebuilt on useResilientQuery by @vellankikoti ([afde9b6](https://github.com/vellankikoti/kubilitics/commit/afde9b62923aee70158eb924442de34dde5edc83))
- Migrate /events onto resilient.WrapClusterHandler by @vellankikoti ([c7a9aab](https://github.com/vellankikoti/kubilitics/commit/c7a9aab56d11936b60946095b5684d8e976d9927))
- Use shared ClusterUnreachableBoundary by @vellankikoti ([684cef7](https://github.com/vellankikoti/kubilitics/commit/684cef7c889a33e70dbb1cbcdb0fb56deb0a9758))
- Delete onboardingStore, derive onboarded-state from clusterPresenceStore by @vellankikoti ([a5cce40](https://github.com/vellankikoti/kubilitics/commit/a5cce403e3ebc420cd970e3aa82f1a092373a1f8))
- Remove FEATURE_PRESENCE_V2 flag; V2 path is now the path by @vellankikoti ([05fffca](https://github.com/vellankikoti/kubilitics/commit/05fffcad183b1acd7dd22df8cef80d0880aef496))
- Extract namespaceStore from clusterStore by @vellankikoti ([97184e4](https://github.com/vellankikoti/kubilitics/commit/97184e46a21ab0cc4a6d4fad9a98cf009fb797c6))
- Extract demoStore from clusterStore by @vellankikoti ([462aa1b](https://github.com/vellankikoti/kubilitics/commit/462aa1ba5bde392daee1409b4fa8cdcdd6fabf91))
- Extract kubeconfigStore from clusterStore by @vellankikoti ([ec36000](https://github.com/vellankikoti/kubilitics/commit/ec360002dbbfcc1592202bd9f34699478df8c481))
- Extract appModeStore from clusterStore; derive isOnboarded from presence by @vellankikoti ([0f40696](https://github.com/vellankikoti/kubilitics/commit/0f40696f288bc0d7d7374891a464a06c1fb79a6b))
- Remove clusterStore mirror writes from domain stores by @vellankikoti ([baea946](https://github.com/vellankikoti/kubilitics/commit/baea94689fd8b7eea297ee5de58ff3501239ad2d))
- UseActiveClusterId prefers presence session_id by @vellankikoti ([b066d59](https://github.com/vellankikoti/kubilitics/commit/b066d59fb5709cd7d86988841dbbb3d6b434f5e5))
- Migrate cluster readers to clusterPresenceStore by @vellankikoti ([b3c4d00](https://github.com/vellankikoti/kubilitics/commit/b3c4d00f33d54343123d082d7df0edc6c243df80))
- Delete /connect page + FlagGatedRoute + reroute CTAs by @vellankikoti ([0e21dd5](https://github.com/vellankikoti/kubilitics/commit/0e21dd59d5b71ed0010ff57d2387be3bec499054))
- Complete sweep + delete legacy stores/pages by @vellankikoti ([66c0e61](https://github.com/vellankikoti/kubilitics/commit/66c0e615406d28ce223d34339f433a2b70419175))


### Documentation

- Open-core tiers + 1.2.0 roadmap drafts (#91) by @vellankikoti ([4dce8a1](https://github.com/vellankikoti/kubilitics/commit/4dce8a14b5cb8f35585e0da374cfbcd7170ce8c1))
- UI smoke harness as migration-phase gate by @vellankikoti ([879b6db](https://github.com/vellankikoti/kubilitics/commit/879b6db87a7773ef27d3ab5c53ac0891009f9bb9))
- Comprehensive research-validated architecture update (v2.0) by @claude ([b328c7e](https://github.com/vellankikoti/kubilitics/commit/b328c7ec2aacd14a6d4c338b3aae89c6e677ae54))
- Chat-quality bench — 250 prompts, systematic-quality validation by @vellankikoti ([df97a9a](https://github.com/vellankikoti/kubilitics/commit/df97a9a66245c1a49329e341530eb14fa79406ed))
- Attach bench log artifact by @vellankikoti ([510dd35](https://github.com/vellankikoti/kubilitics/commit/510dd3574069158d6afc0950385660057d170be0))
- Rate-limit runbook by @vellankikoti ([f972d3c](https://github.com/vellankikoti/kubilitics/commit/f972d3c1b209b41678f8ae3f10d6b95856db0806))
- 2026-04-22 Together.ai iteration traces + scenarios-100 suite by @vellankikoti ([d7853c6](https://github.com/vellankikoti/kubilitics/commit/d7853c6378e146054b934b21d132f150a1d49b7a))
- Phase 3 C.51 — plain-English for all 183 tools by @vellankikoti ([556637e](https://github.com/vellankikoti/kubilitics/commit/556637e9f0dfc6e0c0dcf704e5cbbcb24f88e372))
- Week 1 — complete inspect surface + retire 42 observe_* by @vellankikoti ([de514bf](https://github.com/vellankikoti/kubilitics/commit/de514bfd790714ac4e7b7840132c43a8dedb01c4))
- Correct retirement count 42 → 25 with exact name lists by @vellankikoti ([965cfa5](https://github.com/vellankikoti/kubilitics/commit/965cfa55dcbba12d19b06ee01e3485246cb4067b))
- Refresh onboarding docs for auto-discover flow by @vellankikoti ([809a28f](https://github.com/vellankikoti/kubilitics/commit/809a28f1e12379018ad8769f7c727ce3cb97ac38))


### Fixed

- Soft-fail cosmetic aux workflows by @vellankikoti ([3aad41f](https://github.com/vellankikoti/kubilitics/commit/3aad41f4072fb03b90f2c8617f7b0f5f55ebf735))
- Adopt self-pull Pattern E for tap sync (#88) by @vellankikoti ([bf2293f](https://github.com/vellankikoti/kubilitics/commit/bf2293f3b8a26373caaf959e55ce11cecd10b992))
- Consolidate chart publishing to signed OCI on ghcr.io (#90) by @vellankikoti ([f36e1c0](https://github.com/vellankikoti/kubilitics/commit/f36e1c06ee24e814fa346b58f5a9683c2027f6b1))
- Surface helm push errors in workflow log (#92) by @vellankikoti ([44373c2](https://github.com/vellankikoti/kubilitics/commit/44373c21f2995e73dcc49a8adeab70119ce3c563))
- Use kotg-schema chat.proto + AIControl as the contract by @vellankikoti ([43ab63c](https://github.com/vellankikoti/kubilitics/commit/43ab63c0243b7339c2a5e7193644b73dc1f2b4ec))
- Honor -config flag and fail loud on misconfigured LLM by @vellankikoti ([671d0ab](https://github.com/vellankikoti/kubilitics/commit/671d0ab77ff9d9e752c39731c4fbe53fe5c75c7b))
- Standardize backend HTTP port to 8190 (was stale 819) by @vellankikoti ([745ebc3](https://github.com/vellankikoti/kubilitics/commit/745ebc35d21c1e887447f5628f4c1ab3037c6aae))
- Accept JSON array shape from backend /events endpoint by @vellankikoti ([ac3907a](https://github.com/vellankikoti/kubilitics/commit/ac3907ac2d252e854cfcfea955040b5487e1dda5))
- Inject focus_cluster_id into LLM system prompt + tool args by @vellankikoti ([acc9c2a](https://github.com/vellankikoti/kubilitics/commit/acc9c2abc8c08e98b3176acc58bb7d5afa919096))
- Honor session focus_cluster_id, drop first-cluster fallback by @vellankikoti ([6e071b0](https://github.com/vellankikoti/kubilitics/commit/6e071b0a7131538d112afdf16d486d91088602a4))
- Teach LLM that list_resources handles namespaces by @vellankikoti ([53836e3](https://github.com/vellankikoti/kubilitics/commit/53836e3caa8eae1529bd033460cbebd85bf928c5))
- Summarize large list payloads so gpt-4o-mini can answer by @vellankikoti ([aac35fa](https://github.com/vellankikoti/kubilitics/commit/aac35fabf7322c701cbe0684d9afd58a8f444c23))
- Belt-and-braces string truncation on tool output by @vellankikoti ([10b91f3](https://github.com/vellankikoti/kubilitics/commit/10b91f3f1ce69093de8b4829222c59704ac2ed73))
- MaxTurns 10 -> 20 with KOTG_AGENT_MAX_TURNS override by @vellankikoti ([4da1197](https://github.com/vellankikoti/kubilitics/commit/4da11972cf35cc44f41afc7c22edb450d3d2c46e))
- Retry on 429 with hint-aware backoff by @vellankikoti ([3145f8e](https://github.com/vellankikoti/kubilitics/commit/3145f8e05b53909a2a559b29306f008445ddb123))
- Small-VM root disk 8GB -> 30GB by @vellankikoti ([43b7332](https://github.com/vellankikoti/kubilitics/commit/43b733201e1633cd4b9fef1693bf4f18362d0625))
- Stream=false must be explicit; accept object OR string tool args by @vellankikoti ([531e847](https://github.com/vellankikoti/kubilitics/commit/531e847655223901f59057c5e66b50a579a6f15c))
- Honest headline + INCOMPLETE badges + real latencies by @vellankikoti ([6bd3896](https://github.com/vellankikoti/kubilitics/commit/6bd38963478c94d0d5f50b46e2464dc8709b4325))
- Use plan-canonical plain-english seed names (20/20 match taxonomy) by @vellankikoti ([755b4e1](https://github.com/vellankikoti/kubilitics/commit/755b4e16280082c8ef7301a309c24bb83a68309b))
- Recover from json.Marshal panic in compactArgs by @vellankikoti ([7ad5a6f](https://github.com/vellankikoti/kubilitics/commit/7ad5a6f01a7b052345a999d2b18d460f0a1735a7))
- Retire 3 duplicate/loop-trap tools from MCP taxonomy by @vellankikoti ([e1d0a3e](https://github.com/vellankikoti/kubilitics/commit/e1d0a3e36305574f12520bec5e67820919d6b51e))
- Retire 4 legacy RBAC event observers superseded by who_can_do by @vellankikoti ([d9484fb](https://github.com/vellankikoti/kubilitics/commit/d9484fb89f183ae1e5d3b3b213def79d207cdadf))
- Read rollout changeCause camelCase by @vellankikoti ([1abfd70](https://github.com/vellankikoti/kubilitics/commit/1abfd70a6beae5f457d61d2ebf814cca3103dc91))
- Honor ctx cancellation during retry backoff by @vellankikoti ([331e1c6](https://github.com/vellankikoti/kubilitics/commit/331e1c6516c2cfcde2b7981fcedf713f83c515cd))
- DeriveSessionID rejects empty cluster_id by @vellankikoti ([46e28ad](https://github.com/vellankikoti/kubilitics/commit/46e28ad22aa6ce57a34dcd2efc52cfa84964c01b))
- Dispatch triage_cluster / list_problems / search_logs by @vellankikoti ([26fca36](https://github.com/vellankikoti/kubilitics/commit/26fca3621e98c4bd30afd1033f817f8de191089d))
- Isolated brain ports (:50071/:28081) to preserve desktop backend by @vellankikoti ([db779b1](https://github.com/vellankikoti/kubilitics/commit/db779b10dacef71e20d506be6e0e6a2992de52eb))
- Passthrough Flusher on response recorder (SSE fix) by @vellankikoti ([e8a234b](https://github.com/vellankikoti/kubilitics/commit/e8a234bda2c1629a975dd8a4bd4aecb7101840c2))
- PresenceEntryPoint waits for presence snapshot before redirecting by @vellankikoti ([1d15253](https://github.com/vellankikoti/kubilitics/commit/1d15253f994c2360c8bb50d059ece5b94ab70a44))
- Stop the loading-spinner-forever on stale backend by @vellankikoti ([3bb2a9f](https://github.com/vellankikoti/kubilitics/commit/3bb2a9f1b646d8bd28f191392c881d5083342039))
- Normalize backend wire-shape server_url \u2192 serverUrl by @vellankikoti ([c948f66](https://github.com/vellankikoti/kubilitics/commit/c948f66f4068879b50d2bdf5613c44f080879c5b))
- Auto-register kubeconfig clusters on click by @vellankikoti ([9ba1bdb](https://github.com/vellankikoti/kubilitics/commit/9ba1bdb0ab486d9c2788d3b642f510013912e31c))
- Pass real kubeconfig_path when auto-registering by @vellankikoti ([d010d93](https://github.com/vellankikoti/kubilitics/commit/d010d93f3dad4946a4b2a70b8f8e936badddbc6f))
- Working end-to-end click-to-register flow by @vellankikoti ([47a4f82](https://github.com/vellankikoti/kubilitics/commit/47a4f82fcb7c8795c68239b3cb876674f2a4cc56))
- Registered[] populates end-to-end — dedup merge + refresh on mutation by @vellankikoti ([be554ae](https://github.com/vellankikoti/kubilitics/commit/be554ae90d60966950e0586365f828a6e05ce05e))
- Page-inner truly responsive \u2014 kill the max-width cap by @vellankikoti ([a145dce](https://github.com/vellankikoti/kubilitics/commit/a145dce09e94ef7e2f95ff4a74a42c1c3eece469))
- Sidebar 0s + AI settings crash (permanent, tests locked) by @vellankikoti ([c5f7911](https://github.com/vellankikoti/kubilitics/commit/c5f79116f052f8ac977b09463ca391fc3450d580))
- ChatStore UnknownBlock test uses a genuinely unknown frame type by @vellankikoti ([0c0c5ec](https://github.com/vellankikoti/kubilitics/commit/0c0c5ecd6133aea59589af5ad98e4e4669fd1de5))
- Sweep all '&& backendBaseUrl' gates — '' means same-origin by @vellankikoti ([b2cb12e](https://github.com/vellankikoti/kubilitics/commit/b2cb12ed24840052ca6542bea9f08f90e6f89e2a))
- .gitignore was hiding brain source — release workflow could never build by @vellankikoti ([aac8065](https://github.com/vellankikoti/kubilitics/commit/aac80652dec74a9a410e4e8f0a808a93fb517f21))
- Populate go.sum /go.mod hashes for standalone (GOWORK=off) builds by @vellankikoti ([33f2e26](https://github.com/vellankikoti/kubilitics/commit/33f2e26fac1d40e3104ed0950be73a0a39483321))


### Misc

- Regenerate for v1.1.0 (#89) by @github-actions[bot] ([870393c](https://github.com/vellankikoti/kubilitics/commit/870393c2839c19081c1f4683d4263f17e46a9977))
- Rename module path to github.com/vellankikoti/kotg.ai/kubilitics-ai by @vellankikoti ([576bd72](https://github.com/vellankikoti/kubilitics/commit/576bd7224320c3eb9c6f7f7c45b9c9884a37ac22))
- Include earlier systematic-quality plan + count-tools diagnostic by @vellankikoti ([3ae846b](https://github.com/vellankikoti/kubilitics/commit/3ae846bcb5b8ec850cd59a088938e9ce024b4b13))
- Delete legacy k8s/cluster_discovery.go shim by @vellankikoti ([95ba462](https://github.com/vellankikoti/kubilitics/commit/95ba4626875f53ae0dc2b13573de7df2e9da3556))


### Tests

- Pre-migration UI smoke — 8 PASS, 1 PARTIAL, 1 known-FAIL by @vellankikoti ([8f00de8](https://github.com/vellankikoti/kubilitics/commit/8f00de8dcddf92403ef0336d1dd01ce5bcaedb45))
- Lock guardrails against sensitive K8s data leaks by @vellankikoti ([2d326ec](https://github.com/vellankikoti/kubilitics/commit/2d326ec6d62bf9978e33b00746e60f141374a881))
- Stabilize pre-1.1.0 flakes on CI by @vellankikoti ([ffb28d1](https://github.com/vellankikoti/kubilitics/commit/ffb28d147b51535b3556231e2ad8fc8430b1370a))
- Retarget tests to Week-1 composites after observe_* retirement by @vellankikoti ([bf0155d](https://github.com/vellankikoti/kubilitics/commit/bf0155d409b1584e2ea9f07d6619e8b93e6458ad))
- Assert switch-cluster navigates per FEATURE_PRESENCE_V2 by @vellankikoti ([dec0fcb](https://github.com/vellankikoti/kubilitics/commit/dec0fcb23b2f796ac7c864793096d0f61dff25d9))

## [1.1.0-rc.1] - 2026-04-23


### Added

- Add AIConfig with defaults (feature-flagged off) by @vellankikoti ([ab7de7c](https://github.com/vellankikoti/kubilitics/commit/ab7de7cd984531fe4dfb7192c41c567948440d46))
- SidecarStatus, CapabilitiesSnapshot, error sentinels by @vellankikoti ([4d2135f](https://github.com/vellankikoti/kubilitics/commit/4d2135fa42f4981625ebc7d5454434b06bc09345))
- ActionGate interface + NoOpGate (subproject 3 plugs the real gate) by @vellankikoti ([7ae50ff](https://github.com/vellankikoti/kubilitics/commit/7ae50ff2bd74c9cc2535da75219e9e8336232ae6))
- Per-spawn ephemeral CA + server + client cert mint by @vellankikoti ([ee09ef1](https://github.com/vellankikoti/kubilitics/commit/ee09ef1892f577a77f27dbe093099b43451ecffc))
- Goroutine-safe state machine with transition rules by @vellankikoti ([f19674c](https://github.com/vellankikoti/kubilitics/commit/f19674c9a31efc1b1f2bfb3ebcec2c251f77898d))
- Exponential backoff with restart cap and window reset by @vellankikoti ([ed58e7f](https://github.com/vellankikoti/kubilitics/commit/ed58e7f859a64c13c638c9dba5286a5415c8d7ac))
- Idle timer with reset and active-stream suppression by @vellankikoti ([a3d10b3](https://github.com/vellankikoti/kubilitics/commit/a3d10b31bc5dce3c61c9d5406185f32ea03ebc88))
- Spawn sidecar with stdin certs and READY <port> handshake by @vellankikoti ([16e1b8c](https://github.com/vellankikoti/kubilitics/commit/16e1b8cf187f0cba77053bee8cd63e751bccd78d))
- MTLS dial + grpc-health check + capabilities fetch by @vellankikoti ([95a787b](https://github.com/vellankikoti/kubilitics/commit/95a787b37dfeee4ff828d4a11fe542bbd98c650a))
- Orchestrator wiring spawn+dial+health+caps+idle+refresh by @vellankikoti ([8d1263d](https://github.com/vellankikoti/kubilitics/commit/8d1263d10c5c8601a9f37ed6fe8bec0deb7abe20))
- Per-user token-bucket rate limiter by @vellankikoti ([cfc01be](https://github.com/vellankikoti/kubilitics/commit/cfc01be5bbe85c09ddfc61db152cc289f26ea3be))
- Prometheus collectors for duration, errors, rate drops by @vellankikoti ([7c3c738](https://github.com/vellankikoti/kubilitics/commit/7c3c73825f555228be3cbd1bd4cc72452ccb40cd))
- Cluster_id enforce + metadata inject + SpawnID guard + rate limit by @vellankikoti ([b4e9586](https://github.com/vellankikoti/kubilitics/commit/b4e958698e3441b1122bb6fc0385e03a6d43e8a2))
- /api/v1/ai/status + /refresh with feature-flag gating by @vellankikoti ([46d9610](https://github.com/vellankikoti/kubilitics/commit/46d9610882db759c3f4839874a567707c6defc25))
- /capabilities — read-only default + ?warm + 400 on missing cluster by @vellankikoti ([5034561](https://github.com/vellankikoti/kubilitics/commit/5034561e705c379906d18effe2bea1c29cf29eee))
- /chat WS upgrade with cluster_id enforce + bidi proto-to-JSON pump by @vellankikoti ([b7e36d2](https://github.com/vellankikoti/kubilitics/commit/b7e36d29d0f834eded5456c33b48e5124df05446))
- Wire supervisor+proxy+handlers into server (gated on ai.enabled) by @vellankikoti ([68057ac](https://github.com/vellankikoti/kubilitics/commit/68057acd70eb9a8f4dd556400ca94dc9e7f2aaed))
- Surface ai.* config block (disabled by default) by @vellankikoti ([da97628](https://github.com/vellankikoti/kubilitics/commit/da9762890bb4f4c829d571580489cbe071ede92f))
- Plumb provider/endpoint/model/apiKeyEnv from cfg.AI through supervisor argv by @vellankikoti ([959d7c6](https://github.com/vellankikoti/kubilitics/commit/959d7c6686122a611d1f4af5f4767552b02845c9))
- POST /api/v1/ai/sessions for explicit session creation by @vellankikoti ([90f1857](https://github.com/vellankikoti/kubilitics/commit/90f185771d518acb8773e1d18cad474a472d3078))
- GRPCClient (Chat + AIControl) with bufconn-tested ensureOpen + Send/Capabilities by @vellankikoti ([b4682aa](https://github.com/vellankikoti/kubilitics/commit/b4682aa2b22f77308c763c073f1bd71f2a12758f))
- HTTPClient.GetStatus for control-plane queries by @vellankikoti ([48f193a](https://github.com/vellankikoti/kubilitics/commit/48f193af2e298b53c1ed679b2eab3b8add7afce2))
- Wire backend → kubilitics-ai via aiclient (gRPC+HTTP), drop supervisor bootstrap by @vellankikoti ([5ed7ab1](https://github.com/vellankikoti/kubilitics/commit/5ed7ab12d3cae5ae0bf428d4b93997ba62051cdc))
- Replace supervisor-era ai.* with endpoint/httpEndpoint pattern by @vellankikoti ([7338989](https://github.com/vellankikoti/kubilitics/commit/73389894ff29075be42f422fb18559bccf1fb0aa))
- ChatStore with transcripts + connection state + applyEvent reducer by @vellankikoti ([41999b9](https://github.com/vellankikoti/kubilitics/commit/41999b9bb518884bd5746fe178eb66bb72b3f656))
- AiContextStore with implicit/explicit precedence by @vellankikoti ([b1c599c](https://github.com/vellankikoti/kubilitics/commit/b1c599cb0af3cd23fd6a6c98ae27c0cfd1d8eff2))
- Protocol types + pure eventMapper for frame→Block translation by @vellankikoti ([1dce84c](https://github.com/vellankikoti/kubilitics/commit/1dce84cf162bc1e5424a54eeee74557f1f8730ea))
- WebSocket adapter with state callbacks + JSON frame parsing by @vellankikoti ([cb4f098](https://github.com/vellankikoti/kubilitics/commit/cb4f09862d9a0964ff0584e5e37a03f2532ac787))
- UseAIStatus (adaptive polling) + useAICapabilities (warm-aware) by @vellankikoti ([12eecea](https://github.com/vellankikoti/kubilitics/commit/12eecea0dae813b3d4ee4cba8c0dd1bc2167caf4))
- UseAIContext (page publish) + useChatHotkey (Cmd+I) by @vellankikoti ([e05e448](https://github.com/vellankikoti/kubilitics/commit/e05e448a0cbb347b68987b15a7091d82ee9375ef))
- UseChatController orchestrator (session/WS/send/cancel/newChat) by @vellankikoti ([263d420](https://github.com/vellankikoti/kubilitics/commit/263d42019aa59839d17b27463be633383d8b29b2))
- TextBlock (markdown+streaming cursor) + UnknownBlock fallback by @vellankikoti ([dbc35c5](https://github.com/vellankikoti/kubilitics/commit/dbc35c59166a99a298ebfe13eacb34d7bb07a704))
- Turn / UserMessage / SystemNotice with historical+error states by @vellankikoti ([537e0ce](https://github.com/vellankikoti/kubilitics/commit/537e0ce212b74f933379c703397dd631aab46069))
- ChatTranscript with react-virtual + auto-scroll-to-bottom by @vellankikoti ([2923454](https://github.com/vellankikoti/kubilitics/commit/29234545c5545626a716c631677b87c5f2f00780))
- ChatInput with Enter-to-send, Shift+Enter newline, prefill+select, Stop button by @vellankikoti ([c5df2b5](https://github.com/vellankikoti/kubilitics/commit/c5df2b5464d6d72acb58eb64a4cf155dbae69c85))
- ChatStatusPill + ChatHeader (new chat / copy / close) by @vellankikoti ([1b5fde6](https://github.com/vellankikoti/kubilitics/commit/1b5fde6783bbd1e590bc484939596804524acca8))
- ChatPanel container with capability gating + session-reset state by @vellankikoti ([6fe1295](https://github.com/vellankikoti/kubilitics/commit/6fe1295635afdc56e8ecc50f4bfd77c6037d29a4))
- AskAIButton (sets explicit context + prefill + opens panel) by @vellankikoti ([69b5a84](https://github.com/vellankikoti/kubilitics/commit/69b5a84d606db3516b840bb3e64efcb7bf45e179))
- Wire ChatPanel into AppLayout + Header pill + Cmd+I shortcut by @vellankikoti ([e25650a](https://github.com/vellankikoti/kubilitics/commit/e25650a47ced96de78097c506c5dbd0cef1bca0b))
- Sidebar 'AI Assistant' entry with status dot by @vellankikoti ([5ddf483](https://github.com/vellankikoti/kubilitics/commit/5ddf48331aa6747d990729c8b38a03538dfbb7e9))
- Per-row AskAIButton on Pods/Deployments/Nodes/Events lists by @vellankikoti ([2d2524e](https://github.com/vellankikoti/kubilitics/commit/2d2524eb7e609f2e24b6cbc53a86b1ed122becb2))
- Per-page AskAIButton + useAIContext publish on 6 detail pages by @vellankikoti ([4979e78](https://github.com/vellankikoti/kubilitics/commit/4979e78f6270d4bea8a30bef8e9055fef7bb0cc7))
- Vendor kubilitics-ai as sub-chart of parent kubilitics chart by @vellankikoti ([3af9495](https://github.com/vellankikoti/kubilitics/commit/3af9495784cd6b3ffad24d1a0f9335543cc4c45c))
- Terraform + Makefile to provision Ollama bench VM on AWS (g5.xlarge GPU, stop/start lifecycle) by @vellankikoti ([2406f45](https://github.com/vellankikoti/kubilitics/commit/2406f45de24485d4d7d208ff48069ff04c296341))
- AI provider config UI + validate connection + diagnostics by @vellankikoti ([c8da034](https://github.com/vellankikoti/kubilitics/commit/c8da0341d16021e637b50b98450c17d8bd4e7f70))
- Real /settings/ai/budget page + chat panel hardening by @vellankikoti ([0885d34](https://github.com/vellankikoti/kubilitics/commit/0885d34b47459673fbe9fb3ffded1bcf7f5125cd))
- Events serializer emits camelCase + involvedObject shape by @vellankikoti ([4625899](https://github.com/vellankikoti/kubilitics/commit/4625899c7a341d5d307d5eb59a777caad56a59df))
- /metrics/summary unscoped cluster aggregate by @vellankikoti ([056db89](https://github.com/vellankikoti/kubilitics/commit/056db89bb1d77819dfc81d70ad2933dbcd1e9f55))
- Ingress /tls-info subresource for cert expiry by @vellankikoti ([629a4c9](https://github.com/vellankikoti/kubilitics/commit/629a4c9df91b32d4fa497481cfd69403b8229ba8))
- Event bus + wire into setActiveCluster by @vellankikoti ([a8cb4bf](https://github.com/vellankikoti/kubilitics/commit/a8cb4bfef944c4df8e2fa2daea5798071e501c6a))
- AI Settings keychain round-trip commands by @vellankikoti ([d5dfbb1](https://github.com/vellankikoti/kubilitics/commit/d5dfbb1063a016eba66822778a46500d7e9bbc2d))
- AiConfigStore round-trips via Tauri keychain by @vellankikoti ([5002169](https://github.com/vellankikoti/kubilitics/commit/50021694f3246d10d954015de00d4c56ddc3b6fd))
- Subscribe 9 cache stores to cluster-switch event bus by @vellankikoti ([5f6d27a](https://github.com/vellankikoti/kubilitics/commit/5f6d27a95722d30983c40d01277bf9161518687b))
- WS banner + Tauri reset_budget/get_budget_status commands by @vellankikoti ([bfde4c1](https://github.com/vellankikoti/kubilitics/commit/bfde4c15f97943cce63a3ce3f55954c9d551d561))
- Phase 6 F.1 + F.2 — brain sidecar bundled + spawned by @vellankikoti ([91b8841](https://github.com/vellankikoti/kubilitics/commit/91b8841441f6715372563abd95a5c23ced7248c3))
- Phase 3 C.52 — Tool Catalog page by @vellankikoti ([3799e19](https://github.com/vellankikoti/kubilitics/commit/3799e19380bbe41607f0ced3a06dd9a1261134d4))
- Phase 6 F.4 — signed auto-update via tauri-plugin-updater by @vellankikoti ([7b6d94e](https://github.com/vellankikoti/kubilitics/commit/7b6d94ea7abfbc9dd5b6783a0eda8ab053d26b19))
- Phase 6 F.9 + F.12 — Homebrew formula + changelog automation by @vellankikoti ([abcd8bd](https://github.com/vellankikoti/kubilitics/commit/abcd8bd4eeb7b5dc7df9df4b219cff5f76b08f31))
- Paste-any-key always visible, prefix-detected by @vellankikoti ([f4960ea](https://github.com/vellankikoti/kubilitics/commit/f4960ead15b35b2eee8c2595575dfcf9858caf4f))


### Changed

- Drive from aiclient (gRPC + HTTP) instead of supervisor by @vellankikoti ([18043ff](https://github.com/vellankikoti/kubilitics/commit/18043ffdf806d76597d6a51c71466059351358cd))
- Drive WS chat pump + endpoints from aiclient via proxy by @vellankikoti ([e75c0b8](https://github.com/vellankikoti/kubilitics/commit/e75c0b83df33415b05ad9c8e01345713ee0cde8a))
- Keyring crate for cross-platform keychain (Option A) by @vellankikoti ([39786e1](https://github.com/vellankikoti/kubilitics/commit/39786e1435e47084bd4b276a29027b9ffe70e65d))
- Scope debounce state to controller instance by @vellankikoti ([511e185](https://github.com/vellankikoti/kubilitics/commit/511e18566dcbd80d100ae96cb14121f8937b3340))


### Documentation

- AI capabilities summary for technical investors by @vellankikoti ([16524f5](https://github.com/vellankikoti/kubilitics/commit/16524f5fd29a23c5bdb14924346f4c74c10fb2b2))
- Real Ollama qwen2.5:3b CPU numbers + cold-load tax + one-line fix paths by @vellankikoti ([29a4826](https://github.com/vellankikoti/kubilitics/commit/29a4826f0f447b9adf6b0b697c7c818709bd90a9))
- Add 166-tool LLM-direct path coverage matrix by @vellankikoti ([b66b979](https://github.com/vellankikoti/kubilitics/commit/b66b979bd76db4811db026f1ffa5882f4301a1a5))
- Comprehensive 498-case tool coverage benchmark results by @vellankikoti ([0194b08](https://github.com/vellankikoti/kubilitics/commit/0194b081b92c778c7d7fee5ede73948d63b18728))
- Comprehensive bench v2 — alias expansion + autonomy + bug fix by @vellankikoti ([c504dbe](https://github.com/vellankikoti/kubilitics/commit/c504dbe321b4c50717070ea1962d4bbc95927c3e))
- Critical review — what's shipped vs honest gaps to best-on-earth K8s AI by @vellankikoti ([c08bdb0](https://github.com/vellankikoti/kubilitics/commit/c08bdb04f7f83bb7660531396a920cd762648d28))
- E2e findings 2026-04-20 against docker-desktop by @vellankikoti ([e7d6bf9](https://github.com/vellankikoti/kubilitics/commit/e7d6bf9cc5564a0340e04b939e63a03e6d425658))
- Mark P0 #1-#3 fixed, P0 #4 deferred to proto unification by @vellankikoti ([b000713](https://github.com/vellankikoti/kubilitics/commit/b0007134ec6b8ea6e3cb345f661b1abb2dc7d518))
- Note audit correlation ID + addon-audit DELETE fixes by @vellankikoti ([dfbb4d0](https://github.com/vellankikoti/kubilitics/commit/dfbb4d09d041459576d2e5536bd1fa45cb6cb5e6))


### Fixed

- Mount AI routes on apiRouter (was 404'd by subrouter NotFoundHandler) by @vellankikoti ([5434a9b](https://github.com/vellankikoti/kubilitics/commit/5434a9b9577a3c0bcb01623cbf521bee86efc7a5))
- Viper BindEnv for nested ai.* keys + STUB_*_ADDR env in stub for port flexibility by @vellankikoti ([cf60c4f](https://github.com/vellankikoti/kubilitics/commit/cf60c4fb969819e28ea029a688b35bea5a9c083d))
- Adapt to subproject 3a contract changes by @vellankikoti ([f7560a5](https://github.com/vellankikoti/kubilitics/commit/f7560a5b82343c435e269a4cbe743af2db9e9824))
- Standardize backend HTTP port to 8190 (was stale 819) by @vellankikoti ([20a392a](https://github.com/vellankikoti/kubilitics/commit/20a392a46236ee8334de7eb71b088a529f41391e))
- Export HOME for ollama pull in cloud-init by @vellankikoti ([2ba1ad2](https://github.com/vellankikoti/kubilitics/commit/2ba1ad2d7a7b803993d0cc4c04dc3773548af8f8))
- Headlamp-style sync — kills sidebar-0s + chat wrong-cluster-id by @vellankikoti ([d42a94e](https://github.com/vellankikoti/kubilitics/commit/d42a94ea4db06f1273fa36b990fdbf039c0b31e0))
- Remove duplicate hook; enhance existing restore with (name, serverUrl) identity by @vellankikoti ([ed703e7](https://github.com/vellankikoti/kubilitics/commit/ed703e737801fc83fffeba098593c137996dd93c))
- Clear empty-state when AI is not ready (not configured / unreachable / key missing) by @vellankikoti ([d0f7b50](https://github.com/vellankikoti/kubilitics/commit/d0f7b5046c01da006667ebff65cbeb9be329fa22))
- Disable Configure-budget button (was 404'ing on stub route) by @vellankikoti ([32ad3ed](https://github.com/vellankikoti/kubilitics/commit/32ad3ed45982422f8b564981a60eb24be99f47ec))
- Persist + reload saved config; relax chat-panel models check by @vellankikoti ([d187077](https://github.com/vellankikoti/kubilitics/commit/d187077b1c6c098a13dfac6c56b3d83df65427d4))
- Real tool/action/plan/citation renderers — replace "UI support coming soon" stubs by @vellankikoti ([e2bd7ff](https://github.com/vellankikoti/kubilitics/commit/e2bd7ff0eb2fc0e74fa71acc583823f500cbe186))
- Page-error guard + finalize stuck tool spinners on turn completion by @vellankikoti ([d301afa](https://github.com/vellankikoti/kubilitics/commit/d301afab29430f7a46d31953dd25055aeb22cdfa))
- Permanent robustness for cluster-unreachable counters by @vellankikoti ([91d6970](https://github.com/vellankikoti/kubilitics/commit/91d6970f39e3fb87cea5c93e679d3b96d11b4e37))
- Flatten AssistantEvent payload so frontend can read fields by @vellankikoti ([93f8fab](https://github.com/vellankikoti/kubilitics/commit/93f8fab530a102600a9cb519edeab1cf5b5a6d91))
- Gate input on user-saved AI config, not just brain readiness by @vellankikoti ([0fdf387](https://github.com/vellankikoti/kubilitics/commit/0fdf387fbda3f9cc8795f7e4e14f581a3e9e9073))
- Bypass per-client circuit breaker for node/namespace count by @vellankikoti ([2a435fe](https://github.com/vellankikoti/kubilitics/commit/2a435fe86092353caee490e261450ba46b2595f4))
- Header AI dot colors reflect real readiness by @vellankikoti ([3573daa](https://github.com/vellankikoti/kubilitics/commit/3573daa48987912abefe59ef8a36e62535ef921b))
- Readable text + selection colors by @vellankikoti ([4c286a9](https://github.com/vellankikoti/kubilitics/commit/4c286a98df1606662c4587bbcd905a9c36470385))
- Default OpenAI model to gpt-4o, not gpt-4o-mini by @vellankikoti ([6fdaddc](https://github.com/vellankikoti/kubilitics/commit/6fdaddc954bbb23ee9c9515e04604882495e2855))
- GRPC + REST simultaneous bind (ephemeral port support) by @vellankikoti ([537f485](https://github.com/vellankikoti/kubilitics/commit/537f485c27b0fcc071cc43a15e5847dc4133c43a))
- Debounce non-secret field saves (500ms) by @vellankikoti ([78972fb](https://github.com/vellankikoti/kubilitics/commit/78972fbea9bf187b97c452e942fe565348fc8d99))
- Migrate AISettingsPage to useAIConfigStore by @vellankikoti ([2222e15](https://github.com/vellankikoti/kubilitics/commit/2222e15d2e24d88acbdc39f042ea5d977978c0d3))
- Render BudgetExceededBanner when chat stream emits budget_exceeded by @vellankikoti ([23ab2d0](https://github.com/vellankikoti/kubilitics/commit/23ab2d059cf775b283f960fe0ed32f255c3e204b))
- Validate provider enum + model + base_url in save_ai_config by @vellankikoti ([d9d15ad](https://github.com/vellankikoti/kubilitics/commit/d9d15add2ad52ac098a88d04772b5fcd322e3d31))
- BudgetExceededBanner clears after successful reset by @vellankikoti ([b260666](https://github.com/vellankikoti/kubilitics/commit/b260666498f0a1f8e807941e1143ec55f76af0cb))
- Resolve 10 pre-existing lint errors blocking CI by @vellankikoti ([f344717](https://github.com/vellankikoti/kubilitics/commit/f344717bcff76d670f9b26d6ab51d387ca88e260))
- Unblock PR #87 — Go errcheck + desktop-ci stub for brain sidecar by @vellankikoti ([ec59cd2](https://github.com/vellankikoti/kubilitics/commit/ec59cd2a4451d6f5b222edef80e410973bf3bbe1))
- Unblock cluster delete + one-click AI connect by @vellankikoti ([2adaaf2](https://github.com/vellankikoti/kubilitics/commit/2adaaf23514c7ba63ffdefcae750cb5ae77dc969))
- 9 more errcheck sites for CI by @vellankikoti ([026dc5a](https://github.com/vellankikoti/kubilitics/commit/026dc5ab510aa9207528f7a9131bece974318982))
- Detect_available_providers async — no nested tokio by @vellankikoti ([31d786d](https://github.com/vellankikoti/kubilitics/commit/31d786d0529e172fc68d5b9aefa8df9b0303c76a))
- Restart brain after save_ai_config so new key takes effect by @vellankikoti ([96b8fb3](https://github.com/vellankikoti/kubilitics/commit/96b8fb3b934f5a28bd2b82c8b8218812f3cba9ee))
- Brain actually reads saved config + key by @vellankikoti ([3a9b5d3](https://github.com/vellankikoti/kubilitics/commit/3a9b5d31b1f0cf564352f14f45d23f2481279181))
- Stop keychain dialog storm on every brain respawn by @vellankikoti ([89026b8](https://github.com/vellankikoti/kubilitics/commit/89026b8428a09e45a4586fe7c829d9b6df838ce6))
- Enable backend /api/v1/ai/* routes in desktop mode by @vellankikoti ([2298a85](https://github.com/vellankikoti/kubilitics/commit/2298a85b6be927a8abba041936b485886c0afc39))
- /health SPA route was swallowed by Vite backend proxy by @vellankikoti ([19f0df5](https://github.com/vellankikoti/kubilitics/commit/19f0df55c105f44936ce85a8cb78124304a7865b))
- Rust keychain tests — custom in-memory credential builder by @vellankikoti ([851105f](https://github.com/vellankikoti/kubilitics/commit/851105fba2dcba6ca5f85b02beb89b3fafba9e1c))
- Resolve 12 golangci-lint issues blocking 1.1.0 CI by @vellankikoti ([74c98ef](https://github.com/vellankikoti/kubilitics/commit/74c98efdefa6fe2cf11175aea71425d5f0f92edb))
- Brain source path — kotg.ai has no nested subdirectory by @vellankikoti ([e885d35](https://github.com/vellankikoti/kubilitics/commit/e885d3552ca800e147875277917a07b8c4d52d40))
- Version gate accepts prerelease tags by @vellankikoti ([7b3e115](https://github.com/vellankikoti/kubilitics/commit/7b3e1159ad2a0c82979e6deacc26469c13c3a991))
- Aux workflows skip prerelease tags by @vellankikoti ([a8d962a](https://github.com/vellankikoti/kubilitics/commit/a8d962a95986605f69b1e44568fa7fc1a2086e83))
- Brain source IS nested — revert earlier wrong fix by @vellankikoti ([ab9024c](https://github.com/vellankikoti/kubilitics/commit/ab9024c677943de734f2d943dc439267c89bd12e))
- GOWORK=off for brain builds by @vellankikoti ([008c510](https://github.com/vellankikoti/kubilitics/commit/008c5104e544515031fa48e1f81916473f7a235c))


### Misc

- Archive supervisor + certs + gate behind //go:build never (build broken; T9-T11 fix) by @vellankikoti ([e4f5bf7](https://github.com/vellankikoti/kubilitics/commit/e4f5bf7313c7b5a472beea4cf689de2a0a32642e))
- Default to t3.large CPU + qwen2.5:3b for cost (~$0.083/hr) by @vellankikoti ([6c15c97](https://github.com/vellankikoti/kubilitics/commit/6c15c971d0e8177faae21a29c89f96f6706f557b))
- Bump version to 1.1.0 + finalize updater pubkey by @vellankikoti ([de98cfe](https://github.com/vellankikoti/kubilitics/commit/de98cfe220e59f0f89e699227c48c7191adb6863))


### Tests

- Stub sidecar binary that performs full handshake by @vellankikoti ([cbd4694](https://github.com/vellankikoti/kubilitics/commit/cbd4694d995bb1e11aed43467fe760210b98f4db))
- Stub kubilitics-ai (echo) for manual smoke by @vellankikoti ([e312bc7](https://github.com/vellankikoti/kubilitics/commit/e312bc7ce3d371dca206963b437d374ad1c54361))
- Lock Event serializer contract for kubilitics-ai by @vellankikoti ([83424d8](https://github.com/vellankikoti/kubilitics/commit/83424d8ad1ec6b12c3d8093cbc43d381db751b60))
- E2E keychain round-trip regression by @vellankikoti ([c31f5ed](https://github.com/vellankikoti/kubilitics/commit/c31f5ed4e1399649d2503d348eb66919ed727952))
- Align addon audit test with migration 052 by @vellankikoti ([0a9e9a2](https://github.com/vellankikoti/kubilitics/commit/0a9e9a275e40cfe2b0928f549449b6f460e1e464))

## [1.0.0] - 2026-04-18


### Added

- Migration 051 + agent models by @vellankikoti ([44c217e](https://github.com/vellankikoti/kubilitics/commit/44c217e2b0fcd870c0415aa7dc86629748cff9b4))
- Bootstrap + access JWT signer by @vellankikoti ([61e50e5](https://github.com/vellankikoti/kubilitics/commit/61e50e5b7e0fec499ab2a783bb80bca747a2d3d0))
- Refresh token + argon2id hashing by @vellankikoti ([e46b2a1](https://github.com/vellankikoti/kubilitics/commit/e46b2a102ca5307dff1ad763f265a0cc0284d3f1))
- Repository for agent_clusters, bootstrap_tokens, credentials by @vellankikoti ([b4ba66b](https://github.com/vellankikoti/kubilitics/commit/b4ba66b8f33917e33f01852d61f3e1e7c1cedc08))
- TokenReview wrapper by @vellankikoti ([e6b9c55](https://github.com/vellankikoti/kubilitics/commit/e6b9c55dba5fc4c4aec9e5f1a0553636769eee5c))
- /agent/register handler (bootstrap + SA paths) by @vellankikoti ([7a7f3aa](https://github.com/vellankikoti/kubilitics/commit/7a7f3aa2004eec2a56884fe1cf4ef1b9de2804bf))
- /agent/token/refresh handler by @vellankikoti ([524688a](https://github.com/vellankikoti/kubilitics/commit/524688ac0558fc612c5f2d7353cacdddae5a7fc1))
- /agent/heartbeat handler + access verification by @vellankikoti ([ecde174](https://github.com/vellankikoti/kubilitics/commit/ecde174f412623751861ea21ed9ce1fb6c024ced))
- Admin endpoint to mint bootstrap tokens by @vellankikoti ([dc657dd](https://github.com/vellankikoti/kubilitics/commit/dc657ddf969be2427a8c398ae983208eab6c181a))
- Wire agent handlers into hub server by @vellankikoti ([6b4ca52](https://github.com/vellankikoti/kubilitics/commit/6b4ca52f482c1e2fd34d0e717f100d79efe76826))
- Module skeleton, config, credstore, cluster-uid by @vellankikoti ([effe6f1](https://github.com/vellankikoti/kubilitics/commit/effe6f170577ee10f81920eea8d4cc836918a92c))
- Hub HTTP client with TLS pinning by @vellankikoti ([f26a4b4](https://github.com/vellankikoti/kubilitics/commit/f26a4b49dde196e0aaf25740f5add58f1679c584))
- Bootstrap + heartbeat loop + main wiring by @vellankikoti ([21a620e](https://github.com/vellankikoti/kubilitics/commit/21a620e00cc32994a710d8e5266b419a54c6e6e5))
- Dockerfile and Helm chart by @vellankikoti ([ceb7ba7](https://github.com/vellankikoti/kubilitics/commit/ceb7ba79d2390be15996181816662799935db89b))
- Hub signing-secret + env wiring by @vellankikoti ([cb7b2b6](https://github.com/vellankikoti/kubilitics/commit/cb7b2b6f0723bf671ac830a7c04683ad6a30aa7d))
- Per-cluster rate limit on agent heartbeat endpoint by @vellankikoti ([27a1f43](https://github.com/vellankikoti/kubilitics/commit/27a1f431fcef9aa41dffaa04fd10f4f419454af1))
- Hub self-registers + browser onboarding bypassed when clusters exist by @vellankikoti ([c963ac5](https://github.com/vellankikoti/kubilitics/commit/c963ac5f0598c369202db72c06e13255e9a0f0c0))
- Chart default authMode=disabled — open URL, see insights, done by @vellankikoti ([1cc8831](https://github.com/vellankikoti/kubilitics/commit/1cc8831797650707c67786ca75d13811fad31d92))
- Add ColumnConfig.sortValue for sort-only data by @vellankikoti ([7c69ec1](https://github.com/vellankikoti/kubilitics/commit/7c69ec1118a237ef857feec8a0ff57f9e9252153))
- Backend + client now sort cross-page on every meaningful column by @vellankikoti ([3c9d72a](https://github.com/vellankikoti/kubilitics/commit/3c9d72ac77bfbbca81232da56e1580643555f1e0))


### Documentation

- Seamless install instructions + add kubilitics-agent README by @vellankikoti ([b24e673](https://github.com/vellankikoti/kubilitics/commit/b24e67351972c6769a2ec44bce884d5aa81852d6))


### Fixed

- Code review — Organization struct, db/json tags, rows.Err check by @vellankikoti ([bc0259a](https://github.com/vellankikoti/kubilitics/commit/bc0259a2a378a993388a22309b6050a2458dc0ff))
- Defensive secret copy, require exp, type-confusion tests by @vellankikoti ([9e29980](https://github.com/vellankikoti/kubilitics/commit/9e299808fc694c9ff9f084c73619a5d01b207d19))
- Remove dead ErrRefreshFormat, length-check decoded salt/hash, edge tests by @vellankikoti ([adb2323](https://github.com/vellankikoti/kubilitics/commit/adb23238adbe1bd41bceccc41729e2fb8d3d7768))
- Check RowsAffected error, add replay + not-found tests by @vellankikoti ([6f65993](https://github.com/vellankikoti/kubilitics/commit/6f65993ab78152e32f4940decbd25cabe797a5eb))
- Persist epoch bump, propagate token-mint errors, anchor test paths, add 3 register tests by @vellankikoti ([c8c73fe](https://github.com/vellankikoti/kubilitics/commit/c8c73fef80d09fd14bc041aa7193a4b667edbaa9))
- Document expiry-check defense, assert 401 error codes by @vellankikoti ([8ee868c](https://github.com/vellankikoti/kubilitics/commit/8ee868c96b993c782d76699f081b07eaf4a39446))
- Production-blockers — auth bypass for agent paths, agent re-registration loop, admin auth, epoch bump race by @vellankikoti ([0ddc488](https://github.com/vellankikoti/kubilitics/commit/0ddc48868688439f3e23a417fda223632c4a7427))
- E2e fixes uncovered during release audit by @vellankikoti ([b6cfab6](https://github.com/vellankikoti/kubilitics/commit/b6cfab6ddada5ed75165b2ef9632aafbfd612695))
- Permanently fix "Update Failed" — register process plugin + honest UX by @vellankikoti ([bb5de08](https://github.com/vellankikoti/kubilitics/commit/bb5de08b5c9e4303a14ba9d29842f8ad3f8542b3))
- Skip MSI bundler on prerelease tags (numeric-only constraint) by @vellankikoti ([d3870b3](https://github.com/vellankikoti/kubilitics/commit/d3870b3c768e30996b84db90273ad4290521fc42))
- Restore multi-tab terminal in workload detail pages by @vellankikoti ([838a965](https://github.com/vellankikoti/kubilitics/commit/838a965f2f67ccb6ddcfe35db3757981db508d9b))
- In-cluster browser deployment — same-origin backend + honest wizard copy by @vellankikoti ([5f29120](https://github.com/vellankikoti/kubilitics/commit/5f29120a72adec657244de4dc0a0d497062454c7))
- Production browser counts as 'backend configured' by @vellankikoti ([4b68b9b](https://github.com/vellankikoti/kubilitics/commit/4b68b9bddae30980c8f78c80cea67988683a1d15))
- Cluster service understands Source='in-cluster' by @vellankikoti ([fb528c7](https://github.com/vellankikoti/kubilitics/commit/fb528c78593f430e29c653a6ea5ff5dae2eef10f))
- Notification bell — square icon button, no inline label, no idle dot by @vellankikoti ([14a1798](https://github.com/vellankikoti/kubilitics/commit/14a179866eaea11c9f3a20fa872ab16b9e8514c8))
- Sort 'Age' columns by creation timestamp, not formatted string by @vellankikoti ([509b582](https://github.com/vellankikoti/kubilitics/commit/509b582e2f9c3ff7e2ea9eaceed81e20839d8006))
- Push column sort to server so pagination respects user sort by @vellankikoti ([79ed3a1](https://github.com/vellankikoti/kubilitics/commit/79ed3a1a5342f206eba06c5f11815b6e036c2347))
- Server sort + newest-first default for ALL paginated list pages by @vellankikoti ([3060e70](https://github.com/vellankikoti/kubilitics/commit/3060e70c654313392e1517178df5024604cca288))
- Cross-page sort works on direct K8s path too by @vellankikoti ([4a3d618](https://github.com/vellankikoti/kubilitics/commit/4a3d61869c8c723d27dc4401133d26f68727ce68))


### Misc

- Release v1.0.0-rc.1 by @vellankikoti ([694b168](https://github.com/vellankikoti/kubilitics/commit/694b1684d620845205402cc8bf5010ca72e305e1))
- Release v1.0.0 by @vellankikoti ([aadb745](https://github.com/vellankikoti/kubilitics/commit/aadb745b83e06686397dfaf7e3e38c8730319889))


### Tests

- Heartbeat epoch_mismatch coverage by @vellankikoti ([bb375e4](https://github.com/vellankikoti/kubilitics/commit/bb375e49f74c7ce6e985cb1f7b14e821d80cf13b))
- Kind-based e2e for same-cluster registration by @vellankikoti ([199e62f](https://github.com/vellankikoti/kubilitics/commit/199e62f2df2d872da22c12426fca09a97b809228))
- Add GetOrReconnectClient stubs to addon test mocks by @vellankikoti ([bb3b2dd](https://github.com/vellankikoti/kubilitics/commit/bb3b2dd8d7a0ff68a27d9b306ef6f284f235008c))

## [0.3.0] - 2026-04-17


### Added

- CSV export with enriched K8s resource details for topology and blast radius by @vellankikoti ([f2b46d9](https://github.com/vellankikoti/kubilitics/commit/f2b46d9c7f3ac1c645340aa836ef4b7c7a978924))
- Add CSV export to resource-scoped topology view by @vellankikoti ([151fcd3](https://github.com/vellankikoti/kubilitics/commit/151fcd3210a6b7e4f89b22d32196734cef1a5c15))
- Add export to Events and Gateways list pages — 52/52 coverage by @vellankikoti ([289ea2a](https://github.com/vellankikoti/kubilitics/commit/289ea2ac224140ac3af532ce9f457d029142e098))
- Seamless auto-updates — Slack/Docker-style background download + restart by @vellankikoti ([5b83464](https://github.com/vellankikoti/kubilitics/commit/5b834641b2b75fb19115394058f744bbe14d6d04))
- Slack/Docker-style auto-update banner with release notes and progress bar by @vellankikoti ([52679ad](https://github.com/vellankikoti/kubilitics/commit/52679ad77da609b4a5f98cfe9ab6068c36bebb2e))
- Upgrade resource creation templates to production-ready with examples by @vellankikoti ([d31b043](https://github.com/vellankikoti/kubilitics/commit/d31b043a82a6b951617b79b9203b3b11eecaf580))
- Fleet groups visually separate clusters with colored sections by @vellankikoti ([33ef614](https://github.com/vellankikoti/kubilitics/commit/33ef61490d1a9caa6e88423843eff3f498828eb6))
- Proper group CRUD — dialog with cluster selection, delete button by @vellankikoti ([cdde02c](https://github.com/vellankikoti/kubilitics/commit/cdde02cdf11400bc0e5374a64a0a0f4474efb7ed))
- Redesign ContextPicker — split layout, larger logo, compact cards by @vellankikoti ([81e4b63](https://github.com/vellankikoti/kubilitics/commit/81e4b63ff80b9cfb941908e5f62befaeb92bebb8))
- Topology canvas — scroll to pan when zoomed in by @vellankikoti ([2f9de01](https://github.com/vellankikoti/kubilitics/commit/2f9de01cd9a4e63e39a0c0a7440dc9804c6826cf))
- Overhaul blast radius engine — failure modes, fixed %, remediation engine (T2/T4/T5) by @vellankikoti ([943bfa3](https://github.com/vellankikoti/kubilitics/commit/943bfa3b8740cd455d1121589d5c6ba2917d4f2e))
- Add structural health scores and namespace risk ranking (T6, T8) by @vellankikoti ([421dadb](https://github.com/vellankikoti/kubilitics/commit/421dadb95d5c9f09f1eadd6e69d1f4b9ba55d6aa))
- Add SPOF inventory backend (T7) — detect, enrich, and expose single points of failure by @vellankikoti ([0d10c15](https://github.com/vellankikoti/kubilitics/commit/0d10c15be641917697df848b3900d1e18371aca7))
- Add topology diffing, cross-NS viz, health/risk/SPOF/diff frontend by @vellankikoti ([1d20572](https://github.com/vellankikoti/kubilitics/commit/1d2057226e791161deb1b8425b5a3551fdf35c7f))
- Add compliance mapping engine — CIS v1.8 + SOC2 (T14) by @vellankikoti ([c3f5b29](https://github.com/vellankikoti/kubilitics/commit/c3f5b29b7c879ae0c7fb4d4fc2b329e1527f1fc2))
- Add pre-apply blast radius, resilience reports, scheduled reports (T10, T11, T12) by @vellankikoti ([f8530d6](https://github.com/vellankikoti/kubilitics/commit/f8530d65813480e017dbf29f52092e39093bfb80))
- Wire intelligence pages into navigation — routes, sidebar, tabs by @vellankikoti ([1fe1d74](https://github.com/vellankikoti/kubilitics/commit/1fe1d7419dca14b98948b523c424d53b9df9d136))
- Extract resilience PDF renderer into dedicated module by @vellankikoti ([fbc2d21](https://github.com/vellankikoti/kubilitics/commit/fbc2d21d93754eae7af48068b1c766ed8612dd5e))
- Add SQLite-backed topology snapshot store (Gap 3) by @vellankikoti ([680d0e0](https://github.com/vellankikoti/kubilitics/commit/680d0e053383b416ba9423bd29a80b180842d2e3))
- Pillar 3 simulation engine + database migrations for Pillars 4-5 by @vellankikoti ([4c71e7d](https://github.com/vellankikoti/kubilitics/commit/4c71e7d6e2556259c879ec222dc1c43070bd07a0))
- Pillar 3 simulation frontend + Pillar 4 auto-pilot backend by @vellankikoti ([f6aecf8](https://github.com/vellankikoti/kubilitics/commit/f6aecf8bc8f9ec7e21fb1c9cad1c3b0461742389))
- Pillar 4 auto-pilot frontend + Pillar 5 fleet x-ray backend by @vellankikoti ([bab9d10](https://github.com/vellankikoti/kubilitics/commit/bab9d1043f214918fa09e79e1fde4bc771d38d7a))
- Pillar 5 fleet x-ray frontend — complete 3-pillar implementation by @vellankikoti ([6b6d9a8](https://github.com/vellankikoti/kubilitics/commit/6b6d9a8d518457db1b4ff67bd983b28a0adc5cdd))
- Add ListPagination to HealthDashboard namespace table by @vellankikoti ([71c81b9](https://github.com/vellankikoti/kubilitics/commit/71c81b93b3ca14035b8c27997bb23070ca1c7eb5))
- Add ListPagination to RiskRanking namespace table by @vellankikoti ([e4e5660](https://github.com/vellankikoti/kubilitics/commit/e4e56605e43558327a0a873be9a0c7744124b0d9))
- Upgrade SPOFInventory pagination to use ListPagination component by @vellankikoti ([6321eee](https://github.com/vellankikoti/kubilitics/commit/6321eee6e963aa8fdab00bd1f7986bb70e9d3a33))
- Simulation visual system — 6 purpose-built visualization components by @vellankikoti ([cf422c2](https://github.com/vellankikoti/kubilitics/commit/cf422c2061cf9235ea7898c2d4257120401deee6))
- Upgrade SimulationPage to use visual system components by @vellankikoti ([35ce079](https://github.com/vellankikoti/kubilitics/commit/35ce07999a573b7a3859b0b6692821da4617486a))
- DevOps infrastructure — Sentry, Semgrep PR gating, release hardening, layout consistency by @vellankikoti ([4ccc22b](https://github.com/vellankikoti/kubilitics/commit/4ccc22b1e6828c2c3c8a516439fc25301a0c4bc2))
- Observability 2.0 — Events Intelligence, OTel Traces, UI Polish by @vellankikoti ([6aa5fe8](https://github.com/vellankikoti/kubilitics/commit/6aa5fe80c9c835fb5daaf11ada3de8875f4c8b17))
- Multi-terminal + state persistence (Lens-style) by @vellankikoti ([eed046c](https://github.com/vellankikoti/kubilitics/commit/eed046cbfea95637079c3e1ac99aea2fdc650593))
- Multi-session cluster shell panel (Lens-style) by @vellankikoti ([0f59f71](https://github.com/vellankikoti/kubilitics/commit/0f59f71a3467735274f5de8a912b6042ffbfefd0))
- Metrics trend shows time delta — "vs 30s ago" helper text by @vellankikoti ([6515f0e](https://github.com/vellankikoti/kubilitics/commit/6515f0ef62a8b43d477bde59500dfd9a0e2e26c0))
- Metrics page — enterprise reorder + clean value formatting by @vellankikoti ([02737ab](https://github.com/vellankikoti/kubilitics/commit/02737ab19893dd553b57ad9968382187ee602fae))
- Dynamic cluster sizing — auto-tune settings based on pod count by @vellankikoti ([debce5d](https://github.com/vellankikoti/kubilitics/commit/debce5d722a6ce785b5562247c67032244ade916))
- Enterprise hardening — server-side pagination, performance optimization, concurrency safety by @vellankikoti ([501e099](https://github.com/vellankikoti/kubilitics/commit/501e09943545747af87c3ce1eeb5e3f319620d0a))
- Brand watermark on all fullscreen/presentation views by @vellankikoti ([2c2e18b](https://github.com/vellankikoti/kubilitics/commit/2c2e18b62bbf2d6d8e848c66cb5dfde17a0a3cc3))
- Intelligence engine rewrite — incidents, insights, causal chains all working by @vellankikoti ([549d3fd](https://github.com/vellankikoti/kubilitics/commit/549d3fd99c0fca5cbf7552cca3d386f942c716c3))
- Enterprise notification pipeline — rate limit, grouping, cooldown, dedup by @vellankikoti ([d281707](https://github.com/vellankikoti/kubilitics/commit/d2817072b7098322519bb0e02b82274668fc8247))
- Built-in trace generator — auto-generates realistic OTel traces from cluster workloads by @vellankikoti ([9e24c27](https://github.com/vellankikoti/kubilitics/commit/9e24c27e0b8aa3171585ed04031e49c6d647aaff))
- Real OTel integration — remove fake traces, add setup guide + addon by @vellankikoti ([22f219c](https://github.com/vellankikoti/kubilitics/commit/22f219c49957b221a3dc7ce64d674128b5584ede))
- Go module + SQLite store with spans/traces schema by @vellankikoti ([2bf5547](https://github.com/vellankikoti/kubilitics/commit/2bf5547c55340372ecb904824b97ca6220a953fe))
- OTLP receiver + REST query API + Dockerfile by @vellankikoti ([7c33fb6](https://github.com/vellankikoti/kubilitics/commit/7c33fb6d41edb8dc4d6321eafe21bc011b12e25e))
- Tracing backend — agent manifests + puller + enable/disable/instrument API by @vellankikoti ([149ab30](https://github.com/vellankikoti/kubilitics/commit/149ab306468a0ccec6051d7f948daa31f9d1ace4))
- Tracing setup UI — enable wizard + deployment picker + status badge by @vellankikoti ([c2b7283](https://github.com/vellankikoti/kubilitics/commit/c2b728334da29cb957afafe0351cffcefd2260b7))
- One-click tracing — auto-installs cert-manager + OTel Operator + agent by @vellankikoti ([ce8637d](https://github.com/vellankikoti/kubilitics/commit/ce8637d0a2804798061085501c21e94000d33e6a))
- Real distributed tracing — demo app with OTel SDK generates live traces by @vellankikoti ([5b0d597](https://github.com/vellankikoti/kubilitics/commit/5b0d597bb1c2331c906c9db028f177a8a723fd59))
- Trace puller working — PullNow triggers on status check + bootstrap on startup by @vellankikoti ([b2459c7](https://github.com/vellankikoti/kubilitics/commit/b2459c744d31cc3b95a9dc56d0e9d0d87434f987))
- Add v2 data models — sub-scores, impact classification, audit trail by @vellankikoti ([0174df0](https://github.com/vellankikoti/kubilitics/commit/0174df0059c4f3a538457d57bbc6f91ad99a0645))
- Add critical system component definitions by @vellankikoti ([45d4798](https://github.com/vellankikoti/kubilitics/commit/45d4798c58f52359d9523b512f89ddb000d2df6d))
- Add composite scoring model — resilience, exposure, recovery, impact by @vellankikoti ([3162656](https://github.com/vellankikoti/kubilitics/commit/3162656dbdc5c9a423d5d56aa77104014a6ea21b))
- Add natural language verdict generator by @vellankikoti ([aa15dd7](https://github.com/vellankikoti/kubilitics/commit/aa15dd70f03e097f0822e964700a6e6f8653bc68))
- Add impact classification engine — broken/degraded/self-healing by @vellankikoti ([2718d96](https://github.com/vellankikoti/kubilitics/commit/2718d96cdf9a7d839285be9ab92c7de26d0e329e))
- Add Endpoints informer and ServiceEndpoints/PodOwners to graph by @vellankikoti ([9dfc4a6](https://github.com/vellankikoti/kubilitics/commit/9dfc4a6aa18067cd739b3e6e26dbf29a301636a7))
- Wire classification engine into computeSingleResourceBlast by @vellankikoti ([b82e0dd](https://github.com/vellankikoti/kubilitics/commit/b82e0dd92e8f4f959ab05e7494a90d175ae4e6da))
- Add failure mode auto-detection and audit param support by @vellankikoti ([3a0cb10](https://github.com/vellankikoti/kubilitics/commit/3a0cb10a04b7e08c1084eadc2b28a6bd575154d0))
- Update GetSummary and computeNamespaceDeletion to use v2 scoring by @vellankikoti ([9a0c8ae](https://github.com/vellankikoti/kubilitics/commit/9a0c8ae880996bad35ba7b2b81305444f2716191))
- Add failure mode + audit params to API client and hook by @vellankikoti ([302e195](https://github.com/vellankikoti/kubilitics/commit/302e195d39ee02c2de4de0527afef3e66ffce24e))
- Add CoverageBanner — persistent amber bar for partial coverage by @vellankikoti ([e6d24d3](https://github.com/vellankikoti/kubilitics/commit/e6d24d3d37b25485d070301103bccb7e8c2a7483))
- Rewrite CriticalityBanner — headline with failure mode dropdown by @vellankikoti ([930021b](https://github.com/vellankikoti/kubilitics/commit/930021bec889064a4074f7fe9047083121311789))
- Add ScoreTooltip — hover tooltip for scoring factors by @vellankikoti ([1624a68](https://github.com/vellankikoti/kubilitics/commit/1624a683b652199dfbe0e65b6daa358c7d2847c2))
- Rewrite RiskIndicatorCards — hybrid score+badge+context cards by @vellankikoti ([c0cb3f0](https://github.com/vellankikoti/kubilitics/commit/c0cb3f05f17d4bb89b8ce2ae6e1acecadac15595))
- Add ScoreDetailSheet — full breakdown side panel with export by @vellankikoti ([d0fec6d](https://github.com/vellankikoti/kubilitics/commit/d0fec6d79a824fa6f57c842db93596b17c2dfb2d))
- Wire v2 components into BlastRadiusTab — coverage banner, failure mode selector, score detail sheet by @vellankikoti ([b0be2fe](https://github.com/vellankikoti/kubilitics/commit/b0be2fed019cb89c937ffa48a1d1be625b35f4db))
- Blast radius engine v2 — failure impact simulation by @vellankikoti ([c0d8d63](https://github.com/vellankikoti/kubilitics/commit/c0d8d63543e6509a38f41e8e20c340d49dde83b4))
- Add root cause heuristic utility by @vellankikoti ([f4d6dfa](https://github.com/vellankikoti/kubilitics/commit/f4d6dfa8c491a8ae72e511f0b2da1a960b9f175c))
- Add insight pod name parser by @vellankikoti ([436703c](https://github.com/vellankikoti/kubilitics/commit/436703c60c604538dbba82b8c41adcbc6d210edd))
- Add useInvestigateData hook by @vellankikoti ([ea8f939](https://github.com/vellankikoti/kubilitics/commit/ea8f93965ad420cb1d91957c15194abf8afd4735))
- Redesign InsightsBanner with inline triage panel by @vellankikoti ([daf5f96](https://github.com/vellankikoti/kubilitics/commit/daf5f96be2fb678a57838b99837e0e167ebcb3fa))
- Wire HealthDashboard to new inline InsightsBanner by @vellankikoti ([1010c66](https://github.com/vellankikoti/kubilitics/commit/1010c6660838052998ee978ab052d152c70aed2d))
- Add HealthIssueDetail page by @vellankikoti ([473da86](https://github.com/vellankikoti/kubilitics/commit/473da8645bba5ce3acf04cf0122255f3de1f5ca3))
- Add /health/issues/:insightId route by @vellankikoti ([5e8d6e1](https://github.com/vellankikoti/kubilitics/commit/5e8d6e1af1c76bd02a310c3508cc2888abb3f790))
- Add GraphLegend and FirstPaintOverlay by @vellankikoti ([5106b65](https://github.com/vellankikoti/kubilitics/commit/5106b651b95ec293c41919cbf57922c8930e5429))
- Add useWorkspaceData hook + YAMLDropZone component by @vellankikoti ([5ec3c81](https://github.com/vellankikoti/kubilitics/commit/5ec3c81c8b03f69f5fed6eca2bc8edbd82d14068))
- Add ImpactBar with expand-on-click micro-details by @vellankikoti ([27f53f3](https://github.com/vellankikoti/kubilitics/commit/27f53f3d14615e42aaf28b28dfd5d93a5e0a6d1b))
- Add IntelligenceWorkspace page — unified impact canvas by @vellankikoti ([cc057ea](https://github.com/vellankikoti/kubilitics/commit/cc057ea6892940820e81dce119f4b0cf8895b8b1))
- Add workspace CTA to Simulation page by @vellankikoti ([f06eeaa](https://github.com/vellankikoti/kubilitics/commit/f06eeaa64ef712063612041112f0373956c3e08d))
- Add route + replace 3 tabs with Analyze Impact button by @vellankikoti ([dba1046](https://github.com/vellankikoti/kubilitics/commit/dba1046ff204f52770c111956e232aa3ff51b287))
- Add CausalChain, CausalNode, CausalLinkV2 types by @vellankikoti ([c197d08](https://github.com/vellankikoti/kubilitics/commit/c197d08ad5646000185eb09864e98a0ad6552fb0))
- Add causal_chains table with upsert/get methods by @vellankikoti ([9d819b9](https://github.com/vellankikoti/kubilitics/commit/9d819b9bbed32456c25cabf9bec39449336c51d8))
- Implement ChainBuilder with recursive cause walking by @vellankikoti ([8ed3bb7](https://github.com/vellankikoti/kubilitics/commit/8ed3bb78e6a199888b013e4c4e2a09ce87ce7892))
- Implement ChainCache with persistence and restore by @vellankikoti ([2be85db](https://github.com/vellankikoti/kubilitics/commit/2be85db14f8080cdfc92d4904071a02a7bbb52d2))
- Add rollout_cascade, owner_propagation, service_disruption rules by @vellankikoti ([8afe3bc](https://github.com/vellankikoti/kubilitics/commit/8afe3bc42f29f99087e5731b65c32718706d4b82))
- Add /causal-chain endpoint and wire into insights ticker by @vellankikoti ([eca5d12](https://github.com/vellankikoti/kubilitics/commit/eca5d12bcfd5d6e2a50be8928ff11a8a4559e650))
- Add causalChainStore and useCausalChain hook by @vellankikoti ([2640276](https://github.com/vellankikoti/kubilitics/commit/26402765292072fa13c2a3e1b2ca65fd6072b629))
- Add CausalChainOverlay with node styling and step badge helpers by @vellankikoti ([95042c7](https://github.com/vellankikoti/kubilitics/commit/95042c7a8a83365cd59c1a0d2ab9938d36284ab1))
- Render causal chain step badge in BaseNode header by @vellankikoti ([b284ad9](https://github.com/vellankikoti/kubilitics/commit/b284ad92ea72adee6319c339292079f09086cac4))
- Integrate causal chain overlay with canvas node styling by @vellankikoti ([5b82441](https://github.com/vellankikoti/kubilitics/commit/5b82441a810c4a8963f53b9efe99f8af4cd97d2f))
- Add Root Cause and Chain pills with timeline drawer to ImpactBar by @vellankikoti ([a2f9f68](https://github.com/vellankikoti/kubilitics/commit/a2f9f6888b11a5d255204edf05cd0514ebe9fc5e))
- Add Root Cause toggle to workspace and Why? button to InsightsBanner by @vellankikoti ([b5020f0](https://github.com/vellankikoti/kubilitics/commit/b5020f00014e28a307bd507ef5daf7c1fe80bd4d))
- Wire ChainBuilder and ChainCache into backend bootstrap by @vellankikoti ([a8f65d8](https://github.com/vellankikoti/kubilitics/commit/a8f65d85835a6532f546e69d5f57e062f9610cad))
- Add amber dashed edges for causal chain overlay by @vellankikoti ([2306cf7](https://github.com/vellankikoti/kubilitics/commit/2306cf765cc1b0d3c24b418728fc424fd9ee5b50))
- Add click-to-select on timeline drawer steps by @vellankikoti ([92cdbe9](https://github.com/vellankikoti/kubilitics/commit/92cdbe9c34604f49cc5b063546573ccc6a270e6d))
- Add OTel demo app for tracing feature testing by @vellankikoti ([d19b95d](https://github.com/vellankikoti/kubilitics/commit/d19b95dede1ea921878e2233e25360a9c6244829))
- Add provider logo SVGs and mapping by @vellankikoti ([0867b92](https://github.com/vellankikoti/kubilitics/commit/0867b9229210b16b54a3f34c0d07128464570d7a))
- Add curated cloud service icons and mapping rules by @vellankikoti ([f08a2ee](https://github.com/vellankikoti/kubilitics/commit/f08a2ee5ad7bcbbf66d4868836c80e374466b475))
- Add useCloudContext hook for provider-aware rendering by @vellankikoti ([eba708c](https://github.com/vellankikoti/kubilitics/commit/eba708c57533378fed5534853af0c4e450c3bb49))
- Add cloudIconUrl override prop to K8sIcon by @vellankikoti ([c73da68](https://github.com/vellankikoti/kubilitics/commit/c73da68765dd839353268e7c57362a60e26c7038))
- Add provider badge and cloud icon to BaseNode by @vellankikoti ([9b4dbe6](https://github.com/vellankikoti/kubilitics/commit/9b4dbe6d27084831de05827ca0341e373ef18518))
- Add provider badge to CompactNode by @vellankikoti ([8e06718](https://github.com/vellankikoti/kubilitics/commit/8e0671855e1d85ccbe42c631d8f17b507b30b03e))
- Add provider badge and cloud icon to ExpandedNode by @vellankikoti ([78390a6](https://github.com/vellankikoti/kubilitics/commit/78390a60b5500b228a2122667cc8887ee3abc313))
- Add provider logo to cluster selector and dropdown by @vellankikoti ([be92090](https://github.com/vellankikoti/kubilitics/commit/be9209065ccc95d033bd718287fa909733346de7))
- Enhance Draw.io export with provider context and wire into toolbar by @vellankikoti ([ad79f92](https://github.com/vellankikoti/kubilitics/commit/ad79f92a2404f0deb1531d5932b5826174963cae))
- Add Go workspace connecting backend and kcli by @vellankikoti ([dd97653](https://github.com/vellankikoti/kubilitics/commit/dd9765395c8fd02ee6bfc53eccb4b3f42c0a7b4c))
- Enhance deploy table — colored UP-TO-DATE/AVAILABLE columns by @vellankikoti ([3ad9578](https://github.com/vellankikoti/kubilitics/commit/3ad9578eda4e4c51e2d5286e2bec22faa96432cf))
- Add ErrParseFailed error code for malformed API responses by @vellankikoti ([7a6c390](https://github.com/vellankikoti/kubilitics/commit/7a6c390574d92feec88808a4763d53a240686111))
- Add enterprise scoring engine — circuit breakers, penalty deductions, 5 categories by @vellankikoti ([5315351](https://github.com/vellankikoti/kubilitics/commit/5315351bafbdb0ed99921a24823827e3728dbc87))
- Wire enterprise scorer into overview cache and summary endpoint by @vellankikoti ([0ba6ee4](https://github.com/vellankikoti/kubilitics/commit/0ba6ee4d0c000f5f36f69907198ba554d5c60c3c))
- Frontend uses 5-category backend breakdown — adds Workloads to health widget by @vellankikoti ([2ac57c8](https://github.com/vellankikoti/kubilitics/commit/2ac57c8db8e1e79d0532e4f2347b9ec644fef44c))
- Add node-drain failure mode — simulates node eviction cascade by @vellankikoti ([be56d9c](https://github.com/vellankikoti/kubilitics/commit/be56d9c34628d36adc78e8b1753a44480ca2cac5))
- Traffic-weighted exposure scoring — high-RPS services score higher than low-RPS by @vellankikoti ([2c87e53](https://github.com/vellankikoti/kubilitics/commit/2c87e53d2dbb4eac28f0e253625e2d5c26f7aa62))
- Add confidence score (0-100) — shows data source completeness by @vellankikoti ([78fd98a](https://github.com/vellankikoti/kubilitics/commit/78fd98a93ff9330a2f4a98a91a0c0cbb13fe2236))
- Cascading amplification — broken services with upstream callers get amplified blast weight by @vellankikoti ([a7ac4a4](https://github.com/vellankikoti/kubilitics/commit/a7ac4a4ba9495242c8f704d744ebb3042ab6621a))
- Error rate per edge + data freshness in confidence by @vellankikoti ([34b23a1](https://github.com/vellankikoti/kubilitics/commit/34b23a1c4f6685b3f834df3039ed5458365825e5))
- Debounced search + clear button on all list pages, URL state for Pods filters by @vellankikoti ([16a8691](https://github.com/vellankikoti/kubilitics/commit/16a8691b551d7c5484f580c13dfe92fabc123c40))
- Consolidate Events + Events Intelligence tabs into one by @vellankikoti ([3e87d0a](https://github.com/vellankikoti/kubilitics/commit/3e87d0a4b66f4d68307c1fa7079328134a09790a))
- Replace proprietary agent with standard OpenTelemetry Collector by @vellankikoti ([0280cd4](https://github.com/vellankikoti/kubilitics/commit/0280cd48de07c4b839acda2796662a3a16903ef7))
- Redesign Traces tab — group by operation with severity-driven hierarchy by @vellankikoti ([fec4dc5](https://github.com/vellankikoti/kubilitics/commit/fec4dc5fd364b7781db50dc68e8ef6a9c197e0de))
- One-click per-deployment auto-instrumentation by @vellankikoti ([129a93d](https://github.com/vellankikoti/kubilitics/commit/129a93d56173e1a9556ef4122e7e3d6f5caf3123))
- Enterprise-grade per-deployment instrumentation by @vellankikoti ([b7234f2](https://github.com/vellankikoti/kubilitics/commit/b7234f275219212cc4c266ed44c0103caa171fa6))
- Scaffold kubilitics-otel chart by @vellankikoti ([494819f](https://github.com/vellankikoti/kubilitics/commit/494819f7db7a8ec259c83893a015def762c635e1))
- Document required and optional values by @vellankikoti ([734c1d6](https://github.com/vellankikoti/kubilitics/commit/734c1d6e8d97409d1d2a589de1465f50fed68d71))
- Add helper templates and value validation by @vellankikoti ([d9dd42e](https://github.com/vellankikoti/kubilitics/commit/d9dd42e7d86eaa7ba4bd4f9df2dbbca570250f78))
- Namespace, service account, RBAC, and service templates by @vellankikoti ([d2f836b](https://github.com/vellankikoti/kubilitics/commit/d2f836b04913f882a51906ef69e08adf0b432652))
- Collector configmap, deployment, and instrumentation CR templates by @vellankikoti ([b17132b](https://github.com/vellankikoti/kubilitics/commit/b17132b72994565fecc63c57334a29bd6b483ffa))
- Wrap helm template execution for chart rendering by @vellankikoti ([f250f08](https://github.com/vellankikoti/kubilitics/commit/f250f083f7f8835132b2fe925d0e06b578488af5))
- Static troubleshooting database with 14 failure modes by @vellankikoti ([4b599c1](https://github.com/vellankikoti/kubilitics/commit/4b599c168e697372fc3b16069f83f576a3298afc))
- Types for new tracing status response by @vellankikoti ([0f6391d](https://github.com/vellankikoti/kubilitics/commit/0f6391da50e4b36aae98f551aeb52e2be585554a))
- Rewrite GetTracingStatus as read-only computeTracingStatus + handler by @vellankikoti ([80e6551](https://github.com/vellankikoti/kubilitics/commit/80e655160ad6dd4861286d50eb87b7254b5d4a5d))
- Add diagnostics, install yaml, and instrument-command endpoints by @vellankikoti ([cf32e57](https://github.com/vellankikoti/kubilitics/commit/cf32e574220edb43a7d898ae6993a6e81d242399))
- Observability client for read-only tracing endpoints by @vellankikoti ([e24e566](https://github.com/vellankikoti/kubilitics/commit/e24e566c13d337e3acb0227b6bc6933504ef0b72))
- InstallCommandBlock with tabs, copy feedback, and dim state by @vellankikoti ([80ccfbd](https://github.com/vellankikoti/kubilitics/commit/80ccfbd8dede1450ac48669285d4efb8d7780eff))
- ComponentFlowRail horizontal pipeline visualization by @vellankikoti ([03a0330](https://github.com/vellankikoti/kubilitics/commit/03a03304d3e5abe4081eeafd12debcd4059f08fa))
- Collapsible DiagnosticsPanel with auto-expand on failure by @vellankikoti ([d297dd6](https://github.com/vellankikoti/kubilitics/commit/d297dd64f939950b0a3a029360118e7a0b54a96e))
- SetupObservability with live status, hero card, and stagger entry by @vellankikoti ([288776c](https://github.com/vellankikoti/kubilitics/commit/288776c55c6809a5b8ce346b26fbeac4865482ce))
- TracingStatusPill in header + route for SetupObservability by @vellankikoti ([9a37893](https://github.com/vellankikoti/kubilitics/commit/9a378932f6dbccd1adc9cb035254275149e196d3))
- Enterprise hardening — input validation, robust chart path, uniform errors by @vellankikoti ([dd8275a](https://github.com/vellankikoti/kubilitics/commit/dd8275a27f827771ad7ae38c6dce110bae4bbfff))
- Embed helm chart into backend binary for self-contained sidecar by @vellankikoti ([8d001b7](https://github.com/vellankikoti/kubilitics/commit/8d001b720876251dc2cac56e4ba9d480cccf18ed))
- Real per-cluster span tracking in OTLP receiver by @vellankikoti ([47fe1cc](https://github.com/vellankikoti/kubilitics/commit/47fe1cc9ece829dabae7df2105f7be174ff03d1e))
- Wire Cmd+R reload and Cmd+Shift+R force reload by @vellankikoti ([2f0edce](https://github.com/vellankikoti/kubilitics/commit/2f0edce6be0c00ff4168578b62861a920c11dfcb))
- Add cluster source column with kubeconfig/upload tag by @vellankikoti ([b77ff07](https://github.com/vellankikoti/kubilitics/commit/b77ff07772efde7ca71e4b987cf017ab068cc07a))
- Set Source on AddCluster and AddClusterFromBytes by @vellankikoti ([5e1c3ab](https://github.com/vellankikoti/kubilitics/commit/5e1c3ab2386f5c2a9404a1331e47f3e654a05db5))
- Add DeploymentMode + KubeconfigSync* fields by @vellankikoti ([212bd4f](https://github.com/vellankikoti/kubilitics/commit/212bd4fa5924ce5724ecd0151488882b1625c92a))
- Resolve watched kubeconfig paths by @vellankikoti ([2013fb2](https://github.com/vellankikoti/kubilitics/commit/2013fb216486759422687ae14b60094bfc6c6aee))
- Pre-destructive JSON snapshots by @vellankikoti ([4c9d4da](https://github.com/vellankikoti/kubilitics/commit/4c9d4da9032a9f005b7369326384276e226422fa))
- Syncer with safety cap + audit + snapshots by @vellankikoti ([38da405](https://github.com/vellankikoti/kubilitics/commit/38da4055b683f43d5cc70eb07856cf181230c7d5))
- Fsnotify event loop with singleflight coalescing by @vellankikoti ([62ea84a](https://github.com/vellankikoti/kubilitics/commit/62ea84ae007c8c53f86bfef9232f0e37147f3e99))
- Add restore-snapshot subcommand by @vellankikoti ([d908688](https://github.com/vellankikoti/kubilitics/commit/d9086883a5f26d0b268a23774538237d6cad19bf))
- Wire kubeconfig watcher into main startup by @vellankikoti ([b97fcef](https://github.com/vellankikoti/kubilitics/commit/b97fcef637ecb5f9ffe8462382038f8443de0941))
- Add types + severity helpers by @vellankikoti ([5e27924](https://github.com/vellankikoti/kubilitics/commit/5e27924b58302efc996b1455335fd2f054d23855))
- Static REASONS table with 26 K8s reason codes by @vellankikoti ([fc0cbea](https://github.com/vellankikoti/kubilitics/commit/fc0cbea977e723d95a9cda47b17c6424b72e3260))
- DiagnosePod — core per-pod diagnostic function by @vellankikoti ([359af7d](https://github.com/vellankikoti/kubilitics/commit/359af7dfdf09f0970e6fe0428a705f2d48ee80c8))
- Controller, Job, CronJob diagnoses + dispatcher by @vellankikoti ([c9e1aaf](https://github.com/vellankikoti/kubilitics/commit/c9e1aafd6ebe839c616b37238e0aad3b6163d1e9))
- ToDescribeText — clipboard-ready plain text by @vellankikoti ([123bacf](https://github.com/vellankikoti/kubilitics/commit/123bacfcd8179d398361bf59f2de403690705ec6))
- HealthyState + Header leaf components by @vellankikoti ([c4188c7](https://github.com/vellankikoti/kubilitics/commit/c4188c7646f0d9064517cd84bf6b1419b628636c))
- SuggestionButton + ReasonCard by @vellankikoti ([e425e78](https://github.com/vellankikoti/kubilitics/commit/e425e780968a44eea116db52eea725adeef30db2))
- ContainerStates + RecentEvents components by @vellankikoti ([e888dd8](https://github.com/vellankikoti/kubilitics/commit/e888dd8cafc423681199af6d6647e255470c356c))
- CopyAsDescribeButton by @vellankikoti ([f23b033](https://github.com/vellankikoti/kubilitics/commit/f23b0330318a3abfb4a6b5d36648c3ca68d11dda))
- DiagnosePanel — top-level composition by @vellankikoti ([ae3725e](https://github.com/vellankikoti/kubilitics/commit/ae3725e3fbf576962612a0b8c6308c5d17f110fa))
- Embed DiagnosePanel on all 7 workload detail pages by @vellankikoti ([f4a9338](https://github.com/vellankikoti/kubilitics/commit/f4a93388f494004e041345f3df0da17e99465378))
- Add filterYaml pure function with clean/raw presets by @vellankikoti ([25ca53c](https://github.com/vellankikoti/kubilitics/commit/25ca53ce7d38247564f12c0f1368b2e0888c9438))
- Clean/Raw preset toggle in YamlViewer by @vellankikoti ([f1b96e2](https://github.com/vellankikoti/kubilitics/commit/f1b96e28ba0ddf2510e64216f8ee390c2dbc7ce7))
- Pass parsed resource into YamlViewer for Clean/Raw filter by @vellankikoti ([fbb6cc1](https://github.com/vellankikoti/kubilitics/commit/fbb6cc1494e094671e471112c777a19e8879aa24))
- Add apply-ready preset, toJson, isLargeResource, wellKnownFoldPaths by @vellankikoti ([073408f](https://github.com/vellankikoti/kubilitics/commit/073408fe67dec6b94acee0d6552b213e28e8ac0f))
- Add yamlFoldRanges indent walker for well-known fold paths by @vellankikoti ([89cdc5e](https://github.com/vellankikoti/kubilitics/commit/89cdc5e09e3b4986e58bb73ebbaa0b838179ce70))
- Add YamlCopyMenu with 5 copy targets by @vellankikoti ([366ba21](https://github.com/vellankikoti/kubilitics/commit/366ba21d573471e6d35b65f8874c0ed95397fd35))
- Add language prop to CodeEditor (yaml | json) by @vellankikoti ([b1b2d07](https://github.com/vellankikoti/kubilitics/commit/b1b2d078b43563ed1bd293833a67a92b95823b7d))
- Apply-ready preset + YAML/JSON mode toggle in YamlViewer by @vellankikoti ([c2566af](https://github.com/vellankikoti/kubilitics/commit/c2566afe70569c109dbc4084302d7d967c86d3b0))
- Wire YamlCopyMenu into YamlViewer with 5 copy targets by @vellankikoti ([0303c92](https://github.com/vellankikoti/kubilitics/commit/0303c9200eee736782711b3e827a133d00962319))
- Fold menu with Fold All / Unfold All / well-known paths by @vellankikoti ([ab9d4ce](https://github.com/vellankikoti/kubilitics/commit/ab9d4ce5e163ffab1c4a78686b0bc2464ea2547a))
- Field-path breadcrumb follows cursor in read mode by @vellankikoti ([c80d12e](https://github.com/vellankikoti/kubilitics/commit/c80d12ef13d49d4aa24da53983220ace97f8f18c))
- Large-resource banner with auto-fold and session dismissal by @vellankikoti ([e71b6a8](https://github.com/vellankikoti/kubilitics/commit/e71b6a8661dc3952f24b2de2638575c1e8bae32a))
- Wire Blast Radius as a standard tab by @vellankikoti ([d8a5ed2](https://github.com/vellankikoti/kubilitics/commit/d8a5ed212369cf99f48b8c795fed9a8660458379))


### Changed

- Consolidate kubeconfig utils and standardize add-cluster UX by @vellankikoti ([66c84ed](https://github.com/vellankikoti/kubilitics/commit/66c84eda494c0ae0d7b0ca22bed7af07c4d5d99b))
- Remove unreliable inference functions — env-var, volume, netpol, istio by @vellankikoti ([eeaa8b0](https://github.com/vellankikoti/kubilitics/commit/eeaa8b07eec1796806882d6d523ea5fdf2530367))
- Remove old scoring model, keep PageRank and failure mode constants by @vellankikoti ([19e20c5](https://github.com/vellankikoti/kubilitics/commit/19e20c53953c31f9b7970326e6dd3e404bbbae3d))
- Migrate all types to camelCase — align with v2 backend API by @vellankikoti ([771cfd1](https://github.com/vellankikoti/kubilitics/commit/771cfd187f5e9d84b5aaf1e99b70842e39fcd43c))
- Delete mutation endpoints from tracing.go (~600 lines) by @vellankikoti ([078619e](https://github.com/vellankikoti/kubilitics/commit/078619e61dd9fba4e5c39b2e75b14a43682ed2a9))
- Register read-only tracing routes only by @vellankikoti ([ecbbead](https://github.com/vellankikoti/kubilitics/commit/ecbbeadc7730c6065b92da47ec43f9932d6a7a21))
- InstrumentationPanel is read-only command + preflight by @vellankikoti ([a9c7764](https://github.com/vellankikoti/kubilitics/commit/a9c7764cbb86613bba26af1224c440068a390849))
- Empty state links to setup page instead of inline install by @vellankikoti ([f6ab01b](https://github.com/vellankikoti/kubilitics/commit/f6ab01b2a24a5847c83231e9aee39c01c72b2e18))
- Delete legacy tracing mutation client functions by @vellankikoti ([0ac7720](https://github.com/vellankikoti/kubilitics/commit/0ac7720edae5afa80b3fd03e24fff51aff40f26c))
- Delete TracingSetup, TracingStatus, DeploymentPicker (superseded by setup page) by @vellankikoti ([1b25124](https://github.com/vellankikoti/kubilitics/commit/1b251241721327fdd4516f2ce1cc2d511697ea9a))
- Migrate TraceList + TracesPage off legacy install UI by @vellankikoti ([bcda03f](https://github.com/vellankikoti/kubilitics/commit/bcda03f05cdc0f804a7bb6b7d538047da5dcce36))
- Delete tracing.ts — zero remaining importers by @vellankikoti ([665a798](https://github.com/vellankikoti/kubilitics/commit/665a79805f5e3d2d58a9a9ea09f8d83f15cfb5a3))
- Split ResourceTableRow constants to restore Fast Refresh by @vellankikoti ([2d1677a](https://github.com/vellankikoti/kubilitics/commit/2d1677ad473c903d717056fb1f3150a00e957918))
- Shared WorkloadTerminalTab for all workload detail pages by @vellankikoti ([56dc50b](https://github.com/vellankikoti/kubilitics/commit/56dc50b2a19a135673efe056a3ee99bec5cfd461))
- Shared WorkloadTerminalTab for all workload detail pages by @vellankikoti ([15f3ccc](https://github.com/vellankikoti/kubilitics/commit/15f3ccc6211a4353b064ca262713a63b5473aa79))


### Documentation

- Update all documentation for v0.1.0 launch — remove placeholders and outdated claims by @vellankikoti ([9efa2d0](https://github.com/vellankikoti/kubilitics/commit/9efa2d0320138db50be1bc7068bc76c1a581f786))
- Clean up for v0.1.0 launch — remove internal dev docs, fix references by @vellankikoti ([920696b](https://github.com/vellankikoti/kubilitics/commit/920696b953cc1681e2b0b1e9c5fd9d48c22587db))
- Fix all v0.1.0 launch audit issues — ports, versions, broken links, non-existent features by @vellankikoti ([410da56](https://github.com/vellankikoti/kubilitics/commit/410da56b470cc73d54b72fd464e61a237506310c))
- Add seamless auto-update implementation plan by @vellankikoti ([bab5165](https://github.com/vellankikoti/kubilitics/commit/bab5165cf7e609ccd7f0e51da274ace596e14c96))
- Add operational intelligence platform design spec by @vellankikoti ([33d746b](https://github.com/vellankikoti/kubilitics/commit/33d746b5df8d2d9b70b1138436118f85cd339974))
- Add scoring methodology whitepaper — transparent resilience scoring by @vellankikoti ([334b114](https://github.com/vellankikoti/kubilitics/commit/334b114394d741800f417a2ba2ff0e9a0abd26ed))
- Clarify namespace scoping in health/count/instability help text by @vellankikoti ([c7da070](https://github.com/vellankikoti/kubilitics/commit/c7da07063f90221d70434f9a5eca84aca6c947bc))
- Write kubilitics-otel README with prerequisites and values reference by @vellankikoti ([f4d4918](https://github.com/vellankikoti/kubilitics/commit/f4d49183a164aa752d4c63c6e6780c7db372ca99))
- Write down the cluster mutation policy by @vellankikoti ([f9d77a1](https://github.com/vellankikoti/kubilitics/commit/f9d77a14826c428b5c79e2d852e74506d208ad63))
- Document kubeconfig sync configuration and safety mechanisms by @vellankikoti ([e3061f9](https://github.com/vellankikoti/kubilitics/commit/e3061f942e2eadcd0dc2dfba477c9fa07437dfe4))
- Move kubeconfig sync docs to repo-level docs/ by @vellankikoti ([99a01b2](https://github.com/vellankikoti/kubilitics/commit/99a01b282117b038246bf58c24349b54b02225cb))


### Fixed

- Prevent null array crashes with 3-layer defense by @vellankikoti ([c69997d](https://github.com/vellankikoti/kubilitics/commit/c69997d31849d6a6138cdbeb8f8e2299d88df747))
- Log filter UX — visible regex highlight, clear context label by @vellankikoti ([29802bd](https://github.com/vellankikoti/kubilitics/commit/29802bd259277eef4a86d41b77a747579018e1cd))
- CSV export with numbered sections — Resources (1-N) and Relationships (1-N) by @vellankikoti ([42c98d2](https://github.com/vellankikoti/kubilitics/commit/42c98d21c80018bfb5a0ae9c61a064dd2fb8e597))
- Update progress shows correct cumulative percentage by @vellankikoti ([96afefe](https://github.com/vellankikoti/kubilitics/commit/96afefe958351a69103878890b5872191b502d75))
- Bump Helm chart version to 0.1.1 for release consistency by @vellankikoti ([9401fc4](https://github.com/vellankikoti/kubilitics/commit/9401fc49b655b17f27fc684a6666d1b4d914fb66))
- Use tauri_plugin_updater::Builder::new().build() (not init()) by @vellankikoti ([6426fec](https://github.com/vellankikoti/kubilitics/commit/6426fec698e7909a40f879aa007bb5a30ad96e90))
- Skip Apple codesign (hangs in CI) — use Tauri update signing only by @vellankikoti ([144b803](https://github.com/vellankikoti/kubilitics/commit/144b803f7165f50a11afce13f3ee96346551ac92))
- Fix macOS codesign hang — prevent keychain auto-lock during build by @vellankikoti ([1a62fbb](https://github.com/vellankikoti/kubilitics/commit/1a62fbbeebda8a8478c322a1c26d7f43e0c8f1a1))
- Clone-to-namespace strips all auto-generated Pod fields by @vellankikoti ([ffebdd0](https://github.com/vellankikoti/kubilitics/commit/ffebdd0cf48e6111da00eaae15d8dff2f11a2256))
- Settings 'Check for Updates' uses tauri-plugin-updater (not removed command) by @vellankikoti ([61447ac](https://github.com/vellankikoti/kubilitics/commit/61447ac34c9caee84253356f6b462265a26d9256))
- Add 'updater' to Tauri bundle types for signed auto-updates by @vellankikoti ([a3d7b2c](https://github.com/vellankikoti/kubilitics/commit/a3d7b2c8f8b9b217acc3983d9ae4ca2bda660a7c))
- Add createUpdaterArtifacts to generate signed .tar.gz + .sig for auto-updates by @vellankikoti ([b453611](https://github.com/vellankikoti/kubilitics/commit/b4536114b287bc24b1cbbdd5be71c76a4a98347d))
- Set empty TAURI_SIGNING_PRIVATE_KEY_PASSWORD for keyless signing by @vellankikoti ([64a0bff](https://github.com/vellankikoti/kubilitics/commit/64a0bff2bb2335a466a217beb7f02c934564ea28))
- Move TAURI_SIGNING env vars to YAML env block for proper empty-string handling by @vellankikoti ([56541a9](https://github.com/vellankikoti/kubilitics/commit/56541a99fd5bcf71feb641d96052d8c2f6640a8c))
- Regenerate signing key with known password, update pubkey by @vellankikoti ([70655b3](https://github.com/vellankikoti/kubilitics/commit/70655b37389c7e220b6cc63274c257cdd6f918e9))
- Hide Connection Endpoints in desktop app — sidecar manages backend by @vellankikoti ([794cdb1](https://github.com/vellankikoti/kubilitics/commit/794cdb183a682296b7255334f15b4c84c7fb9bbc))
- Unify resource creation — all pages use ResourceCreator, remove dead wizards by @vellankikoti ([788fdf0](https://github.com/vellankikoti/kubilitics/commit/788fdf01dc60d1779efb35d1252f8d6381832c78))
- Remove wizard imports from Header.tsx (wizards directory deleted) by @vellankikoti ([91d94d8](https://github.com/vellankikoti/kubilitics/commit/91d94d80bc065b0f8ae2af6a997ebad2c1cff0e0))
- Use nginx:alpine in templates (universally available, avoids ErrImagePull) by @vellankikoti ([cc8b7f2](https://github.com/vellankikoti/kubilitics/commit/cc8b7f22a3481dde90a9c3226698796587f4fd8d))
- Use nginx:1.27 (stable) in templates instead of :alpine/:latest by @vellankikoti ([014533e](https://github.com/vellankikoti/kubilitics/commit/014533eb2b57fbcff21dacb2cc8d683823d41b1a))
- Proper pluralization for resource kinds + use ResolveGVR in apply by @vellankikoti ([1640a57](https://github.com/vellankikoti/kubilitics/commit/1640a57eed83165fd9781b512279f20ce483ea7d))
- Add ALL cluster-scoped resources to IsNamespaced + add runtimeclasses to GVR map by @vellankikoti ([56d0f30](https://github.com/vellankikoti/kubilitics/commit/56d0f30c28212c3fef1a41208b24baee7f378b43))
- Add Cancel button to cluster selection and context picker screens by @vellankikoti ([d90ee16](https://github.com/vellankikoti/kubilitics/commit/d90ee168690bf901f5a794418ea5ea078566209c))
- Refine context picker — subtle skip link instead of competing Cancel button by @vellankikoti ([1b71312](https://github.com/vellankikoti/kubilitics/commit/1b713126a75b1b7136579093b9971df4b0bee0d2))
- Cancel and Connect side by side — simple, visible, no tricks by @vellankikoti ([a1b702b](https://github.com/vellankikoti/kubilitics/commit/a1b702b4cc9b9ddb15a36f817dea58c371183cef))
- Cancel and Connect buttons same width (200px) by @vellankikoti ([0815a59](https://github.com/vellankikoti/kubilitics/commit/0815a599a0bdf8f13a796c7581cd0094bb1002ea))
- Cancel red outline, Connect blue, both compact h-9 by @vellankikoti ([c3bd76d](https://github.com/vellankikoti/kubilitics/commit/c3bd76dc594d7eea9fed1bbc122749994d6c3649))
- Cancel and Connect same size (w-160, h-11), Cancel red, Connect blue by @vellankikoti ([2d25698](https://github.com/vellankikoti/kubilitics/commit/2d25698418c8602a09e4ad32a39f9d8f31843d3b))
- Fleet page stops flashing skeleton on every 30s auto-refresh by @vellankikoti ([966f6cd](https://github.com/vellankikoti/kubilitics/commit/966f6cdab6d8cc84ae6586a7865b6aec9c301979))
- Fleet page never shows skeleton once data has loaded by @vellankikoti ([fae391e](https://github.com/vellankikoti/kubilitics/commit/fae391e2c27589bd0c2c8e7e11f77eb3a10eeb6e))
- Fleet group creation unresponsive — wrong store method name by @vellankikoti ([3d2fe33](https://github.com/vellankikoti/kubilitics/commit/3d2fe338c3957a5c26bcb2131aece40d179693ee))
- Add Cluster always shows paste/upload options, not just in empty state by @vellankikoti ([4853b2f](https://github.com/vellankikoti/kubilitics/commit/4853b2fcbd2d3c2a9fd30fda252dbb11829e919a))
- Add paste/upload kubeconfig to Desktop ContextPicker screen by @vellankikoti ([c1339b7](https://github.com/vellankikoti/kubilitics/commit/c1339b7534491df0226be1ce27f7e556ff504a6a))
- ContextPicker split layout — equal 50/50 halves by @vellankikoti ([af32065](https://github.com/vellankikoti/kubilitics/commit/af32065c563694c021da0f20b256214e44c7c5f4))
- ContextPicker cluster list scales with viewport for 15-20 clusters by @vellankikoti ([a77e312](https://github.com/vellankikoti/kubilitics/commit/a77e312560eea2b1362063f0437a142ca18f3e45))
- Two template bugs — namespace injection data corruption + stale editor by @vellankikoti ([a6520ae](https://github.com/vellankikoti/kubilitics/commit/a6520ae7dda66812b1c13d1981215f806e7e01d6))
- Make resource detail tab bar sticky across all 52+ detail pages by @vellankikoti ([04cc0ec](https://github.com/vellankikoti/kubilitics/commit/04cc0ecda745a4c910005e4d2815dfcb5cf44d60))
- Make table headers sticky across all list pages by @vellankikoti ([0485c20](https://github.com/vellankikoti/kubilitics/commit/0485c20061f981f391a98e804749f458979e3344))
- Dialog overflow on small screens + port forward validation by @vellankikoti ([84fac25](https://github.com/vellankikoti/kubilitics/commit/84fac2532dcdbb349d0ea69a4508ca7e5d75cb6a))
- Error handling, loading states, and dark mode across 3 files by @vellankikoti ([6b7bfa9](https://github.com/vellankikoti/kubilitics/commit/6b7bfa924225230b4f61961ab8270e762c8c345d))
- Standardize destructive button colors across 23 list pages by @vellankikoti ([259f2c6](https://github.com/vellankikoti/kubilitics/commit/259f2c640d566f937988b3fbfd522fce49fa82a4))
- Terminal/File Explorer toggle buttons invisible in light mode by @vellankikoti ([fe2667b](https://github.com/vellankikoti/kubilitics/commit/fe2667b44d88d2044461127a0ad28636f769275e))
- Toggle buttons and segmented controls visible in both themes by @vellankikoti ([aa7ce51](https://github.com/vellankikoti/kubilitics/commit/aa7ce51f46d6a01281f3aa410823b0b012991ff3))
- Topology components dark mode — 11 hardcoded colors fixed by @vellankikoti ([944abc7](https://github.com/vellankikoti/kubilitics/commit/944abc7669dd70ea82f33982e4f3bae110bb6a53))
- NetworkMetrics headers + MultiPodLogViewer dark/light mode by @vellankikoti ([2e1138f](https://github.com/vellankikoti/kubilitics/commit/2e1138f5cc5eaff4ea1400ebee480b017229a2e2))
- Dark mode for dialogs, search, infrastructure map, CRDs by @vellankikoti ([8b8a2ab](https://github.com/vellankikoti/kubilitics/commit/8b8a2ab6107ee832d35956f963b68cea62755fb5))
- Systematic dark mode sweep — 20 files, ~90 hardcoded colors fixed by @vellankikoti ([34e51b6](https://github.com/vellankikoti/kubilitics/commit/34e51b6639de3317ec13504f979e139a29c5d83f))
- KubeconfigContextDialog text color dark mode by @vellankikoti ([2044ce3](https://github.com/vellankikoti/kubilitics/commit/2044ce39d6813b8e2da97955da5edcf773470e2f))
- InfrastructureMap metric labels dark mode by @vellankikoti ([1d656d1](https://github.com/vellankikoti/kubilitics/commit/1d656d1dfc61d7e16da88b2b36704408f1220e29))
- Last 3 dark mode stragglers — loading bar, pod status, summary node by @vellankikoti ([98b8693](https://github.com/vellankikoti/kubilitics/commit/98b869317b1db51ba3142fe84eca63adf13ccf62))
- Chart tooltips and SVG strokes for dark mode by @vellankikoti ([1fbfa61](https://github.com/vellankikoti/kubilitics/commit/1fbfa61ed61a3642d4af851b0a92d8ce2b39f819))
- Topology page — 22 inline style colors switched to CSS variables by @vellankikoti ([ad1bf4a](https://github.com/vellankikoti/kubilitics/commit/ad1bf4ae1baf85ff8d093e47f20b1de3701558c3))
- Topology toolbar header — solid bg-card with shadow for separation by @vellankikoti ([2b6d4c3](https://github.com/vellankikoti/kubilitics/commit/2b6d4c38dc72f90d148989492790f9f1e787f695))
- Topology toolbar overflow — wraps properly with sidebar open by @vellankikoti ([a55470c](https://github.com/vellankikoti/kubilitics/commit/a55470cf317195180ab9defb4284956270488417))
- Topology page shows page scrollbar instead of trapping in viewport by @vellankikoti ([313e652](https://github.com/vellankikoti/kubilitics/commit/313e65217349abc9453f46aa6963969221c09099))
- Simplify Topology page layout — fixed canvas height, page scrolls by @vellankikoti ([e457d27](https://github.com/vellankikoti/kubilitics/commit/e457d2752ffd0daa9c4d96a9c37a891171a7c90c))
- Resource topology shows loading overlay when switching depth level by @vellankikoti ([c5ff908](https://github.com/vellankikoti/kubilitics/commit/c5ff9087d13d8bc697c3287e8bf59213ea9aa672))
- Add wizard stubs so production build passes by @vellankikoti ([37e6575](https://github.com/vellankikoti/kubilitics/commit/37e6575456b8cbd42297e4e341c9555856d11a83))
- EKS/GKE/AKS clusters fail to connect — aws/gcloud not in PATH by @vellankikoti ([071badb](https://github.com/vellankikoti/kubilitics/commit/071badb3567c512b5cfd1746413e558bc9c08c4e))
- Robust PATH enrichment for EKS/GKE/AKS — defense-in-depth by @vellankikoti ([76862d8](https://github.com/vellankikoti/kubilitics/commit/76862d840533ebb49b8a3a07dac09af2772a88bb))
- Update flow — auto-restart after download, better card design by @vellankikoti ([9f3730d](https://github.com/vellankikoti/kubilitics/commit/9f3730d9608e906906753fdf1f537e63a8f098f6))
- Desktop startup never falls to wrong page after update by @vellankikoti ([ef1507a](https://github.com/vellankikoti/kubilitics/commit/ef1507aa00a847e8e94e4366fea2d889f71e6831))
- Fast startup — 3s to dashboard instead of 15s by @vellankikoti ([da1a6e2](https://github.com/vellankikoti/kubilitics/commit/da1a6e240b7976e070b04151d3b251cc18ddfdfd))
- Startup polls every 500ms instead of 1.5s — instant once backend is up by @vellankikoti ([eac2a2a](https://github.com/vellankikoti/kubilitics/commit/eac2a2a82bdc72e0b25dbb0e65c90f0b73af9871))
- Clean startup — wait for backend ready signal, then one API call by @vellankikoti ([992b892](https://github.com/vellankikoti/kubilitics/commit/992b89273ee89dcfa76562423f1d4db9aefad5df))
- Resolve 5 topology engine bugs for credible dependency graphs by @vellankikoti ([55a8769](https://github.com/vellankikoti/kubilitics/commit/55a87695903b855d15ef4571850fa2ba9f169a5a))
- Replace SOC2 stub controls with real RBAC scanning (Gap 7) by @vellankikoti ([df8f9af](https://github.com/vellankikoti/kubilitics/commit/df8f9af668b3cc76dc545131a825d011a25cf499))
- Wire active cache invalidation into informer rebuild pipeline (Gap 2) by @vellankikoti ([1f62810](https://github.com/vellankikoti/kubilitics/commit/1f6281030abd5462df99c344b18aed2adb3f0adc))
- Wire K8s informer events to topology cache invalidation (Gap 2) by @vellankikoti ([a075bee](https://github.com/vellankikoti/kubilitics/commit/a075beec0f24fad91be6b0c8929065a3ac8bce7c))
- Register intelligence API routes in SetupRoutes — fix 404s by @vellankikoti ([ef37011](https://github.com/vellankikoti/kubilitics/commit/ef37011fdb293eb0e8070922eaa3c693b8781a10))
- Restore all pillar routes and sidebar items after merge by @vellankikoti ([453c504](https://github.com/vellankikoti/kubilitics/commit/453c50405a29185015e611db602da5b29bc545d6))
- Restore missing intelligence page routes in App.tsx by @vellankikoti ([8c50d07](https://github.com/vellankikoti/kubilitics/commit/8c50d07171cecef105d4286b3f487bc98ef080c4))
- Auto-Pilot dashboard uses full-width page layout by @vellankikoti ([40cc89f](https://github.com/vellankikoti/kubilitics/commit/40cc89f27f166e04c9b22e1949652b1f1996badb))
- Register report schedule CRUD routes in SetupRoutes by @vellankikoti ([7617ff4](https://github.com/vellankikoti/kubilitics/commit/7617ff4fe9d5a059099f24ba876485ee3e5d2111))
- Node failure simulation uses correct Node key format by @vellankikoti ([3045581](https://github.com/vellankikoti/kubilitics/commit/3045581e480106723c14265fa5bb36ad3d389ae4))
- Robust node matching in simulation — scan by Kind+Name, non-fatal if missing by @vellankikoti ([54f3a13](https://github.com/vellankikoti/kubilitics/commit/54f3a135ca23323a1728ff33098d2c44f0a48766))
- Simulation scoring — SPOF counts only workloads, filter float noise by @vellankikoti ([d49ef83](https://github.com/vellankikoti/kubilitics/commit/d49ef835703bb21ec713b03d1ae51d8f12a1b5b1))
- Align simulation scenario types with backend + resource picker by @vellankikoti ([6104e90](https://github.com/vellankikoti/kubilitics/commit/6104e90358e891ace1f704d0857643ff748446ca))
- Scenario card labels + affected services render crash by @vellankikoti ([8ec508a](https://github.com/vellankikoti/kubilitics/commit/8ec508a9fddf985ad3747bfa7922308e3d5583e7))
- Sprint 0 — WebSocket retry backoff + cluster-scoped node IDs by @vellankikoti ([263178b](https://github.com/vellankikoti/kubilitics/commit/263178bc96f2d5244f2f792a36e716cbb8c1b285))
- Sprint 0 — EnsureMaps nil-safety + report schedule routes by @vellankikoti ([7fcaa6b](https://github.com/vellankikoti/kubilitics/commit/7fcaa6b87857ceef2b4b53114b0bf0aa5b7a38b7))
- Wire Auto-Pilot into server startup + lazy graph engine init by @vellankikoti ([9c32a2d](https://github.com/vellankikoti/kubilitics/commit/9c32a2d9210a28de042284523ffc087fa92e7c86))
- Cluster switch clears ALL backend query cache — no stale data flash by @vellankikoti ([7c9ee2c](https://github.com/vellankikoti/kubilitics/commit/7c9ee2c62e94a6166e788487cb2e9b6d0ab7d2f6))
- Cluster connect dead-click after logout by @vellankikoti ([407ea96](https://github.com/vellankikoti/kubilitics/commit/407ea96c6be46c6d2cd6b1266accfda67ebac803))
- Eliminate 7 dead-end UX flows — toast feedback on all silent failures by @vellankikoti ([f9099ac](https://github.com/vellankikoti/kubilitics/commit/f9099aca90dc794f092a8b1b09982b3f44a637b5))
- Production hardening — 10 critical fixes from Staff+ audit by @vellankikoti ([df7319f](https://github.com/vellankikoti/kubilitics/commit/df7319fbf132dee49e653e599d4c1a3e57141796))
- 3 ship-blocking issues from code review by @vellankikoti ([4ea5796](https://github.com/vellankikoti/kubilitics/commit/4ea57965161101c5d950e5005651863bc6a795b1))
- Resolve ALL remaining code review issues — enterprise hardening by @vellankikoti ([5ede44f](https://github.com/vellankikoti/kubilitics/commit/5ede44f8b549ba64d2d88f9ce7d5d15f0234ea67))
- Terminal auto-reconnects when tab becomes visible by @vellankikoti ([a317460](https://github.com/vellankikoti/kubilitics/commit/a317460a4ba87b1f372cd04babe5c2560c44e8d1))
- Add floating Exit button + Escape key for maximized terminal by @vellankikoti ([e551b7c](https://github.com/vellankikoti/kubilitics/commit/e551b7c0c2e85641fc46b9be4537418658bc91cb))
- Terminal header with action buttons visible again by @vellankikoti ([a2bd95a](https://github.com/vellankikoti/kubilitics/commit/a2bd95ae10e2770957eb234f5f3f96c6f2a4b5b1))
- Terminal refit on tab switch — multiple attempts + ResizeObserver by @vellankikoti ([d2fc534](https://github.com/vellankikoti/kubilitics/commit/d2fc5340521a0e8d1235fe9ea670966b034b54c3))
- Metrics time delta uses filteredHistory.ts (unix seconds) instead of missing timestamp field by @vellankikoti ([c9b4529](https://github.com/vellankikoti/kubilitics/commit/c9b4529eeeaf1be1d772bb8d41ea9b27c47e14fd))
- Metrics time label more visible — 11px, font-medium, full opacity by @vellankikoti ([6e676ef](https://github.com/vellankikoti/kubilitics/commit/6e676ef2c3e10e1ab211978a6a616c8a42fbd3d5))
- Hide per-pod table for single pods — only show for 2+ replicas by @vellankikoti ([86901da](https://github.com/vellankikoti/kubilitics/commit/86901da794e48cc2226c658863690c8b099afc0e))
- Server pagination respects page-size selector — default 10 not 50 by @vellankikoti ([d7d3b76](https://github.com/vellankikoti/kubilitics/commit/d7d3b766ef8b7edea3924c8bfec4157c9088a2d7))
- Pods page crash — remove sort state sync useEffect causing React queue error by @vellankikoti ([f929ad9](https://github.com/vellankikoti/kubilitics/commit/f929ad9e4862730a7017ac6ceadf17b397bcebbd))
- Terminal state preserved across tab switches — keep-alive uses real dimensions by @vellankikoti ([76ee52c](https://github.com/vellankikoti/kubilitics/commit/76ee52ca903cb129373e784550a5d1bb9b19973c))
- Terminal state truly persisted — pin keep-alive content refs across renders by @vellankikoti ([ec882cd](https://github.com/vellankikoti/kubilitics/commit/ec882cd2db73bc6b9710c8593591276b0ec0833f))
- Header and stats show server total (99) not page count (10) by @vellankikoti ([fd19510](https://github.com/vellankikoti/kubilitics/commit/fd195107a49456bd42b40bb4c90ba98c36eeb21d))
- Terminal state actually preserved — same element type for keep-alive tabs by @vellankikoti ([52ff21b](https://github.com/vellankikoti/kubilitics/commit/52ff21b3e60488eae5e3694e1a61dbed0f2cf1ba))
- Brand watermark larger — 36px logo, bigger padding, more visible by @vellankikoti ([18d1ad8](https://github.com/vellankikoti/kubilitics/commit/18d1ad81fea5fe42091606f314995662d79a0519))
- Brand watermark sized to enterprise standard — 44px logo, prominent pill by @vellankikoti ([25fc549](https://github.com/vellankikoti/kubilitics/commit/25fc549b5255ff5f7bc5784028aded67641d62cc))
- Header brand matches enterprise standard — bolder wordmark, cleaner zone by @vellankikoti ([32ec2d7](https://github.com/vellankikoti/kubilitics/commit/32ec2d734a396444ddbaffb9206966920cdb632d))
- Header layout like Docker Desktop — clean logo zone, taller header by @vellankikoti ([d225b99](https://github.com/vellankikoti/kubilitics/commit/d225b99434439156fcde4994f25dede51732a016))
- Events intelligence cluster ID mismatch — use currentClusterId exclusively by @vellankikoti ([063c222](https://github.com/vellankikoti/kubilitics/commit/063c222b12052087ad9f8487bf91150efaf8d49d))
- Insight notifications persist seen IDs in localStorage — no re-fire on refresh by @vellankikoti ([40b4338](https://github.com/vellankikoti/kubilitics/commit/40b4338fff1bf9f67cccffa885278d09bc4679ee))
- Release audit — fix 2 blockers + all high-priority issues by @vellankikoti ([2f66c85](https://github.com/vellankikoti/kubilitics/commit/2f66c8543f772eec2acf440e3a9a5cec19ba249e))
- Remaining audit items — card consistency, semantic colors, type safety by @vellankikoti ([09c5652](https://github.com/vellankikoti/kubilitics/commit/09c56527a71ed8f6b09173badfc42776d34a8f6d))
- Add missing logs components — gitignore was blocking src/components/logs/ by @vellankikoti ([335fe76](https://github.com/vellankikoti/kubilitics/commit/335fe76847a0c6de2da4fa45fdad36a85f5cd4db))
- Move Tauri signing key password to GitHub secret by @vellankikoti ([f5cc63e](https://github.com/vellankikoti/kubilitics/commit/f5cc63e36f182732833523771d3de3b338361bd4))
- Insight flood — auto-prune to max 3 active per rule + investigate → incidents by @vellankikoti ([242694e](https://github.com/vellankikoti/kubilitics/commit/242694e76018fc459b2edaddeebde8397f46e5ac))
- Event row shows full pod name — removed max-w-[220px] truncation by @vellankikoti ([4d878ac](https://github.com/vellankikoti/kubilitics/commit/4d878acc4634edd357780f49996825da65412af0))
- Text truncation audit — 12 fixes across 11 files by @vellankikoti ([bc75711](https://github.com/vellankikoti/kubilitics/commit/bc75711b158bf1a9f48fc77e202686ba426be6af))
- Traces tab stuck on skeleton — React Query v5 disabled query stays isLoading by @vellankikoti ([6dc8597](https://github.com/vellankikoti/kubilitics/commit/6dc8597b0505b80c2a6326c94fd74c48ca8a0323))
- Traces page default to 24h — was 1h, traces missed if older by @vellankikoti ([b4ffd88](https://github.com/vellankikoti/kubilitics/commit/b4ffd884ced20668c996c23baccd4648cd2a0e74))
- OTLP endpoint was /api/v1/v1/traces (double v1) — now correct by @vellankikoti ([8d0bba4](https://github.com/vellankikoti/kubilitics/commit/8d0bba42c88e4deb558d1d9668adb5d4f620db1c))
- Traces setup guide — remove Add-ons link, self-contained helm commands by @vellankikoti ([60ca14b](https://github.com/vellankikoti/kubilitics/commit/60ca14b25681cc360dad79e5a1f22d0cf815c097))
- Traces setup — remove Docker-only host, works on any cluster (EKS/AKS/GKE/local) by @vellankikoti ([dcb05f9](https://github.com/vellankikoti/kubilitics/commit/dcb05f92443498979de921edf6ce9762dabb9525))
- Traces setup simplified — 2 steps instead of 4, no cluster-specific hacks by @vellankikoti ([b872b61](https://github.com/vellankikoti/kubilitics/commit/b872b613c658596bb9f17564aecc71d9866f6268))
- Traces setup — clean, no confusing env vars, auto-detect service names by @vellankikoti ([eaa8fa6](https://github.com/vellankikoti/kubilitics/commit/eaa8fa61fb5bc03b02de3ec3b90f92e281f8268b))
- Tracing status returns available_deployments for deployment picker by @vellankikoti ([91b5fc6](https://github.com/vellankikoti/kubilitics/commit/91b5fc6772f9fcf55808c5e803c9473786878944))
- Trace-agent imagePullPolicy IfNotPresent + Docker image built locally by @vellankikoti ([400f3a0](https://github.com/vellankikoti/kubilitics/commit/400f3a025b05d203e9768a8243c2a5d2e045cd9f))
- Enable tracing returns immediately — installs cert-manager + OTel Operator async by @vellankikoti ([e9f3d6a](https://github.com/vellankikoti/kubilitics/commit/e9f3d6a9b7ecea19b817df4acee5b6903bfd81bf))
- Tracing status updates after setup — invalidate query cache on complete by @vellankikoti ([e96c5be](https://github.com/vellankikoti/kubilitics/commit/e96c5be93356aaf6aa54c6331b9379ad3ff96393))
- Tracing status query was disabled — baseUrl is '' in dev mode by @vellankikoti ([bce1a65](https://github.com/vellankikoti/kubilitics/commit/bce1a6559edd7f00cbdbd9226a0d9597bdd8875d))
- Traces crash on agent_span_count + double-click header to maximize by @vellankikoti ([09d89d6](https://github.com/vellankikoti/kubilitics/commit/09d89d65023b038c038e62d8281a11f941ef49e5))
- Traces page shows Instrument Applications button + wizard skips to picker by @vellankikoti ([cdade62](https://github.com/vellankikoti/kubilitics/commit/cdade62170f621ab7cf3203ca84b4bac94730aa9))
- Instrument dialog shows deployment picker instead of skipping to done by @vellankikoti ([f4c0e4e](https://github.com/vellankikoti/kubilitics/commit/f4c0e4e7a1e1c34e58aca48d3f8669e8907a3419))
- Trace puller Services field type mismatch — 373 traces now flowing by @vellankikoti ([fd7dff2](https://github.com/vellankikoti/kubilitics/commit/fd7dff2e31bf257aaf8cb0bcdc0510266f1ecffb))
- Traces page — 10 per page, fetch limit 50 by @vellankikoti ([d5e52af](https://github.com/vellankikoti/kubilitics/commit/d5e52afc61535186fc126d3221c5628ff1f625c8))
- Double-click maximize only on empty header area — not buttons/inputs by @vellankikoti ([6ab2d50](https://github.com/vellankikoti/kubilitics/commit/6ab2d5049e42a48e8daa3df4f3a05ff63be74757))
- Resource traces tab uses OTel spans — was querying wrong table by @vellankikoti ([f4a589a](https://github.com/vellankikoti/kubilitics/commit/f4a589a6d294cb3bfad974fe0088efd3212c39e5))
- Double-click maximize — check data-tauri-drag-region, not exclude list by @vellankikoti ([9c124e3](https://github.com/vellankikoti/kubilitics/commit/9c124e35175b8ca32e23277f153adaee121e55ec))
- Resource traces tab infinite skeleton — query key had unstable object ref by @vellankikoti ([c6e11d3](https://github.com/vellankikoti/kubilitics/commit/c6e11d39838638a838c9032aa38d54de62b3d47b))
- Resource traces tab — 10 per page with pagination controls by @vellankikoti ([21e7709](https://github.com/vellankikoti/kubilitics/commit/21e7709195c48f18f6d319342a7eb7f2d470ee00))
- Correct test thresholds and remove incorrect addEdge symmetry test by @vellankikoti ([5f71e96](https://github.com/vellankikoti/kubilitics/commit/5f71e96c7bee86ab3bf46b5743e55b281a026b05))
- Auto-fallback to first cluster when persisted cluster ID is stale by @vellankikoti ([2599a2c](https://github.com/vellankikoti/kubilitics/commit/2599a2c88da2a8d179dbf06bdd0794e9fa4c1df3))
- Sidebar counters use activeCluster.id instead of stale currentClusterId by @vellankikoti ([5dad077](https://github.com/vellankikoti/kubilitics/commit/5dad0773cff2257e66bdf0929952b068d3ffbc4b))
- Headlamp-style cluster sync — always fetch from backend, auto-select, poll by @vellankikoti ([0ff5ede](https://github.com/vellankikoti/kubilitics/commit/0ff5edec259410f9a13988d92dc866f181fb0388))
- Terminal colors — inject LS_COLORS, color aliases, and colored prompt by @vellankikoti ([f05e751](https://github.com/vellankikoti/kubilitics/commit/f05e7516dfea5f32cdcf68b2cc126cfe30e1fe93))
- Terminal colors — use POSIX-compatible shell setup, no bash-only syntax by @vellankikoti ([94d0429](https://github.com/vellankikoti/kubilitics/commit/94d0429bc759d411c7d95a1049f8f3b5bab13499))
- Terminal colors — write rc file to /tmp so aliases survive exec by @vellankikoti ([d3831f0](https://github.com/vellankikoti/kubilitics/commit/d3831f0f70de148339d78cace302c59a19602e2c))
- Terminal colors — use printf for rc file to avoid heredoc escaping issues by @vellankikoti ([a148b78](https://github.com/vellankikoti/kubilitics/commit/a148b78ec1afb6441b0934841abab92c6fb0d896))
- Terminal colors + cluster restore race condition by @vellankikoti ([3cd5cc8](https://github.com/vellankikoti/kubilitics/commit/3cd5cc8d84504d74a018298e36f2f9f940ca3f40))
- Show terminating pods with correct status by @vellankikoti ([f1d2dfb](https://github.com/vellankikoti/kubilitics/commit/f1d2dfbfd7dda104575aea92b39effcd570a6a1e))
- Add nil pointer guards to ingress, scaling, rbac matchers by @vellankikoti ([51afb03](https://github.com/vellankikoti/kubilitics/commit/51afb03dd78f7a52dd06b06e9927d8ca091eb874))
- Use word-boundary matching in DNS matcher to prevent false matches by @vellankikoti ([6b4576f](https://github.com/vellankikoti/kubilitics/commit/6b4576f197f472a3a1f007c31abbdf7c8bf170f8))
- PDF export timeout, transformGraph error guard, highlight Set performance by @vellankikoti ([b2ca58a](https://github.com/vellankikoti/kubilitics/commit/b2ca58a5c2a464e6d02ceaf9576348352e0bf019))
- Add FailedMatchers tracking, selector logging, global edge dedup by @vellankikoti ([da1b0ac](https://github.com/vellankikoti/kubilitics/commit/da1b0ac521abb8235e8d2577067162ba9e6f9b1a))
- ELK memory leak cleanup + zoom threshold flicker fix by @vellankikoti ([3e99cca](https://github.com/vellankikoti/kubilitics/commit/3e99cca6f1cf6bd0423143aed70bb2f29d124313))
- Add keyboard navigation (Enter/Escape) to topology nodes by @vellankikoti ([fddd10a](https://github.com/vellankikoti/kubilitics/commit/fddd10a5db5a0c2181424ee4253be05845dca112))
- Auto-select pod in useEffect + show blast radius errors in workspace by @vellankikoti ([571d76c](https://github.com/vellankikoti/kubilitics/commit/571d76c3c7fe391ff16cc3a1b6c7482783664682))
- Handle actual insight Detail formats in parseInsightResource by @vellankikoti ([a8f53d8](https://github.com/vellankikoti/kubilitics/commit/a8f53d889c8f98711051e4b1585f224e95aed5a4))
- Add mutual exclusion between blast radius and root cause overlays by @vellankikoti ([b1da627](https://github.com/vellankikoti/kubilitics/commit/b1da627fb76c20071d98c966cab3f3f1350871d2))
- Add Investigate link to timeline footer and lazy-load WhyButton by @vellankikoti ([949fdf0](https://github.com/vellankikoti/kubilitics/commit/949fdf00134b000b1c73319c12c80e03222526ad))
- Ensure Analyze Impact on all resource detail pages + blast score rounding by @vellankikoti ([bba4191](https://github.com/vellankikoti/kubilitics/commit/bba4191c5fc6fa41a4725df580cb4decfdba146e))
- Show loading state instead of error when graph is building by @vellankikoti ([013fd4b](https://github.com/vellankikoti/kubilitics/commit/013fd4bcb1ee082d0d57bfca2e7073aff228c6ee))
- Round all scores to 2 decimal places (backend + frontend) by @vellankikoti ([f1393d8](https://github.com/vellankikoti/kubilitics/commit/f1393d8d6488737dee7278cb0a2ec91a4b0c9327))
- Permanent pod status counts from backend summary + header/search UX by @vellankikoti ([13dccc7](https://github.com/vellankikoti/kubilitics/commit/13dccc73d9e1b37448c3686aa26a088deac5784c))
- Branded header redesign + refetch on window focus by @vellankikoti ([731c49a](https://github.com/vellankikoti/kubilitics/commit/731c49ad5682677a31a8ee6e337ab7542570c84c))
- Use python:3.12-slim for OTel demo services by @vellankikoti ([250fa68](https://github.com/vellankikoti/kubilitics/commit/250fa68ac7230e29f2e247c2aa64a2e06b89cc83))
- OTel demo app working — python:3.12 + correct SDK imports by @vellankikoti ([dfabe2e](https://github.com/vellankikoti/kubilitics/commit/dfabe2e26b7350b7156efacab7e74405b7853b38))
- Official provider logo SVGs + namespace list from cluster by @vellankikoti ([1a1d9df](https://github.com/vellankikoti/kubilitics/commit/1a1d9dfcc870fea1d5052f1806e8faba8d63c809))
- Solid white bg for header provider logo + fleet page icons by @vellankikoti ([803d7c2](https://github.com/vellankikoti/kubilitics/commit/803d7c2e7045f022f1ebbd6aaf8e667014ddd6e6))
- Clean resource detail header layout by @vellankikoti ([78a9105](https://github.com/vellankikoti/kubilitics/commit/78a9105820a84fb23d1f40305cbb58c0cfa027bb))
- Kcli table colors — brighter text for dark terminals + cluster shell tooltip by @vellankikoti ([4e9dfef](https://github.com/vellankikoti/kubilitics/commit/4e9dfef838f90fc1162f3f2a76c9f2c77d15c540))
- Namespace column uses Info color (cyan) instead of Muted by @vellankikoti ([44c5ee3](https://github.com/vellankikoti/kubilitics/commit/44c5ee3cb6e3d65f5d1e28e4a74e2ca412ba5676))
- Reduce default kubectl timeout from 5m to 15s by @vellankikoti ([e07ab93](https://github.com/vellankikoti/kubilitics/commit/e07ab93a00b4d1edd514450e1a5fa52b91ad6216))
- Replace Sscanf with strconv in metrics parsing by @vellankikoti ([7a6e6a0](https://github.com/vellankikoti/kubilitics/commit/7a6e6a07a843832430952619ed318f31f568fb75))
- Guard table footer rendering against empty/short footer rows by @vellankikoti ([56abe88](https://github.com/vellankikoti/kubilitics/commit/56abe88060256e4d7b1224a4f47c843d2813bb06))
- Health reports UNKNOWN instead of 100 when API returns no data by @vellankikoti ([9a3ab46](https://github.com/vellankikoti/kubilitics/commit/9a3ab469ebbe856075f4d10312f266d49484ed05))
- Add 60s timeout to multi-pod log stream cleanup by @vellankikoti ([d57b6b1](https://github.com/vellankikoti/kubilitics/commit/d57b6b1b1dc01f1f934edb0a2a1e5785c4afa48a))
- Incident command warns when API data is incomplete by @vellankikoti ([43049fe](https://github.com/vellankikoti/kubilitics/commit/43049fe8ec86d34fa743211fd561abd9d8a52702))
- Search reports which contexts failed instead of silent partial results by @vellankikoti ([bdfe415](https://github.com/vellankikoti/kubilitics/commit/bdfe4151fdaf42f94d35712ded9565473887b59e))
- Show '? (error)' instead of blank when node API fails in where by @vellankikoti ([add1783](https://github.com/vellankikoti/kubilitics/commit/add1783825276fba3583c4145c291b0b4035355d))
- Detect destructive commands inside sh -c wrappers by @vellankikoti ([8e8577a](https://github.com/vellankikoti/kubilitics/commit/8e8577a79e38472e50d828d33fba59c0d0109516))
- Show helpful empty state messages instead of empty tables by @vellankikoti ([9808e87](https://github.com/vellankikoti/kubilitics/commit/9808e878ea7e8a1f26d83e02714a7b15b4106a60))
- Commit remaining fixes — show --help, exec tests, e2e script by @vellankikoti ([f11d6d9](https://github.com/vellankikoti/kubilitics/commit/f11d6d9bed37219384c44a97e077a7d0e6b09e81))
- Provider logos, delete cluster, status filters, health indicators by @vellankikoti ([a9c2c31](https://github.com/vellankikoti/kubilitics/commit/a9c2c31ab7a6dd2f38341c4f49d1721ed370e361))
- Replica penalty cliff, cross-ns off-by-one, dynamic normalization by @vellankikoti ([966a9b6](https://github.com/vellankikoti/kubilitics/commit/966a9b622a2214688a7308234dd55a2f4b77ebda))
- PDB matching uses pod labels instead of service name heuristic by @vellankikoti ([6faae9e](https://github.com/vellankikoti/kubilitics/commit/6faae9e3884fb304a891ffc7f8ff6c6b9162a981))
- Health widget showed "At Risk" for healthy clusters by @vellankikoti ([ab2bc0a](https://github.com/vellankikoti/kubilitics/commit/ab2bc0ac093a96fac1bd793f6908808d6f0d5de1))
- Fix all audit findings — wire CrashLoop/OOMKilled/pressure into scorer, fix status mapping by @vellankikoti ([8843842](https://github.com/vellankikoti/kubilitics/commit/88438429a711ece3bab3d6e514b9f03012c659c3))
- Show affected pods for restart storm alerts by @vellankikoti ([92520eb](https://github.com/vellankikoti/kubilitics/commit/92520eb133100c53578b404ef7a63e6d763049b8))
- Include affected pod names in restart storm detail by @vellankikoti ([1aebf28](https://github.com/vellankikoti/kubilitics/commit/1aebf2812942b069af04c86d2d3762dbdca2cdb5))
- Restart storm shows pod count, not event count by @vellankikoti ([189a242](https://github.com/vellankikoti/kubilitics/commit/189a24294e77a0968953b3c17ce8c5124d7ced25))
- Add missing useEffect import in HealthIssueDetail — was crashing on load by @vellankikoti ([c07f49c](https://github.com/vellankikoti/kubilitics/commit/c07f49ccc273313f8b3006cf3ebe537a03be0599))
- Namespace fallback builds pod info directly — fixes 0 restart count by @vellankikoti ([5743385](https://github.com/vellankikoti/kubilitics/commit/574338531466299916dab36657d63591205cfdbf))
- Age column sorts chronologically, server pagination resets on namespace change by @vellankikoti ([7e59261](https://github.com/vellankikoti/kubilitics/commit/7e5926189b78086fb614e62e0759a546b406b7c7))
- Re-sort merged multi-namespace results before pagination by @vellankikoti ([d2a06cb](https://github.com/vellankikoti/kubilitics/commit/d2a06cb9a1fc064b8defa04fb6a7a2369bc18357))
- Page reset on filter/sort/namespace change across all list pages by @vellankikoti ([feb2de4](https://github.com/vellankikoti/kubilitics/commit/feb2de46252e6d74808a92ab0c6ae53b288f266a))
- Nodes pagination actually slices, Pods stats filter-aware, ResourceListFactory uses standard pagination by @vellankikoti ([c390707](https://github.com/vellankikoti/kubilitics/commit/c390707a103fb34cd0d463c07f34b3c62cebed79))
- Stable sort, page-aware stat labels, compound age parsing by @vellankikoti ([4cc4135](https://github.com/vellankikoti/kubilitics/commit/4cc413590ea8257878c6804e5709e8da92fe98a0))
- Show NotReady when running but readiness probe failing — critical UX fix by @vellankikoti ([d323e65](https://github.com/vellankikoti/kubilitics/commit/d323e657bdaa2089673953c4cbd7ba90aa74609a))
- Remove misleading 99+ badge from Events tab by @vellankikoti ([9454697](https://github.com/vellankikoti/kubilitics/commit/94546977545c0edf6f5d9a086c4d7c1e14e94c7f))
- Receiver accepts gzip + json.Number for OTLP/HTTP from standard collector by @vellankikoti ([11849c9](https://github.com/vellankikoti/kubilitics/commit/11849c9add4ce8b7683fb5726b0740db76370140))
- TraceList queries the active cluster, not the first connected one by @vellankikoti ([0a0748c](https://github.com/vellankikoti/kubilitics/commit/0a0748c08ba2f176e87706296ef92b1ab74414f3))
- Preserve root_service across batches + show all services in row by @vellankikoti ([24aec9d](https://github.com/vellankikoti/kubilitics/commit/24aec9d0818354ce28dd9d848cf65cb71a3f575e))
- Resolve 4 pre-existing test failures unrelated to tracing refactor by @vellankikoti ([9983246](https://github.com/vellankikoti/kubilitics/commit/998324682a148690cc6844603a013e858e7b79a9))
- Surface connection errors instead of silent failures by @vellankikoti ([1f34037](https://github.com/vellankikoti/kubilitics/commit/1f3403786c4e790a654427d3cc506287a24bc885))
- Allow tab switching + copy when InstallCommandBlock is dimmed by @vellankikoti ([2be93dc](https://github.com/vellankikoti/kubilitics/commit/2be93dc01f3f4e37daecd2287eec9d322fb5435b))
- Wrap long stat-card labels instead of clipping mid-word by @vellankikoti ([e4f283a](https://github.com/vellankikoti/kubilitics/commit/e4f283ac0ed82b5df51f8e908ea84bf3b96450c4))
- Session restore no longer clobbers fresh connection by @vellankikoti ([33c182e](https://github.com/vellankikoti/kubilitics/commit/33c182ebfd3587c22bb4be38be0252452d082d07))
- Lazily reconnect cluster client when pool miss by @vellankikoti ([437a5dc](https://github.com/vellankikoti/kubilitics/commit/437a5dc22fab04111f056c2a9236f74d2b1200ac))
- Coalesce and lock lazy reconnect path; add negative cache by @vellankikoti ([279e2fd](https://github.com/vellankikoti/kubilitics/commit/279e2fd2e268246177bd842ac3e4d4177b88d13c))
- Post-review cleanup — revert 1440 cap, fix rowHeight, URL-safe reload, dead code by @vellankikoti ([4781537](https://github.com/vellankikoti/kubilitics/commit/47815374a8636172eeb1148b56f7af2532df44ba))
- Complete monorepo move — add entry point files by @vellankikoti ([4a80401](https://github.com/vellankikoti/kubilitics/commit/4a804016863c4f7487c5b55c0cef78c606174789))
- Migration 050 backfill pattern handles Windows paths by @vellankikoti ([60f4ea5](https://github.com/vellankikoti/kubilitics/commit/60f4ea5e651851015b49ba2806883d38bedf1c54))
- Close Source-tagging race + audit restore ops by @vellankikoti ([45135c1](https://github.com/vellankikoti/kubilitics/commit/45135c15affded70163f1fa38544d452c81beb87))
- Surface event.message in collapsed EventRow by @vellankikoti ([4a60a2b](https://github.com/vellankikoti/kubilitics/commit/4a60a2bdb193231621c23652e5ed0044ca35b5c6))
- Pass kind explicitly into diagnoseWorkload by @vellankikoti ([4a4a2d3](https://github.com/vellankikoti/kubilitics/commit/4a4a2d32ef186da28b92d1862c8028552defd54d))
- Exclude init containers from healthy oneLine count by @vellankikoti ([a410265](https://github.com/vellankikoti/kubilitics/commit/a41026515aa395d50a6609d45a45cc501b72f71a))
- Make cluster-id resolution bulletproof for counters by @vellankikoti ([7ea151e](https://github.com/vellankikoti/kubilitics/commit/7ea151e5f20580c410ce9fd87f673d25c8931ea8))
- React to external cluster delete within a frame by @vellankikoti ([ab2ed05](https://github.com/vellankikoti/kubilitics/commit/ab2ed05a2b4203fd153a6bf5e69de62a5d5ff9f8))
- LineCount counts displayYaml so header stays accurate in Clean mode by @vellankikoti ([47239d0](https://github.com/vellankikoti/kubilitics/commit/47239d0dbdefdec36d3f7a2c0f13c90bc9f1351b))
- Active pill contrast + well-known fold actions actually fold by @vellankikoti ([d821d75](https://github.com/vellankikoti/kubilitics/commit/d821d75f59cbafe296708541722f311e8018a486))
- Compile-time guard against devtools feature in release builds by @vellankikoti ([d9c0e70](https://github.com/vellankikoti/kubilitics/commit/d9c0e7035a2ad796546e5c264560e175dea220e7))
- ReconnectCluster never falls back to system kubeconfig by @vellankikoti ([78c05d2](https://github.com/vellankikoti/kubilitics/commit/78c05d2fe9acf48905b1cc66a6dd74154183ff7c))
- Orphan upload-sourced clusters whose file is gone by @vellankikoti ([16f80f6](https://github.com/vellankikoti/kubilitics/commit/16f80f6eeaebe4fa7a6b668b40a7f4acd947752d))
- Await reconnect before navigating to Dashboard by @vellankikoti ([ed6d5ec](https://github.com/vellankikoti/kubilitics/commit/ed6d5ec92b4c9b0b974cd86e3e02bf54b075fcea))
- Terminal/Logs tabs find pods via labels not ownerReferences by @vellankikoti ([ce00a42](https://github.com/vellankikoti/kubilitics/commit/ce00a4269249f7ae15227c230788a69923306996))
- Clear React Query cache on every cluster switch by @vellankikoti ([894ef3a](https://github.com/vellankikoti/kubilitics/commit/894ef3a96c3b058d5a7302c94a13a6dbd61cfca2))
- Gate Events/Traces tab content on resource name being loaded by @vellankikoti ([93d128e](https://github.com/vellankikoti/kubilitics/commit/93d128e83a41899984b40c7190222dd0ab8491ba))
- Show 'No cluster connected' instead of 'Create X' when disconnected by @vellankikoti ([e71919f](https://github.com/vellankikoti/kubilitics/commit/e71919f9d9d53087140049a41937d3705902f81c))
- Gate all fallback count queries on isConnected by @vellankikoti ([088adc4](https://github.com/vellankikoti/kubilitics/commit/088adc4b69c08809dc426ac321e98cba3b2f9327))
- Running+ready pods with past crashes are healthy, not broken by @vellankikoti ([430c2a0](https://github.com/vellankikoti/kubilitics/commit/430c2a00418c7388253222120c5cba0bccc09b20))
- Explicit namespace overrides project filter in resource list queries by @vellankikoti ([48f08e3](https://github.com/vellankikoti/kubilitics/commit/48f08e373239b59b9d4fcaf75424ce0771479d16))
- Restore labelSelector pod query (proven working approach) by @vellankikoti ([b7edbb7](https://github.com/vellankikoti/kubilitics/commit/b7edbb763ea16c2982fbdb91db4a6dc8cb069fa3))
- Self-contained DeploymentTerminalTab + disable Dependabot by @vellankikoti ([b8e77b0](https://github.com/vellankikoti/kubilitics/commit/b8e77b009425ff188292276a6b95179e553fa31c))
- Regenerate signing keypair + set on both repos by @vellankikoti ([a1b8166](https://github.com/vellankikoti/kubilitics/commit/a1b816655ab6ff5166da2db1f493b1f6dae04a7d))
- Pass Apple notarization env vars to Tauri build step by @vellankikoti ([f2c24fc](https://github.com/vellankikoti/kubilitics/commit/f2c24fc6f7c7086140f5c9e0339b9a23cfcb641a))
- Patch 3 critical + 12 high CVEs in dependencies by @vellankikoti ([7758844](https://github.com/vellankikoti/kubilitics/commit/7758844a9d7d8baca6ac8f9791c8024ac26a139d))
- Resolve Go + frontend code scanning alerts by @vellankikoti ([ec6d67b](https://github.com/vellankikoti/kubilitics/commit/ec6d67b211c884941ce59bf121c4d3241717bccb))


### Misc

- Add .superpowers and temp files to gitignore by @vellankikoti ([5ff9db9](https://github.com/vellankikoti/kubilitics/commit/5ff9db9d3f83510b425c5752becab672d824a564))
- Update Cargo.lock and generated schemas for updater plugin by @vellankikoti ([d3b7e1b](https://github.com/vellankikoti/kubilitics/commit/d3b7e1b50cebea5709cb660e1a4def233566b6b1))
- Update Cargo.lock by @vellankikoti ([d912f27](https://github.com/vellankikoti/kubilitics/commit/d912f27f0298b2b52cc73f10572abbc0fdb9ff35))
- Bump version to 0.1.2 by @vellankikoti ([938260a](https://github.com/vellankikoti/kubilitics/commit/938260a133bbb401a52f06620904b50cb5ddaf01))
- Bump all 6 version files to 0.1.2 by @vellankikoti ([8a10834](https://github.com/vellankikoti/kubilitics/commit/8a10834dc314581004d137b863d255d4e8c088af))
- Bump to v0.1.3 — EKS/GKE/AKS PATH fix by @vellankikoti ([b838652](https://github.com/vellankikoti/kubilitics/commit/b838652c37d6810ee8b2a514c497481e3bf8fd3d))
- Release v0.1.5 by @vellankikoti ([4ccda84](https://github.com/vellankikoti/kubilitics/commit/4ccda84310989db44763bb2b3a7c8131f581055a))
- Bump version to 0.2.0 across all 6 version files by @vellankikoti ([880eeed](https://github.com/vellankikoti/kubilitics/commit/880eeedebd4d286dc908c79daf86bbca45b6be17))
- Add .gitignore for trace-agent binary by @vellankikoti ([b623161](https://github.com/vellankikoti/kubilitics/commit/b623161c2f1c14889c548b010bfcb003ee5fba9e))
- Add .worktrees to gitignore by @vellankikoti ([8cbdbfc](https://github.com/vellankikoti/kubilitics/commit/8cbdbfc6972fe8b5078fe3cf7574dc29b399d168))
- Release v0.2.5 by @vellankikoti ([41d46fb](https://github.com/vellankikoti/kubilitics/commit/41d46fb81747eeda620b5cfb9163ad693fc8cc28))
- Copy kcli source into monorepo as kubilitics-cli by @vellankikoti ([ebe4bde](https://github.com/vellankikoti/kubilitics/commit/ebe4bde5054222c419c26dd21df00025282f4918))
- Update kcli module path for monorepo by @vellankikoti ([cbf5295](https://github.com/vellankikoti/kubilitics/commit/cbf5295674b80fd7fe30aec794be87dd1c5cfcb1))
- Update all kcli import paths for monorepo by @vellankikoti ([1088c04](https://github.com/vellankikoti/kubilitics/commit/1088c04b5a47e74fc733bb3a0ced9c0cce693bb6))
- Consolidate kcli gitignore into monorepo by @vellankikoti ([36d2209](https://github.com/vellankikoti/kubilitics/commit/36d22095050c00c042d7fada63ddf249a2831bc9))
- Delete agent_manifests.go and puller.go (replaced by Helm chart) by @vellankikoti ([3c5de11](https://github.com/vellankikoti/kubilitics/commit/3c5de111733efa7ef28a5daf30662a984801d89c))
- Sync package-lock.json after phase 4 install by @vellankikoti ([9ca9af4](https://github.com/vellankikoti/kubilitics/commit/9ca9af4d5fb61fc11942b651dc689df70365a866))
- Tailwind ease-class warnings + automated chart sync by @vellankikoti ([267df5c](https://github.com/vellankikoti/kubilitics/commit/267df5cd6603027a1be80938d15c90a1e7f43c5b))
- Drop stale kcli/ rule after monorepo move by @vellankikoti ([882e792](https://github.com/vellankikoti/kubilitics/commit/882e79297580f4f9ad3961a33a70e5f768c0a77d))
- Go mod tidy — add 26 missing go.sum entries by @vellankikoti ([36632ed](https://github.com/vellankikoti/kubilitics/commit/36632edd9057b191ce594442b7193d8a0390ee16))


### Performance

- Parallelize 30 K8s API calls in GetClusterSummary by @vellankikoti ([3dd8ca4](https://github.com/vellankikoti/kubilitics/commit/3dd8ca417320ea9d06e268d328248fd7ac2da4f7))


### Tests

- Add integration tests for real-world scenarios by @vellankikoti ([1d2aa38](https://github.com/vellankikoti/kubilitics/commit/1d2aa38c6a801ffbf125fd8660c6ea230c743830))
- Add unit tests for ingress, scaling, rbac, selector, dns matchers by @vellankikoti ([3eb915f](https://github.com/vellankikoti/kubilitics/commit/3eb915fa4f9a4dc5e4d6703926c2aa488fa52883))
- Add 12 test scenarios — circuit breakers, penalty deductions, edge cases by @vellankikoti ([8d89a67](https://github.com/vellankikoti/kubilitics/commit/8d89a67a7519f73fbb83a17b404b96e0f084ecfd))
- Implement all 7 integration scenarios + node drain test — zero skips remaining by @vellankikoti ([15f4d7b](https://github.com/vellankikoti/kubilitics/commit/15f4d7b7237a02b69a4d930406d09da61d297dec))
- Cover diagnostics database lookups and invariants by @vellankikoti ([d33e044](https://github.com/vellankikoti/kubilitics/commit/d33e0442ad566da1339e18d1d328fb9f76d7b7ce))
- Assert Instrumentation CR and air-gap image override render correctly by @vellankikoti ([6dd3e46](https://github.com/vellankikoti/kubilitics/commit/6dd3e46dc8e79da9b3e4bea75bc40c6377308af4))
- Shared fixtures for diagnose tests by @vellankikoti ([e05321c](https://github.com/vellankikoti/kubilitics/commit/e05321cdc696f184ea2e4fa68bdaa736a6161bb5))
- Align standard-tabs assertion with actual tab list by @vellankikoti ([5c4a5b0](https://github.com/vellankikoti/kubilitics/commit/5c4a5b06419a93c5a4dbd4eb13f2661958b7922a))
- Pin namespace-vs-project priority in useK8sResourceList by @vellankikoti ([e0ed704](https://github.com/vellankikoti/kubilitics/commit/e0ed704aee7d9f818be4c5a19eb5ddaac38dc695))

