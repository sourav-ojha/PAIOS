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

---

## P1 — will cause rework or measurable cost

### G6. Ingestion is listed but never designed
§12 names 12 source types. No mention of: incremental sync vs. full re-index, deletion propagation (a doc deleted at source stays in the index forever → the briefing cites deleted facts), rate limits, auth token rotation, or what happens when a connector breaks silently. Each connector is a permanent maintenance liability; twelve is a part-time job.

**Close it:** one connector in Phase 4, chosen by what the briefing starved for. Design deletion propagation before the second.

### G7. Retrieval quality has no evaluation harness
The system's usefulness *is* retrieval quality, and there's no way to tell if a change made it worse. Every prompt tweak and every reindex is then a blind change.

**Close it:** 20 golden question→expected-doc pairs in a file. Run before/after any knowledge-layer change. This is an afternoon of work and it's the difference between engineering and guessing.

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
