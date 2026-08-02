# Gap Analysis — what `initial-plan.md` does not address

**Method:** read the document as a spec and list what an implementer would be blocked on, what would silently go wrong, and what will hurt in month six. Ranked by severity.

Severity: **P0** = blocks Phase 1 or causes irreversible harm · **P1** = causes rework or real cost · **P2** = will hurt later, cheap to defer.

---

## P0 — must resolve before building

### G1. No definition of what the briefing actually contains
The entire project's value rests on §10 "prepare daily briefings," and the document never specifies the output. No sections, no inputs, no length, no ordering rule, no "what makes a good briefing vs. a bad one." This is the deliverable, and it's a blank.

Without it you cannot tell whether Phase 4 succeeded, and you'll build ingestion for data the briefing never uses.

**Close it:** write one briefing by hand, today, for tomorrow morning. That artifact is the spec. Mine it for required inputs.

### G2. No prompt-injection threat model
The plan ingests GitHub issues, email, Slack, Discord, and web content (§12) — all attacker-writable — into a memory system whose contents are fed to agents that can execute code and open PRs (§17 Phase 5). §16 Security discusses secrets and encryption but never mentions injection. This is the most likely way the system causes real damage.

**Close it:** trust labels on ingested content, untrusted content delimited as data, memory writes gated. See TDD §6 T1/T4.

### G3. Workspace isolation has no enforcement mechanism
§7 and §13 state isolation as a requirement. Nothing in the architecture enforces it — the knowledge layer chosen (§6 L1) provides scope conventions, not boundaries. See Architecture Review §2.

**Close it:** isolation = process + database boundary. Verify with a negative test, not by inspection.

### G4. "Human in the loop" is never mechanized
§5 says "AI executes, human decides." §10 says "escalate only high-value decisions." Nowhere is there an approval mechanism, an escalation threshold, or a definition of "high-value." Absent this, the system either asks about everything (useless) or nothing (dangerous), and which one is decided accidentally by prompt wording.

**Close it:** enumerate the irreversible actions (merge PR, send message, delete, spend over $X, write to isolated-workspace memory) and require explicit approval for each. Everything else proceeds. This is a list, not a framework.

### G5. No cost enforcement, only cost intentions
§15 lists nine good practices, zero controls. First runaway agent loop on a large repo produces a surprise bill.

**Close it:** hard per-workspace monthly cap checked before every call, fails closed. TDD §5.

### G7. Synthesis quality has no evaluation harness — promoted from P1, confirmed 2026-08-02
Originally flagged as a hypothetical ("every prompt tweak and every reindex is a blind change"). No longer hypothetical: three free local models (`gemma3:1b`, `qwen2.5-coder:7b`, `gemma4:12b`) were tested against one fully-grounded extraction question and all three got it wrong — two fabricated confident wrong answers, one denied a fact stated in the source doc's own title. Retrieval was correct in every case; this was caught only because the answer happened to be checked by hand against the source ADR. Without that check, a wrong answer ships silently in a briefing — exactly the failure mode this project exists to prevent (initial-plan.md §10: "trust its recommendations").

**Close it:** 20 golden question→expected-answer pairs, not just question→expected-doc — retrieval-only eval isn't enough, the finding was a synthesis failure with correct retrieval. Include at least one "trap" question per doc type modeled on the real failure: a plausible-but-wrong alternative sitting near the correct answer in the same source (here, "Prisma" listed as a rejected alternative one paragraph from "Drizzle," the actual answer). Run before promoting any model — local or cloud — to default. TDD §5/§5.1.

---

## P1 — will cause rework or measurable cost

### G6. Ingestion is listed but never designed
§12 names 12 source types. No mention of: incremental sync vs. full re-index, deletion propagation (a doc deleted at source stays in the index forever → the briefing cites deleted facts), rate limits, auth token rotation, or what happens when a connector breaks silently. Each connector is a permanent maintenance liability; twelve is a part-time job.

**Close it:** one connector in Phase 4, chosen by what the briefing starved for. Design deletion propagation before the second.

### G8. Nothing describes memory *conflict* resolution
§8/§9 cover categories and expiry. Neither covers: two documents state contradicting facts; a decision is reversed (Redis rejected in March, adopted in July — §8's own example); a fact is superseded but not deleted. The briefing will confidently state stale conclusions. Temporal-graph systems (Graphiti) exist specifically for this; GBrain's model is weaker here.

**Close it:** decisions are dated, immutable, append-only ADRs; a superseding decision links its predecessor. Retrieval prefers the most recent. Cheap convention, avoids a hard technical problem.

### G9. Multi-machine story is absent
GBrain's brain is a git repo + a local index. The doc mentions VPS deployment (§5) and implies laptop use. Two machines = index divergence and git conflicts on agent-written memory.

**Close it:** decide now — one authoritative instance on the VPS, CLI talks to it remotely. Do not sync indexes.

### G10. No observability
Nothing on logs, traces, or "why did the briefing say that?" Debugging an LLM system without transcript retention is guesswork.

**Close it:** persist every prompt/response with retrieval doc ids. `audit.jsonl` + transcript files. TDD §3.2.

### G11. Idea incubation underspecified despite being the best idea in the doc
§11 lists nine questions but no state model: where does a half-interviewed idea live, can you resume, what happens to rejected ideas, what promotes an idea to a project.

**Close it:** TDD §4.3 state machine.

### G12. §18 deliverables are unfunded
Nine artifacts demanded per phase (risk assessment, cost estimation, rollback strategy, testing strategy…). Eight phases × nine artifacts = 72 documents. This will not happen, and pretending it will means none of them get done properly.

**Close it:** per phase, one page: what shipped, what it cost, how to undo it, acceptance results. ADRs for decisions only.

### G20. Search-result chunks are not the same as page content — found 2026-08-02
Not anticipated in the original plan at all. GBrain's `query` tool ranks and returns `chunk_text` fragments for relevance scoring; these can cut off mid-document (observed: an ADR's chunk ended right after its frontmatter, before the section holding the actual decision reasoning). Code that synthesizes answers directly from `chunk_text` is reading a fragment, not the document, and a model will fabricate plausible content to fill the gap rather than say "I don't have this."

**Closed:** resolve every retrieved slug through `get_page` (full `compiled_truth`) before it reaches a model or a human. TDD §3.4. Applies to any future code that touches GBrain's `query` results, not just `cos`.

### G21. "Survives restarts" and "survives an infra change" are different claims — found 2026-08-02
The original persistence design (TDD §7) proved out named volumes surviving `docker compose down`/`up`. It did not anticipate — and got tested by — a Docker engine swap (Docker Desktop → OrbStack) mid-project. The named volume did not carry over; a fresh empty one was created under the same name. No data was actually lost, because brain markdown lives on the host filesystem independent of Docker, but the recovery step (re-import, re-mint tokens) is real operational work that wasn't documented as something to expect.

**Closed:** TDD §7 now states the boundary explicitly. Worth generalizing: any host migration, engine swap, or "reinstall Docker to fix something" event should be treated as equivalent to a fresh install for every containerized service — the only thing that should survive by default is what's on the host filesystem (bind mounts, git repos), never what's in a named volume.

---

## P2 — defer, but note

- **G13. No conversation history model.** Chat history is listed as a *source* (§12) but there's no design for the Chief of Staff's own conversational continuity across days.
- **G14. Agent failure handling.** No retries, no timeouts, no partial-failure semantics, no "agent produced garbage" path.
- **G15. Knowledge quality decay.** "Knowledge Curator" is named but its actual job (dedupe, staleness detection, broken-link repair) is undefined. GBrain does some of this natively — verify what, before building any of it.
- **G16. No local-model path.** "Vendor independence" (§5) but every design point assumes a hosted API. If the office workspace can't use hosted LLMs (TDD §10.1), you need an Ollama path — cheap to add now, expensive to retrofit.
- **G17. Onboarding a new workspace is undefined.** Adding "Client C" should be one command. Unspecified, it becomes a 20-step manual ritual.
- **G18. No metric for "reduced context switching"** (§3) — unfalsifiable as written, so it can't fail, so it can't inform anything.
- **G19. Secrets in workspaces (§7 lists "Secrets" as workspace content)** — but the knowledge layer must never hold secrets (§16). Direct contradiction in the source document; resolve by keeping secrets in env/vault and storing only *references* in the brain.

---

## Cross-cutting observation

Twelve of these nineteen gaps share one root cause: **the document specifies structure (layers, agents, sources, categories) but not behavior (inputs, outputs, failure modes, verification).** The fastest way to close most of them is to build the smallest end-to-end path — Phase 0 in the TDD — because behavior gaps become obvious the moment something runs, and stay invisible for as long as it's a diagram.
