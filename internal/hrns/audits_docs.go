package hrns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func RunAgentInstructionDrift(cfg Config) error {
	a := NewAudit("verify-agent-instruction-drift")
	min := cfg.AgentInstructions.MinRepeatedWords
	if min == 0 {
		min = 24
	}
	type doc struct {
		file  string
		words []string
	}
	var docs []doc
	for _, file := range cfg.AgentInstructions.Files {
		if Exists(file) {
			docs = append(docs, doc{file, words(ReadText(file))})
		}
	}
	var findings []Finding
	for i := 0; i < len(docs); i++ {
		for j := i + 1; j < len(docs); j++ {
			if sameFile(docs[i].file, docs[j].file) {
				continue
			}
			overlap := longestWordRun(docs[i].words, docs[j].words)
			if overlap >= min {
				findings = append(findings, Finding{"agent instruction files contain duplicated prose",
					fmt.Sprintf("%s <-> %s: %d word run", docs[i].file, docs[j].file, overlap)})
			}
		}
	}
	return finishByMode(a, findings, cfg.AgentInstructions.Mode)
}

func sameFile(left, right string) bool {
	li, lerr := os.Stat(left)
	ri, rerr := os.Stat(right)
	return lerr == nil && rerr == nil && os.SameFile(li, ri)
}

func RunDocsDuplication(cfg Config) error {
	a := NewAudit("verify-docs-duplication")
	var docs []struct {
		file   string
		tokens map[string]struct{}
	}
	for _, file := range ListFiles(cfg.DocsDuplication.Roots, nil) {
		if strings.HasSuffix(file, ".md") {
			tokens := tokenSet(ReadText(file))
			if len(tokens) >= cfg.DocsDuplication.MinTokens {
				docs = append(docs, struct {
					file   string
					tokens map[string]struct{}
				}{file, tokens})
			}
		}
	}
	var findings []Finding
	for i := 0; i < len(docs); i++ {
		for j := i + 1; j < len(docs); j++ {
			score := jaccard(docs[i].tokens, docs[j].tokens)
			if score >= cfg.DocsDuplication.Threshold {
				findings = append(findings, Finding{"documents look duplicative",
					fmt.Sprintf("%s <-> %s: %.2f", docs[i].file, docs[j].file, score)})
			}
		}
	}
	return finishByMode(a, findings, cfg.DocsDuplication.Mode)
}

func RunDocProposal(cfg Config) error {
	a := NewAudit("verify-doc-proposal")
	path := os.Getenv("HRNS_DOC_PROPOSAL")
	if path == "" {
		path = cfg.DocsProposal.ProposalPath
	}
	if !Exists(path) {
		fmt.Printf("verify-doc-proposal: PASS (%s not present)\n", path)
		return nil
	}
	var raw map[string]any
	if err := ReadJSON(path, &raw); err != nil {
		return err
	}
	proposals := proposalList(raw)
	type doc struct {
		path, title string
		tokens      map[string]struct{}
		titleTokens map[string]struct{}
	}
	var docs []doc
	for _, file := range ListFiles(cfg.DocsProposal.Roots, nil) {
		if strings.HasSuffix(file, ".md") {
			text := ReadText(file)
			title := firstHeading(text)
			docs = append(docs, doc{file, title, tokenSet(text), tokenSet(title)})
		}
	}
	var findings []Finding
	for i, p := range proposals {
		label := stringValue(p["path"])
		if label == "" {
			label = fmt.Sprintf("proposals[%d]", i)
			findings = append(findings, Finding{"doc proposal missing path", label})
		}
		if stringValue(p["purpose"]) == "" {
			findings = append(findings, Finding{"doc proposal missing purpose", label})
		}
		text := strings.Join([]string{stringValue(p["title"]), stringValue(p["purpose"]), stringValue(p["summary"]), stringValue(p["content"])}, "\n")
		ptokens := tokenSet(text)
		if len(ptokens) < cfg.DocsProposal.MinProposalTokens {
			findings = append(findings, Finding{"doc proposal is too thin to review for duplication", label})
			continue
		}
		titleTokens := tokenSet(stringValue(p["title"]))
		bestDoc := ""
		bestBody, bestTitle := 0.0, 0.0
		for _, doc := range docs {
			if doc.path == stringValue(p["path"]) {
				continue
			}
			body, title := jaccard(ptokens, doc.tokens), jaccard(titleTokens, doc.titleTokens)
			if body >= cfg.DocsProposal.Threshold || title >= cfg.DocsProposal.TitleThreshold {
				if body+title > bestBody+bestTitle {
					bestDoc, bestBody, bestTitle = doc.path, body, title
				}
			}
		}
		if bestDoc != "" && !(stringValue(p["decision"]) == "update_existing" && stringValue(p["target"]) == bestDoc) {
			findings = append(findings, Finding{"new doc proposal overlaps an existing document; update existing doc instead",
				fmt.Sprintf("%s -> %s (body %.2f, title %.2f)", label, bestDoc, bestBody, bestTitle)})
		}
	}
	return finishByMode(a, findings, cfg.DocsProposal.Mode)
}

func WriteDocsIndex(cfg Config) error {
	type entry struct {
		Path       string   `json:"path"`
		Title      string   `json:"title"`
		Headings   []string `json:"headings"`
		TokenCount int      `json:"tokenCount"`
		Tokens     []string `json:"tokens"`
	}
	var docs []entry
	for _, file := range ListFiles(cfg.DocsProposal.Roots, nil) {
		if !strings.HasSuffix(file, ".md") {
			continue
		}
		text := ReadText(file)
		tokens := keys(tokenSet(text))
		sort.Strings(tokens)
		docs = append(docs, entry{file, firstHeading(text), headings(text, 20), len(tokens), tokens})
	}
	payload := map[string]any{"version": 1, "generatedAt": nowISO(), "roots": cfg.DocsProposal.Roots, "documents": docs}
	out, _ := json.MarshalIndent(payload, "", "  ")
	target := cfg.DocsProposal.IndexPath
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, append(out, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("docs-index: wrote %s (%d document(s))\n", target, len(docs))
	return nil
}

func tokenSet(text string) map[string]struct{} {
	stop := map[string]struct{}{"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {}, "으로": {}, "에서": {}, "하는": {}, "그리고": {}, "또는": {}}
	out := map[string]struct{}{}
	for _, token := range regexp.MustCompile(`[a-z0-9가-힣_:-]{3,}`).FindAllString(strings.ToLower(text), -1) {
		if _, skip := stop[token]; !skip {
			out[token] = struct{}{}
		}
	}
	return out
}

func jaccard(left, right map[string]struct{}) float64 {
	intersection := 0
	for token := range left {
		if _, ok := right[token]; ok {
			intersection++
		}
	}
	return float64(intersection) / float64(max(1, len(left)+len(right)-intersection))
}
