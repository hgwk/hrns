package hrns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func initDocsProposal() error {
	path := filepath.Join(".hrns", "doc-proposal.json")
	if Exists(path) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"version": 1,
		"proposals": []map[string]any{{
			"path": "", "title": "", "purpose": "Explain why this must be a new document instead of an update.",
			"summary": "Short summary of the planned content.", "decision": "new_document", "target": "",
		}},
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("created .hrns/doc-proposal.json")
	return nil
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
