package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// goldenCase is one row of cos/eval/golden.json. expected_fact and
// must_not_claim are plain-English statements graded by an LLM judge
// (judgeSynthesize), not substring matches — a first pass at exact-phrase
// matching (docs/03-gap-analysis.md G7, live runs 2026-08-02/03) produced
// mostly false failures: models restate the same fact in different words
// (bolded terms, paraphrased mechanism names, different negation phrasing)
// far more often than they actually fabricate. Substring checks couldn't
// tell the difference; a judge model grading semantic content can.
type goldenCase struct {
	ID           string  `json:"id"`
	Question     string  `json:"question"`
	ExpectedFact string  `json:"expected_fact"`
	MustNotClaim string  `json:"must_not_claim,omitempty"`
	SourceSlug   *string `json:"source_slug"`
	Trap         bool    `json:"trap"`
	Notes        string  `json:"notes"`
}

func loadGoldenCases(path string) ([]goldenCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read golden set: %w", err)
	}
	var cases []goldenCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("parse golden set: %w", err)
	}
	return cases, nil
}

type evalResult struct {
	Case     goldenCase
	Answer   string
	Passed   bool
	Reason   string // judge's stated reason, always populated
	HardFail string // non-judge failure: gbrain/synth error
}

// judgePrompt asks a model to grade one answer against plain-English
// criteria and return strict JSON — kept separate from synthPrompt because
// the judge must not be told which provider produced the answer (avoid
// self-preference) and must return a parseable verdict, not prose.
func judgePrompt(question, answer, expectedFact, mustNotClaim string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are grading an AI-generated answer for factual correctness against a knowledge base. "+
		"Respond with ONLY a JSON object, no other text: {\"pass\": true|false, \"reason\": \"one sentence\"}.\n\n")
	fmt.Fprintf(&b, "Question asked: %s\n\n", question)
	fmt.Fprintf(&b, "Answer given: %s\n\n", answer)
	fmt.Fprintf(&b, "The answer PASSES only if it correctly conveys this fact (paraphrase, different wording, "+
		"and different citation style are all fine): %s\n", expectedFact)
	if mustNotClaim != "" {
		fmt.Fprintf(&b, "\nThe answer FAILS if it claims or implies this (even indirectly): %s\n", mustNotClaim)
	}
	fmt.Fprintf(&b, "\nGrade on substance only. Do not penalize phrasing, formatting, citation style, or which "+
		"source doc is cited, as long as the substantive fact is correct.")
	return b.String()
}

type judgeVerdict struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason"`
}

// judgeAnswer grades one case's answer using the same synthesizer type as
// the case itself. Not a separate "trusted" model — if the provider under
// test is broken enough to fabricate answers, it may also misjudge them.
// Treat judge failures as a signal to spot-check by hand, not as ground
// truth on their own (this harness has no independently-verified judge).
func judgeAnswer(synth synthesizer, question, answer, expectedFact, mustNotClaim string) (judgeVerdict, error) {
	raw, err := synth.synthesize(context.Background(), "", judgePrompt(question, answer, expectedFact, mustNotClaim))
	if err != nil {
		return judgeVerdict{}, err
	}
	// Models sometimes wrap JSON in a code fence despite instructions.
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var v judgeVerdict
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return judgeVerdict{}, fmt.Errorf("parse judge verdict %q: %w", truncate(raw, 200), err)
	}
	return v, nil
}

func runEvalCase(gb *gbrainClient, synth synthesizer, c goldenCase) evalResult {
	res := evalResult{Case: c}

	hits, err := gb.query(c.Question)
	if err != nil {
		res.HardFail = fmt.Sprintf("gbrain query: %v", err)
		return res
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
			res.HardFail = fmt.Sprintf("get_page %q: %v", h.Slug, err)
			return res
		}
		fullPages = append(fullPages, page)
	}

	answer, err := synth.synthesize(context.Background(), buildContext(fullPages), c.Question)
	if err != nil {
		res.HardFail = fmt.Sprintf("synthesize: %v", err)
		return res
	}
	res.Answer = answer

	verdict, err := judgeAnswer(synth, c.Question, answer, c.ExpectedFact, c.MustNotClaim)
	if err != nil {
		res.HardFail = fmt.Sprintf("judge: %v", err)
		return res
	}
	res.Passed = verdict.Pass
	res.Reason = verdict.Reason
	return res
}

// runEval is the G7 gate: run the golden set against a synthesizer before
// promoting it to any default. See docs/03-gap-analysis.md G7 and
// docs/02-technical-design.md §5/§5.1. Each case costs 2 synth calls
// (answer + judge) — factor that into rate-limit pacing and daily quota.
func runEval(args []string) error {
	provider, args := parseProvider(args)
	goldenPath := "eval/golden.json"
	if len(args) > 0 {
		goldenPath = args[0]
	}

	cases, err := loadGoldenCases(goldenPath)
	if err != nil {
		return err
	}

	gb, err := gbrainClientFromEnv()
	if err != nil {
		return err
	}
	synth, err := synthesizerFromEnv(provider)
	if err != nil {
		return err
	}

	// Gemini free tier is rate-limited per-minute (seen: 15 RPM on
	// flash-lite) and each case now makes 2 calls (answer + judge) — pace
	// accordingly rather than let quota errors masquerade as failures.
	pause := 8 * time.Second
	if provider == "ollama" {
		pause = 0
	}

	var passed, trapPassed, trapTotal, hardFails int
	for i, c := range cases {
		if i > 0 && pause > 0 {
			time.Sleep(pause)
		}
		res := runEvalCase(gb, synth, c)

		if c.Trap {
			trapTotal++
			if res.Passed {
				trapPassed++
			}
		}

		switch {
		case res.HardFail != "":
			hardFails++
			fmt.Printf("[ERROR] %s (trap=%v) — %s\n", c.ID, c.Trap, res.HardFail)
		case res.Passed:
			passed++
			fmt.Printf("[PASS] %s (trap=%v) — %s\n", c.ID, c.Trap, res.Reason)
		default:
			fmt.Printf("[FAIL] %s (trap=%v) — %s\n", c.ID, c.Trap, res.Reason)
			fmt.Printf("    answer: %s\n", truncate(res.Answer, 300))
		}
	}

	fmt.Printf("\n%d/%d passed, %d errored (%d/%d trap cases passed)\n",
		passed, len(cases), hardFails, trapPassed, trapTotal)
	if passed < len(cases) {
		return fmt.Errorf("eval failed: %d/%d cases did not pass (%d errored) — do not promote this provider/model to default",
			len(cases)-passed, len(cases), hardFails)
	}
	return nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
