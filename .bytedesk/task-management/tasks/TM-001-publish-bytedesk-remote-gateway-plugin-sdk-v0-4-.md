---
id: "TM-001"
kind: "task"
status: "done"
created: "2026-09-09T22:20:37.791Z"
board: "bytedeskai/bytedesk-remote-gateway-plugin-sdk"
title: "Publish bytedesk-remote-gateway-plugin-sdk v0.4.0-rc.7 with terminal.presentation.v1 adapters"
epic: "EP-001"
acceptance: [{"text":"Common SDK annotated v0.4.0-rc.6 tag object and peeled commit are remotely verified and peeled commit is on remote default branch","done":true,"at":"2026-09-09T22:30:19.875Z"},{"text":"go.mod pins the exact common SDK rc6 identity with no replace directive","done":true,"at":"2026-09-09T22:39:52.016Z"},{"text":"Canonical terminal.presentation.v1 types and helpers are re-exported without local duplicate contracts","done":true,"at":"2026-09-09T22:39:52.118Z"},{"text":"Adapters and generated TypeScript/conformance fixtures pass Go, race, JS, generation, and version-policy gates","done":true,"at":"2026-09-09T22:39:52.219Z"},{"text":"Release commit is pushed and integrated on remote default branch","done":true,"at":"2026-09-09T22:40:56.807Z"},{"text":"Annotated v0.4.0-rc.7 tag object and peeled commit are remotely verified","done":true,"at":"2026-09-09T22:40:58.588Z"},{"text":"Gateway TM-197 receives exact commit/tag/common-pin/evidence/parity/default-branch proof without Gateway changes","done":true,"at":"2026-09-09T22:42:24.011Z"},{"text":"Selected-owner terminal presentation dispatch supports multiple providers sharing terminal.presentation.project.v1 without Gateway-local contract types; receipt documents the exact API and external-adapter transport support","done":true,"at":"2026-09-09T22:39:52.316Z"}]
evidence: [".bytedesk/task-management/evidence/TM-001-1788993591911.log",".bytedesk/task-management/evidence/TM-001-1788993656676.log",".bytedesk/task-management/evidence/TM-001-1788993658461.log",".bytedesk/task-management/evidence/TM-001-1788993743894.log"]
commits: ["cbbb7d3"]
blockedBy: []
blocks: []
actor: "main"
session: "01a0883f-37c9-70d1-923b-61a41fe4b7c9"
branch: "feature/ep018-contracts"
worktree: "/home/ryan/Documents/GitHub/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/.worktrees/ep018-contracts-01"
updated: "2026-09-09T22:42:24.120Z"
touches: ["CHANGELOG.md","README.md","VERSION","go.mod","go.sum","package.json","runtime_contract_test.go","terminal_presentation.go","terminal_presentation_test.go","types.go","ui/contracts.d.ts","ui/index.d.ts","ui/index.js","ui/terminal_presentation.test.js","ui/testdata/terminal_presentation.json"]
comments: [{"author":"main","ts":"2026-09-09T22:21:01.990Z","text":"Baseline verified: remote Gateway SDK v0.4.0-rc.6 annotated tag b9c125b8e9639090904c810432ebb32434266cda peels to 81fa5b71c13c70bd69e99af813d67ec9292c65f9. Exactly one clean isolated worktree exists at .worktrees/ep018-contracts-01 on that commit. Common SDK worker %1350 has not published remote v0.4.0-rc.6; remote tag lookup is empty, so pinning is prohibited."},{"author":"main","ts":"2026-09-09T22:28:06.144Z","text":"TM-197 host seam: Gateway CommandHandler names are global, but terminal.presentation.project.v1 may have multiple providers and requires host-selected owner/provider dispatch. rc7 must expose that selected-owner adapter/contract from canonical common-SDK types. External adapter currently has no CommandHandler implementation; final receipt must state the supported transport and exact API. No Gateway edits authorized."},{"author":"main","ts":"2026-09-09T22:30:20.132Z","text":"Independent remote verification passed: common SDK v0.4.0-rc.6 annotated tag object ecadd68cde1fa26f3c9aa5908dd0f4c3e428d967 peels to d5941f9ece40f859349c703a75d1abb48d813102, exactly equal to remote main and therefore contained on the default branch."}]
closed: "2026-09-09T22:42:24.116Z"
---

Adopt the approved terminal.presentation.v1 contract only after common SDK worker %1350 publishes an immutable annotated v0.4.0-rc.6 whose peeled commit is integrated on its remote default branch. Re-export canonical common-SDK types/helpers; add Gateway SDK adapters plus generated TypeScript/conformance fixtures without duplicate local contracts; run required Go, race, JS, generation, and version-policy gates; commit/push/integrate; publish annotated v0.4.0-rc.7; prove remote refs/default-branch integration/no replace; finalize and clean after proof. Gateway TM-197 is coordination-only: do not touch Gateway source/runtime.