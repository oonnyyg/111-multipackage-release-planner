package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

// Command release-planner reads a batch of packages and commits from JSON
// and prints an offline release plan. It never modifies the repository or
// calls any release service.
func main() {
	inPath := flag.String("in", "", "input JSON file (default: stdin)")
	format := flag.String("format", "json", "output format: json, markdown, or all")
	flag.Parse()
	if err := run(*inPath, *format, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(inPath, format string, stdout io.Writer) error {
	var r io.Reader = os.Stdin
	if inPath != "" {
		f, err := os.Open(inPath)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	in, err := ReadInput(r)
	if err != nil {
		return err
	}
	plan, err := BuildPlan(in)
	if err != nil {
		return err
	}
	switch format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	case "markdown":
		_, err := fmt.Fprint(stdout, RenderMarkdown(plan))
		return err
	case "all":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(plan); err != nil {
			return err
		}
		_, err := fmt.Fprint(stdout, RenderMarkdown(plan))
		return err
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}
