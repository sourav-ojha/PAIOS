# Open-Source Evaluation Matrix

**Date:** 2026-07-30
**Confidence caveat, read first:** `WebFetch` failed this session (upstream model error), so every finding below comes from **web-search result summaries, not primary repository documentation**. Vendor benchmark claims are self-reported and not independently replicated. Both GBrain and Hermes Agent were released in 2026 and are moving fast — API-level details may be stale within weeks. **Verify the shortlisted two repos directly before committing.** Recommendations are robust to the uncertainty; specific numbers are not.

Weighting reflects this project's stated principles: knowledge durability and workspace isolation dominate; raw benchmark scores barely matter.

| Criterion | Weight | Why |
|---|---|---|
| Portability / no lock-in | 25% | `initial-plan.md` §5, §19.10 — the load-bearing principle |
| Workspace isolation | 20% | §7, §13 — multi-client is a legal requirement, not a feature |
| Operational burden (1 person) | 20% | Single operator; ops cost is the real constraint |
| Self-host completeness | 15% | §5 Docker-first, no SaaS dependency |
| Retrieval quality | 10% | Matters, but all credible options beat naive RAG |
| Ecosystem / longevity | 10% | Bus factor, contributor count |

---

## 1. Knowledge layer

| | **GBrain** | **Graphiti (Zep OSS)** | **Mem0** | **Letta** | **Cognee** | **Plain git + ripgrep + pgvector** |
|---|---|---|---|---|---|---|
| Model | Markdown files = truth, Postgres/PGLite index, self-wiring typed graph | Temporal knowledge graph, fact validity windows | Extracted facts in vector store | Agent runtime w/ editable core memory | Doc→graph ingestion pipeline | Files + BM25 + embeddings you own |
| Storage truth | **Markdown in git** | Graph DB | Vector DB | Agent state DB | Graph DB | **Files in git** |
| Portability | **Excellent** — index is derived, rebuildable | Poor — graph is the truth | Weak — facts are extracted artifacts | Poor — coupled to runtime | Weak | **Perfect** |
| Workspace isolation | **Weak** — scope/credential conventions in one brain; single-operator by design | Moderate — namespaces | Moderate | Weak | Weak | **Perfect** (separate dirs/DBs) |
| Temporal / contradiction handling | Weak | **Best in class** — models state change, not duplicate facts | Weak | N/A | Weak | None (you build it) |
| Ops burden | Low-moderate (PGLite easy, PG fleet moderate) | High — graph DB to operate | Low | Moderate | Moderate | **Lowest** |
| Self-host completeness | **Full, MIT** | Graphiti OSS; **full Zep platform is SaaS** | OSS = vector layer; **graph memory is paid** | Full | Full | N/A |
| LLM cost to index | **None** — graph wiring is pattern-matching, no LLM calls | LLM extraction per doc | LLM extraction | — | LLM extraction | None |
| Built-in extras | MCP server (~74 tools), hybrid search + RRF + reranker, overnight enrichment crons | — | — | Agent loop | — | None |
| Fit for *this* project | **Chosen** | Strong second / possible complement | Rejected | Rejected | Rejected | Fallback |

### Recommendation: GBrain, deployed per isolation tier

**Why it wins on this project's weights:** markdown-as-truth directly implements "knowledge is permanent, execution is temporary" — the index is disposable, the notes are yours, and portability is provable with a rebuild test. Zero-LLM graph wiring is a genuine cost advantage under a cost-first principle. The MCP server means Claude Code can read/write the brain with no integration work. And the domain model — people, companies, projects, decisions — matches a Chief of Staff use case rather than a chatbot one, because it was built for exactly that.

**What it loses on, stated plainly:**
- **Isolation is its weakest axis and your strongest requirement.** Single-operator by design; the "company brain" pattern is credential-scoped sources inside one brain, not tenant isolation. **Mitigation:** don't use its multi-source model for client separation — run one instance per isolated workspace (TDD §3.3). This is working *around* the tool, and it's the main risk in this choice.
- **Weak temporal/contradiction handling** (Gap G8). Mitigation: dated immutable ADRs with supersedes-links, a convention not a feature.
- **Default unsupervised memory writes** — must be gated (TDD §6 T4).
- **Young project** (2026), fast-moving, effectively one primary author. Bus factor is real; markdown-as-truth is what makes that survivable.

**Rejected alternatives, with reasons:**
- **Mem0** — the open-source tier gives you the vector layer; graph memory is behind a paid plan. Building the graph story on a paywalled feature violates vendor-independence.
- **Zep platform** — has stepped back from full open self-hosting. Graphiti alone is viable but is a library, not a brain: you'd build ingestion, search, and CLI yourself.
- **Letta** — it's an agent platform where memory is agent state. Adopting it means adopting its runtime, which contradicts "execution engine must be replaceable" and §4's "not another agent framework."
- **Cognee** — reasonable ingestion pipeline, but graph-as-truth loses portability, the top-weighted criterion.
- **Plain git + pgvector** — scores best on portability and ops and is the honest fallback if GBrain disappoints. Costs ~2 weeks to reach parity on hybrid search and entity wiring. **Keep as the documented exit path**; the portability test (TDD §3.3) is what keeps that exit open.

**Worth considering later:** Graphiti *alongside* GBrain, scoped to decisions/ADRs only, if G8 (stale conclusions in briefings) proves painful in practice. Do not do this in Phase 1.

---

## 2. Execution layer

| | **Claude Agent SDK** | **OpenHands** | **Hermes Agent (Nous)** | **Plain scripts + API** |
|---|---|---|---|---|
| What it is | Embeddable agent loop, runs on your infra | Full platform: SDK + CLI + GUI + Cloud | Self-hosted autonomous agent w/ persistent memory + self-written skills | Your own tool loop |
| Coding quality | **Strongest** (88.6% SWE-bench Verified reported) | Good; degrades on long multi-service refactors | Not a coding-agent benchmark leader | Depends on you |
| Model agnostic | Anthropic-centric | **Yes** — multi-provider | Yes | Yes |
| Sandboxing | Yes | **Yes, strong** (Docker-first) | Sandboxed exec + browser control | You build it |
| Subagent fan-out | **Native, strong** | Via ACP/agents | Profiles (multi-agent) | No |
| Memory model | Task-scoped (good — memory belongs to *our* layer) | Task-scoped | **Own persistent memory (SQLite/FTS5) — overlaps and conflicts with our knowledge layer** | None |
| Lock-in risk | Moderate (vendor SDK) | Low | Moderate (opinionated, owns memory) | None |
| Ops burden | **Low** | Moderate-high (Docker-in-Docker for CI) | Moderate-high, bleeding edge | Low |
| Fit | **Chosen, Phase 5** | **Best second adapter** | **Rejected as execution layer** | Phase 0–3 default |

### Recommendation: no execution layer until Phase 5; then Claude Agent SDK as first adapter

Phases 0–3 need *retrieval and synthesis*, not code execution. Adding an execution layer early is the fastest route to burning weeks on orchestration — exactly the "another agent framework" outcome §4 forbids. Direct API calls suffice.

At Phase 5, adopt **Claude Agent SDK** behind the `TaskSpec` contract (TDD §3.4): best coding quality, native subagents, lowest ops cost, and already the tool in use. **OpenHands is the designated second adapter** — model-agnostic, strong sandboxing, and its Agent Client Protocol support means it can host other agents, which makes it a good hedge rather than a competitor. The contract is what makes swapping cheap; write it first.

### Why Hermes Agent is rejected here — and where it *does* fit

Hermes is impressive and genuinely adjacent to this project's vision (self-hosted, persistent memory, self-written skills, overnight crons). But as *this* system's execution layer it's a structural mismatch:

1. **It owns memory.** Hermes has its own persistent store (SQLite + FTS5) plus tiered always-loaded context files. Our design puts memory in a per-workspace knowledge layer that is the durable asset. Two competing memory systems means two sources of truth and an unresolvable question about which one is authoritative.
2. **Tight always-on context budgets** (defaults on the order of ~2.2k chars memory / ~1.4k chars user) — restrictive relative to a briefing that must span many projects.
3. **Unsupervised autonomous writes by default**, including background self-improvement passes that mutate state without surfacing it — the same concern as GBrain's, but harder to gate because memory is core rather than peripheral.
4. **Documented cross-boundary reliability failures** in its multi-agent profiles system (per an arXiv analysis of silent delivery failures) — directly relevant since profiles would be the workspace-isolation mechanism.
5. **It's a peer product, not a component.** Hermes is closer to being *an alternative to this whole project* than a layer inside it.

That last point deserves a direct question: **if Hermes plus GBrain already does 60% of what you want out of the box, is building PAIOS the right call?** Notably, GBrain's own production deployment reportedly runs behind Tan's agents including Hermes — that pairing is a real, working stack. A legitimate answer is "yes, because workspace isolation and client separation are non-negotiable and neither tool provides them." But it should be an answer you give deliberately, and the cheapest way to earn it is to **run Hermes + GBrain for one week before Phase 1** and note exactly what's missing. That week is the highest-value experiment available and it might save you months.

---

## 3. Supporting components — adopt, don't build

| Need | Adopt | Don't build |
|---|---|---|
| Vector search | pgvector (in the Postgres you already run) | Any standalone vector DB |
| Postgres | `postgres:17` container | — |
| Scheduling | cron / systemd timers | A job orchestrator |
| Notifications | ntfy or Telegram bot | A notification service |
| Secrets | `.env` per workspace, later SOPS/age | A secret manager |
| Code search for agents | ripgrep | An index |
| Static analysis ("Security agent") | semgrep, gitleaks, trivy | A Security agent |
| Sandboxing | Docker | — |
| CLI | Python + Typer | A web UI (before Phase 3) |
| Auth | none needed (single operator) | **Especially don't build this** |

The plan's §17 Phase 2 lists "Authentication." For a single-operator, single-machine system, that is pure waste — SSH is your auth. Defer until there is a second human, which may be never.

---

## 4. Decision summary

| Decision | Choice | Confidence | Reversal cost |
|---|---|---|---|
| Knowledge layer | GBrain, per-tier instances | Medium-high | **Low** — markdown in git, portability test enforced |
| Database | Postgres 17 + pgvector from day one (not PGLite) | High | Low |
| Isolation mechanism | Container + database boundary | High | Moderate |
| Execution layer | None until Phase 5, then Claude Agent SDK | High | **Low** — `TaskSpec` contract |
| Second executor | OpenHands (adapter) | Medium | Low |
| Hermes Agent | Rejected as component; **evaluate as a whole-product alternative first** | Medium | — |
| Agent roster | 2 roles, written promotion criterion | High | Low |
| UI | CLI until Phase 3 | High | Low |

Every low-reversal-cost entry is low *because* of a specific mechanism (markdown truth, the TaskSpec contract, the portability test) rather than by luck. Those three mechanisms are what the design is actually buying.

---

## Sources

- [GBrain repository (garrytan/gbrain)](https://github.com/garrytan/gbrain) · [company-brain tutorial](https://github.com/garrytan/gbrain/blob/master/docs/tutorials/company-brain.md) · [gbrain.io security](https://gbrain.io/trust/security)
- [GBrain review — Vectorize](https://vectorize.io/articles/gbrain-review) · [MarkTechPost tutorial](https://www.marktechpost.com/2026/05/22/a-step-by-step-coding-tutorial-to-implement-gbrain-the-self-wiring-memory-layer-built-by-y-combinators-garry-tan-for-ai-agents/) · [PyShine](https://pyshine.com/gbrain-self-wiring-knowledge-graph-ai-agents/)
- [Hermes Agent docs (Nous Research)](https://hermes-agent.nousresearch.com/docs/) · [Persistent memory](https://hermes-agent.nousresearch.com/docs/user-guide/features/memory/) · [Memory providers](https://hermes-agent.nousresearch.com/docs/user-guide/features/memory-providers)
- [Hermes memory system analysis — Glukhov](https://www.glukhov.org/ai-systems/hermes/hermes-agent-memory-system/) · [awesome-hermes-agent](https://github.com/0xNyk/awesome-hermes-agent)
- [Channel Fracture: silent delivery failures in multi-agent systems (arXiv)](https://arxiv.org/pdf/2606.04896)
- [Agent memory frameworks compared — Atlan](https://atlan.com/know/best-ai-agent-memory-frameworks-2026/) · [Mem0 vs Zep vs Letta](https://rohitraj.tech/en/notes/open-source-ai-agent-memory-mem0-vs-zep-letta-2026) · [8 knowledge systems compared](https://fountaincity.tech/resources/blog/agent-memory-knowledge-systems-compared/)
- [OpenHands vs Claude Code](https://nerova.ai/comparisons/openhands-vs-claude-code-2026) · [OpenHands ACP / any coding agent](https://www.openhands.dev/blog/use-any-coding-agent-in-openhands-with-acp) · [Agent framework comparison — Composio](https://composio.dev/content/claude-agents-sdk-vs-openai-agents-sdk-vs-google-adk)
