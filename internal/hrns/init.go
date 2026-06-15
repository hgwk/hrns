package hrns

import (
	"encoding/json"
	"fmt"
	"os"
)

func initCommand(args []string) error {
	profile, err := parseInitProfile(args)
	if err != nil {
		return err
	}
	if err := initConfig(profile); err != nil {
		return err
	}
	if contains(args, "--docs") {
		if err := initDocsProposal(); err != nil {
			return err
		}
	}
	if contains(args, "--instructions") {
		if err := initInstructions(); err != nil {
			return err
		}
	}
	return nil
}

func initConfig(profile string) error {
	target := "hrns.config.json"
	if Exists(target) {
		if profile != "" {
			fmt.Printf("hrns.config.json already exists; --profile %s was not applied\n", profile)
		} else {
			fmt.Println("hrns.config.json already exists")
		}
		return nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg := ConfigForProject(wd)
	applyInitProfile(&cfg, profile)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(target, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("created hrns.config.json")
	return nil
}
