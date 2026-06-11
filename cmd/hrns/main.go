package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hgwk/hrns/internal/hrns"
)

func main() {
	if target := os.Getenv("HRNS_TARGET_ROOT"); target != "" {
		if err := os.Chdir(target); err != nil {
			fmt.Fprintln(os.Stderr, "hrns:", err)
			os.Exit(1)
		}
	}
	args := os.Args[1:]
	switch filepath.Base(os.Args[0]) {
	case "hrns-audit":
		args = append([]string{"audit"}, args...)
	case "hrns-line-audit":
		args = append([]string{"line-audit"}, args...)
	}
	if err := hrns.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, "hrns:", err)
		os.Exit(1)
	}
}
