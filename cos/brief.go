package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// topPagesForContext caps how many full pages we fetch and inline — this is
// the token-cost lever from docs/02-technical-design.md §3.4 (context_refs,
// not inlined content). 5 is generous for Phase 0's small corpus; revisit
// once real workspace brains are much larger.
const topPagesForContext = 5

func gbrainClientFromEnv() (*gbrainClient, error) {
	url := os.Getenv("GBRAIN_URL")
	if url == "" {
		url = "http://localhost:7333/mcp"
	}
	token := os.Getenv("GBRAIN_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GBRAIN_TOKEN not set (mint one with: gbrain auth create cos)")
	}
	return &gbrainClient{baseURL: url, token: token}, nil
}

// buildContext renders full page bodies as delimited, labeled blocks so the
// model can distinguish workspace knowledge from the question itself.
// Every block is untrusted data, never an instruction — see
// docs/02-technical-design.md §6 T1.
//
// Uses full compiled_truth, not query()'s chunk_text: chunk_text can cut off
// mid-section (verified 2026-08-02 — a chunk boundary landed right after an
// ADR's frontmatter, before its "Alternatives Considered" section, and the
// model fabricated plausible-sounding reasoning to fill the gap).
func buildContext(pages []*gbrainFullPage) string {
	var b strings.Builder
	for _, p := range pages {
		fmt.Fprintf(&b, "<doc slug=%q title=%q>\n%s\n</doc>\n\n", p.Slug, p.Title, p.CompiledTruth)
	}
	return b.String()
}

// parseProvider pulls "--provider X" out of args, defaulting to Ollama
// (free, local) unless overridden. Returns the remaining args.
func parseProvider(args []string) (provider string, rest []string) {
	for i, a := range args {
		if a == "--provider" && i+1 < len(args) {
			return args[i+1], append(append([]string{}, args[:i]...), args[i+2:]...)
		}
	}
	return "", args
}

func runAsk(args []string) error {
	provider, args := parseProvider(args)
	if len(args) == 0 {
		return fmt.Errorf("usage: cos ask [--provider ollama|anthropic] \"question\"")
	}
	question := strings.Join(args, " ")

	gb, err := gbrainClientFromEnv()
	if err != nil {
		return err
	}
	hits, err := gb.query(question)
	if err != nil {
		return fmt.Errorf("gbrain query: %w", err)
	}
	if len(hits) == 0 {
		fmt.Println("No relevant pages found in the brain.")
		return nil
	}
	if len(hits) > topPagesForContext {
		hits = hits[:topPagesForContext]
	}

	// De-dupe by slug — query() ranks chunks, so the same page can appear
	// more than once in the top results.
	seen := map[string]bool{}
	var fullPages []*gbrainFullPage
	for _, h := range hits {
		if seen[h.Slug] {
			continue
		}
		seen[h.Slug] = true
		page, err := gb.getPage(h.Slug)
		if err != nil {
			return fmt.Errorf("gbrain get_page %q: %w", h.Slug, err)
		}
		fullPages = append(fullPages, page)
	}

	synth, err := synthesizerFromEnv(provider)
	if err != nil {
		return err
	}

	answer, err := synth.synthesize(context.Background(), buildContext(fullPages), question)
	if err != nil {
		return err
	}
	fmt.Println(answer)
	return nil
}

// briefQuestion stands in for real multi-source aggregation (Phase 4+
// ingestion, per docs/03-gap-analysis.md G6) — Phase 0 has one brain fed by
// one imported corpus, so a single broad query is the whole retrieval step.
const briefQuestion = "What are the most important recent decisions, open questions, and risks recorded in this knowledge base?"

// briefProvider defaults to gemini, not ollama: three local models (1B/7B/
// 12B) all fabricated on a grounded-extraction test (docs/03-gap-analysis.md
// G7) and a briefing is exactly the unsupervised-trust case that failure
// mode is worst for. Override with --provider if you've run the eval
// harness (G7) against a specific model and trust it.
const briefProvider = "gemini"

// runBrief produces the Phase 0 kill-gate deliverable: a briefing citing
// real sources from the brain. Acceptance test is manual, not automated —
// docs/02-technical-design.md §8: "references >=3 source docs with
// citations" and "this told me something I hadn't already tracked."
func runBrief(args []string) error {
	provider, args := parseProvider(args)
	if provider == "" {
		provider = briefProvider
	}

	gb, err := gbrainClientFromEnv()
	if err != nil {
		return err
	}
	hits, err := gb.query(briefQuestion)
	if err != nil {
		return fmt.Errorf("gbrain query: %w", err)
	}
	if len(hits) == 0 {
		fmt.Println("No content in the brain yet — nothing to brief on.")
		return nil
	}
	if len(hits) > topPagesForContext {
		hits = hits[:topPagesForContext]
	}

	seen := map[string]bool{}
	var fullPages []*gbrainFullPage
	for _, h := range hits {
		if seen[h.Slug] {
			continue
		}
		seen[h.Slug] = true
		page, err := gb.getPage(h.Slug)
		if err != nil {
			return fmt.Errorf("gbrain get_page %q: %w", h.Slug, err)
		}
		fullPages = append(fullPages, page)
	}

	synth, err := synthesizerFromEnv(provider)
	if err != nil {
		return err
	}

	prompt := "Write a daily briefing for the operator of this knowledge base. " +
		"Structure it as: Recent Decisions, Open Questions, Risks. " +
		"Under each section, cite the doc slug for every claim. " +
		"If a section has nothing relevant in the docs, write \"Nothing new\" rather than inventing content."
	answer, err := synth.synthesize(context.Background(), buildContext(fullPages), prompt)
	if err != nil {
		return err
	}

	date := time.Now().Format("2006-01-02")
	out := fmt.Sprintf("# Briefing — %s\n\n%s\n", date, strings.TrimSpace(answer))
	fmt.Print(out)

	dir := "../brains/shared/briefings"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create briefings dir: %w", err)
	}
	path := filepath.Join(dir, date+".md")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write briefing: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\nwritten to %s\n", path)
	return nil
}
