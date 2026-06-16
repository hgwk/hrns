package hrns

import (
	"fmt"
	"os"
	"path/filepath"
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

func parseHomeArg(args []string) (string, []string, error) {
	home := ""
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--home" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("usage: --home PATH")
			}
			home = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--home=") {
			home = strings.TrimPrefix(arg, "--home=")
			if home == "" {
				return "", nil, fmt.Errorf("usage: --home PATH")
			}
			continue
		}
		out = append(out, arg)
	}
	return home, out, nil
}

func setHomeOverride(home string) (func(), error) {
	if home == "" {
		return func() {}, nil
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("home %s: %w", home, err)
	}
	old, hadOld := os.LookupEnv("HRNS_HOME")
	if err := os.Setenv("HRNS_HOME", abs); err != nil {
		return nil, err
	}
	return func() {
		if hadOld {
			_ = os.Setenv("HRNS_HOME", old)
		} else {
			_ = os.Unsetenv("HRNS_HOME")
		}
	}, nil
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
