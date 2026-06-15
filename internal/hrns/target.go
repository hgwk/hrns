package hrns

import (
	"fmt"
	"os"
	"strings"
)

func parseTargetArg(args []string) (string, []string, error) {
	target := "."
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--target" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("usage: --target PATH")
			}
			target = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimPrefix(arg, "--target=")
			if target == "" {
				return "", nil, fmt.Errorf("usage: --target PATH")
			}
			continue
		}
		out = append(out, arg)
	}
	return target, out, nil
}

func chdirTarget(target string) (func(), error) {
	if target == "" {
		target = "."
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(target); err != nil {
		return nil, fmt.Errorf("target %s: %w", target, err)
	}
	return func() {
		_ = os.Chdir(wd)
	}, nil
}
