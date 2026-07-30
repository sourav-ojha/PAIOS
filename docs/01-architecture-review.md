# Architecture Review — PAIOS / Chief of Staff

**Reviewer role:** Principal Software Architect
**Input:** `initial-plan.md` v0.1
**Date:** 2026-07-30
**Verdict:** Vision sound. Architecture as written will not ship. Scope is ~10x what one person sustains, and three named assumptions are wrong.

---

## 0. Summary judgment

The document is a good *product constitution* and a poor *architecture*. It describes what the system should feel like, then jumps to a 5-layer diagram and an 8-phase roadmap without ever stating the one thing architecture exists to state: **what is the smallest system that delivers the morning briefing, and what is deliberately excluded.**

Three specific problems, in severity order:

1. **The workspace model and the chosen knowledge layer are structurally incompatible.** (§2)
2. **The agent roster is cargo-culted, not derived from work.** (§3)
3. **The roadmap defers the only thing that proves the thesis to Phase 4.** (§4)

Everything else is fixable detail.

---

## 1. What the document gets right

Genuinely correct calls, worth protecting from later revision:

| Principle | Why it's right |
|---|---|
| "Knowledge is permanent, execution is temporary" | This is *the* correct load-bearing insight. Agent runtimes have a ~12-month half-life; your notes don't. Everything else should bend to keep this true. |
| Open-source first / no rebuild | Correct, and the doc's own §4 non-goals police it well. |
| Cost-first, cheap-model-default | Correct, though the doc has no mechanism to enforce it. See Gap Analysis G7. |
| Docker Compose, local-first | Right call for a single operator. Resist Kubernetes for at least 2 years. |
| Idea incubation via interview (§11) | The most original idea in the document and the most underrated. This is a differentiator; it deserves to be Phase 1, not Phase 4. |
| Human-in-the-loop as a principle | Correct, but stated as a slogan rather than a mechanism. See G4. |

---

## 2. Challenge 1 — GBrain cannot satisfy your workspace model (critical)

**The document's assumption:** §6 Layer 1 names GBrain as the knowledge layer. §7 requires workspace isolation. §13 requires "no credential leakage between workspaces." §2 (Layer 2) treats workspace isolation as a layer that sits *on top of* the knowledge layer.

**Why this is wrong:** GBrain is architecturally a **single-operator** system. Its own docs describe the personal brain as one git repo, one agent. Its multi-user story ("company brain") is *scope-based separation within a single brain*, enforced by per-client OAuth credentials deciding which **sources** you may read/write — not tenant isolation. Isolation is a credential convention, not an architectural boundary.

**Why it matters for you specifically:** Your §13 requirement is *credential separation between Personal / Office / Client A / Client B*. That is exactly the guarantee scope-based source separation does not give you. If Client A's proprietary architecture notes and your Office employer's internal docs live in one brain with one index, then:

- one prompt-injected document in Client A's repo can exfiltrate Office context, because retrieval is global and ranking is content-driven;
- a bad retrieval on a Client B question surfaces Client A material into a model call — a contractual problem, not a bug;
- you cannot hand a client their data, or delete it on engagement end, without surgery on a shared index.

This isn't a GBrain defect. It's a mismatch: it was built for one person with one brain, and you are describing a multi-tenant system where the tenants happen to all be you.

**Recommendation — invert Layers 1 and 2.**

> Workspace is not a layer above knowledge. Workspace is the **unit of deployment** of knowledge.

One brain instance *per workspace*, not one brain with workspace tags. Concretely: one GBrain instance (own Postgres schema or own container + own git repo + own credentials) per workspace, with a thin **Router** in front that resolves "which workspace am I in?" before any retrieval happens. Isolation then comes from the process/DB boundary — the only kind that survives a prompt injection and an LLM's bad judgment.

Costs of this inversion, stated honestly:
- Cross-workspace queries become an explicit federated fan-out you must build (~200 LOC + a merge/rank step). The doc says cross-workspace sharing should be *intentional* — so this cost is aligned with the stated principle, not fighting it.
- N instances = N migrations, N backups. Mitigate with one Compose template + a loop, and accept it: you will have ~8 workspaces, not 800.
- PGLite is single-machine/embedded; a per-workspace fleet pushes you to real Postgres earlier. Do that on day one; don't migrate later.

**Alternative if the inversion feels heavy:** two-tier. One shared brain for `Personal`/`Learning`/`Ideas` (leakage between these is harmless — it's all you), plus one isolated instance per *client or employer* workspace where leakage has legal consequence. This gets 90% of the safety for 30% of the ops. **This is my recommendation for Phase 1.**

---

## 3. Challenge 2 — the nine agents are a cargo cult

**The document's assumption:** §3/§14 define nine named workers: Chief of Staff, Developer, Reviewer, Researcher, Documentation, DevOps, QA, Security, Knowledge Curator.

**Why this is wrong:** This roster is an org chart copied from a software company, not a decomposition of your actual work. It mistakes *job titles* for *system boundaries*. The tell: every one of the nine is described in three words or fewer ("Security — Reviews risks", "QA — Validates implementations"). No inputs, no outputs, no trigger, no success criterion. A component you can't specify in a paragraph isn't a component; it's an aspiration.

The concrete costs of premature agent proliferation:

- **Routing becomes the hard problem.** With 9 agents, the Chief of Staff's real job becomes classification — and misrouting is invisible and expensive. You will spend more time debugging "why did Researcher handle this?" than the agents save.
- **Nine prompts to maintain, drifting apart.** Each is a config surface with its own regressions.
- **Most of them are one tool call.** "Security reviews risks" is `semgrep` + a model call. Wrapping it in an agent identity adds a persona and a handoff, no capability.
- **Subagents already exist.** Claude Code/Agent SDK spawns task-scoped subagents natively. Re-implementing this as durable named "employees" is precisely the "another agent framework" your own §4 forbids.

**Recommendation — derive agents from friction logs, don't declare them.**

Ship **two** roles:
1. **Chief of Staff** — the only conversational surface. Reads memory, briefs, asks, delegates.
2. **Worker** — a generic, stateless executor: takes a task spec + workspace context, runs in a sandbox, returns a diff or a document.

Then keep a friction log for 4 weeks. Promote a specialist only when a distinct *prompt + toolset + evaluation criterion* has empirically diverged from generic Worker — the promotion criterion should be written down before you start. My prediction: Reviewer and Researcher earn it; DevOps, QA, Security, and Documentation stay as *skills/checklists* invoked by Worker, never as standing agents. Knowledge Curator is not an agent at all — it's a cron job (and GBrain already does overnight enrichment natively; don't rebuild it).

---

## 4. Challenge 3 — the roadmap sequences by architecture layer, not by risk

**The document's assumption:** §17 goes infra → workspace → ingestion → Chief of Staff MVP → execution → specialists → learning.

**Why this is wrong:** This is bottom-up construction. It builds three phases of foundation before anything answers a question. Two consequences:

- **The core hypothesis is tested last.** Your actual bet is *"an AI with my accumulated context will tell me something useful about my day that I didn't already know."* That is a genuinely uncertain claim — it may just produce a plausible-sounding list you learn to ignore. Phases 1–3 are weeks of work whose value is entirely contingent on that untested bet. Standard risk-first sequencing says: test it in week one, with 20 hand-copied markdown files and no infrastructure.
- **You will build the wrong ingestion.** Phase 3 ingests GitHub, markdown, PDF, Obsidian, Slack, Discord, email, blogs, books, papers. You cannot know which of those the briefing actually needs until the briefing exists. Building all of them first guarantees most of that work is waste, and each connector is a permanent maintenance liability (auth rotation, API drift).

Also: **§17 Phase 0 says "no coding," and that's a trap.** The cheapest way to challenge these assumptions is a 200-line spike, not a longer document. This review is worth less than a week of running the thing.

**Recommendation — resequence by hypothesis, gated on kill criteria.**

| Phase | Deliverable | Kill criterion |
|---|---|---|
| **0 (week 1)** | Manual briefing. 20 markdown files in a folder, GBrain single instance, one prompt, output to terminal. Zero infra. | If a hand-fed briefing isn't useful, **stop the project.** No amount of infra fixes it. |
| **1 (weeks 2–4)** | Same, but daily and automatic. Cron + real Postgres + backup. You use it every morning. | If you stop opening it within 2 weeks, the concept fails — see Success Metric §3.1. |
| **2** | Workspace isolation, per the §2 inversion. Driven by the first real client workspace. | — |
| **3** | Idea incubation interview. Highest-value/lowest-risk feature in the doc; needs no execution layer. | — |
| **4** | *One* connector — whichever the briefing actually starved for. Then stop and re-evaluate. | — |
| **5+** | Execution layer, specialists, learning loops — only after 3 months of the briefing being load-bearing. | — |

Note what this does: it moves the *irreversible, expensive, lock-in-prone* work (connectors, execution, specialists) behind the *cheap, reversible* work that validates the premise.

---

## 5. Lesser challenges

**5.1 — Layer 3/Layer 4 boundary is undefined.** Layer 3 is "AI workers," Layer 4 "runs tools, calls LLMs, executes workflows." But an agent *is* a prompt plus a tool loop plus LLM calls. As drawn, the layers aren't separable, so the boundary will leak immediately and the "replaceable execution engine" principle will silently die. **Fix:** define the seam as a concrete contract — `TaskSpec {workspace_id, goal, context_refs[], allowed_tools[], budget} → Result {artifacts[], transcript, cost}`. Anything that satisfies it (Claude Agent SDK, OpenHands, a bash script) is an execution engine. That contract, ~40 lines of schema, is the single highest-leverage artifact for vendor independence — write it before any code.

**5.2 — "Vendor independence" is asserted, not designed.** §5 says no irreversible lock-in, but the plan puts the knowledge graph, the ingestion, and the retrieval semantics inside one third-party tool. Real portability means: markdown files are the source of truth (GBrain's markdown-first design is a genuine plus here — protect it), the index is a derived artifact you can rebuild from scratch, and no schema you can't re-derive. **Test:** can you `rm -rf` the database and rebuild from git? If not, you have lock-in regardless of license.

**5.3 — Cost-first has no enforcement mechanism.** §15 lists nine good intentions and zero controls. Intentions do not survive contact with a 200k-token repo. **Fix:** a per-workspace monthly budget in the router, cost logged per task on the `TaskSpec` contract, and a hard cap that fails closed. Cheap-model default is a routing rule with a measured escalation trigger, not a preference.

**5.4 — Memory taxonomy is borrowed from cognitive science and may not pay rent.** Semantic/procedural/episodic (§8) is a clean framework that will produce classification arguments with yourself at 11pm. Ask what *decision* the label changes. If retrieval treats all three identically, the taxonomy is decoration. §9's lifecycle (permanent/project/temporary) is the one that earns its keep — it drives expiry, which is real behavior. **Fix:** ship §9, defer §8 until a retrieval behavior actually depends on it.

**5.5 — Success metrics are unfalsifiable.** "I trust its recommendations," "saves multiple hours weekly" (§3) can't be checked. **Fix:** one hard metric — *"for 10 consecutive working days I open the briefing before my email."* Instrument it. Everything else is vibes, and vibes will let a dead project limp for six months.

**5.6 — §16 Security is a wish list.** "Memory should be encrypted where practical" and "backups should exist" are not requirements. Given multi-client data, the real threats are (a) prompt injection via ingested content — you plan to ingest GitHub issues and email, both attacker-writable — and (b) cross-workspace leakage. Neither is mentioned. **Fix:** treat all ingested content as untrusted data, never as instructions; make workspace boundaries process boundaries (§2); test both. Note that GBrain's default behavior of unsupervised autonomous memory writes is directly relevant: an injected document can persist a false "fact" that poisons every later briefing. Gate writes.

**5.7 — Single point of failure: you.** Nothing in the doc addresses what happens after a two-week vacation, when 40 stale cron-written memories have accumulated and the briefing has quietly become wrong. Self-improving systems drift. **Fix:** the briefing shows its own staleness; memory writes are reviewable in git (they are, if markdown-first is preserved — another reason to protect it).

---

## 6. Assumptions I could not verify

Stated plainly, since they affect recommendations:

- Benchmark numbers for GBrain (P@5 49.1 / R@5 97.9) and Hermes (40% faster with 20+ skills) are **vendor-self-reported** and not independently replicated. Do not use them for selection.
- Star counts and adoption figures for both projects vary wildly across secondary sources (Hermes reported anywhere from 64k to 175k). I did not confirm from the repos — `WebFetch` was unavailable this session (model error), so all findings here come from search-result summaries rather than primary docs. **Verify the two repos directly before committing.**
- Both projects released in 2026 and are moving fast. Any API-level detail in the eval matrix may be stale within weeks.

---

## 7. What I'd change in one sentence each

1. Workspace becomes the deployment unit of the knowledge layer, not a layer above it.
2. Nine agents become two, with a written promotion criterion.
3. Roadmap resequences to test the briefing hypothesis in week one with no infrastructure.
4. Write the `TaskSpec` contract before any code — it is what makes "replaceable execution" true.
5. Phase 0 includes a spike; "no coding" costs more than it saves.
6. Cost and security get mechanisms, not principles.
7. Add kill criteria to phases 0 and 1.
