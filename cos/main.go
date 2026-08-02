// cos is the Chief of Staff CLI: brief, ask, idea.
// Phase 0: talks to one GBrain instance, no workspace routing yet.
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Best-effort: .env lives at the repo root, cos runs from cos/. Missing
	// file is fine — real deployments set env vars directly.
	_ = godotenv.Load("../.env", ".env")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cos <brief|ask|idea> [args]")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "brief":
		err = runBrief(os.Args[2:])
	case "ask":
		err = runAsk(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
