package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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

func runBrief(args []string) error {
	// TODO(phase 0): same shape as runAsk but with a fixed briefing prompt,
	// write output to brains/shared/briefings/YYYY-MM-DD.md as well as stdout.
	// Deferred until `ask` is validated as useful — see docs/02-technical-design.md §8.
	return fmt.Errorf("not implemented yet — use 'cos ask' for now")
}
