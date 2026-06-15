package hrns

import (
	"encoding/json"
	"fmt"
)

func printList(cfg Config, verbose bool) {
	fmt.Println("Stable audits:")
	for _, name := range stableAudits {
		printAuditListItem(name, cfg, verbose)
	}
	fmt.Println("\nConfigurable audits:")
	for _, name := range allAudits[len(stableAudits):] {
		printAuditListItem(name, cfg, verbose)
	}
	fmt.Println("\nConfigured default audit set:")
	audits := cfg.AuditSets.Default
	if len(audits) == 0 {
		audits = stableAudits
	}
	for _, name := range audits {
		normalized := normalizeAuditName(name)
		printAuditListItem(normalized, cfg, verbose)
	}
}

type listAuditItem struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Config   string `json:"config,omitempty"`
	Failure  string `json:"failure,omitempty"`
}

type listJSONPayload struct {
	SchemaVersion     int             `json:"schema_version"`
	StableCount       int             `json:"stable_count"`
	ConfigurableCount int             `json:"configurable_count"`
	DefaultCount      int             `json:"default_count"`
	Stable            []listAuditItem `json:"stable"`
	Configurable      []listAuditItem `json:"configurable"`
	Default           []listAuditItem `json:"default"`
}

func printListJSON(cfg Config) error {
	payload := listJSONPayload{
		SchemaVersion: 1,
		Stable:        auditItems(stableAudits, "stable", cfg),
		Configurable:  auditItems(allAudits[len(stableAudits):], "configurable", cfg),
	}
	audits := cfg.AuditSets.Default
	if len(audits) == 0 {
		audits = stableAudits
	}
	payload.Default = auditItems(audits, "default", cfg)
	payload.StableCount = len(payload.Stable)
	payload.ConfigurableCount = len(payload.Configurable)
	payload.DefaultCount = len(payload.Default)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func auditItems(names []string, category string, cfg Config) []listAuditItem {
	items := make([]listAuditItem, 0, len(names))
	for _, name := range names {
		normalized := normalizeAuditName(name)
		item := listAuditItem{Name: normalized, Category: category, Status: auditStatus(normalized, cfg)}
		if info, ok := auditCatalog[normalized]; ok {
			item.Config = info.Config
			item.Failure = info.Failure
		}
		items = append(items, item)
	}
	return items
}

func printAuditListItem(name string, cfg Config, verbose bool) {
	status := auditStatus(name, cfg)
	fmt.Printf("- %s [%s]\n", name, status)
	if !verbose {
		return
	}
	if info, ok := auditCatalog[normalizeAuditName(name)]; ok {
		fmt.Printf("  config: %s\n", info.Config)
		fmt.Printf("  failure: %s\n", info.Failure)
		if status == "needs config" {
			fmt.Printf("  next: configure %s or remove this audit from the active set\n", info.Config)
		}
	}
}
