package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/multipackage-release-planner/planner"
)

func main() {
	markdownPath := flag.String("markdown", "", "optional path to write the Markdown changelog")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: release-planner [-markdown out.md] <input.json>")
		os.Exit(2)
	}
	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	input, err := planner.DecodeInput(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	plan, err := planner.BuildPlan(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
	if *markdownPath != "" {
		if err := os.WriteFile(*markdownPath, []byte(planner.RenderMarkdown(plan)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}
