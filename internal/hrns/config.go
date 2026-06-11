package hrns

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	AuditSets         AuditSets         `json:"auditSets"`
	LineAudit         LineAuditConfig   `json:"lineAudit"`
	Docs              DocsConfig        `json:"docs"`
	Fixtures          FixturesConfig    `json:"fixtures"`
	Env               EnvConfig         `json:"env"`
	AgentInstructions AgentConfig       `json:"agentInstructions"`
	DocsDuplication   DupConfig         `json:"docsDuplication"`
	DocsProposal      ProposalConfig    `json:"docsProposal"`
	JSONDuplicateKeys JSONKeysConfig    `json:"jsonDuplicateKeys"`
	ForbiddenRefs     ForbiddenConfig   `json:"forbiddenReferences"`
	MagicNumbers      MagicConfig       `json:"magicNumbers"`
	StructureRatchet  RatchetConfig     `json:"structureRatchet"`
	PlaceholderRoutes PlaceholderConfig `json:"placeholderRoutes"`
	ScopeDrift        ScopeDriftConfig  `json:"scopeDrift"`
	Abstractions      AbstractionConfig `json:"speculativeAbstractions"`
	Regression        RegressionConfig  `json:"regressionEvidence"`
	MainDiff          MainDiffConfig    `json:"mainDiff"`
	StopRule          StopRuleConfig    `json:"stopRule"`
	Elegance          EleganceConfig    `json:"elegance"`
}

type AuditSets struct {
	Default []string `json:"default"`
	All     []string `json:"all"`
}
type LineAuditConfig struct {
	MaxLines   int      `json:"maxLines"`
	Roots      []string `json:"roots"`
	Extensions []string `json:"extensions"`
}
type DocsConfig struct {
	Roots []string `json:"roots"`
}
type FixturesConfig struct {
	Roots     []string `json:"roots"`
	TestRoots []string `json:"testRoots"`
}
type EnvConfig struct {
	Example          string   `json:"example"`
	Roots            []string `json:"roots"`
	RequiredPrefixes []string `json:"requiredPrefixes"`
	Ignored          []string `json:"ignored"`
}
type AgentConfig struct {
	Mode             string   `json:"mode"`
	Files            []string `json:"files"`
	MinRepeatedWords int      `json:"minRepeatedWords"`
}
type DupConfig struct {
	Mode      string   `json:"mode"`
	Roots     []string `json:"roots"`
	Threshold float64  `json:"threshold"`
	MinTokens int      `json:"minTokens"`
}
type ProposalConfig struct {
	Mode              string   `json:"mode"`
	Roots             []string `json:"roots"`
	ProposalPath      string   `json:"proposalPath"`
	IndexPath         string   `json:"indexPath"`
	Threshold         float64  `json:"threshold"`
	TitleThreshold    float64  `json:"titleThreshold"`
	MinProposalTokens int      `json:"minProposalTokens"`
}
type JSONKeysConfig struct {
	Mode  string   `json:"mode"`
	Roots []string `json:"roots"`
}
type ForbiddenRule struct {
	Pattern string `json:"pattern"`
	Message string `json:"message"`
}
type ForbiddenConfig struct {
	Mode       string          `json:"mode"`
	Roots      []string        `json:"roots"`
	Rules      []ForbiddenRule `json:"rules"`
	AllowPaths []string        `json:"allowPaths"`
}
type MagicConfig struct {
	Mode          string   `json:"mode"`
	Roots         []string `json:"roots"`
	AllowPaths    []string `json:"allowPaths"`
	AllowedValues []string `json:"allowedValues"`
}
type RatchetMetric struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Max     int    `json:"max"`
}
type RatchetFile struct {
	Path     string          `json:"path"`
	MaxLines int             `json:"maxLines"`
	Metrics  []RatchetMetric `json:"metrics"`
}
type RatchetConfig struct {
	Mode  string        `json:"mode"`
	Files []RatchetFile `json:"files"`
}
type PlaceholderConfig struct {
	Mode  string   `json:"mode"`
	Roots []string `json:"roots"`
}
type ScopeDriftConfig struct {
	Mode string `json:"mode"`
	Base string `json:"base"`
}
type AbstractionConfig struct {
	Mode               string   `json:"mode"`
	Base               string   `json:"base"`
	Terms              []string `json:"terms"`
	SingleUseThreshold int      `json:"singleUseThreshold"`
}
type RegressionConfig struct {
	Mode        string   `json:"mode"`
	Base        string   `json:"base"`
	BugKeywords []string `json:"bugKeywords"`
	TestPaths   []string `json:"testPaths"`
}
type MainDiffConfig struct {
	Mode            string   `json:"mode"`
	Base            string   `json:"base"`
	MaxFiles        int      `json:"maxFiles"`
	MaxChangedLines int      `json:"maxChangedLines"`
	RiskyPatterns   []string `json:"riskyPatterns"`
}
type StopRuleConfig struct {
	Mode                     string   `json:"mode"`
	LogPaths                 []string `json:"logPaths"`
	RepeatedFailureThreshold int      `json:"repeatedFailureThreshold"`
}
type EleganceConfig struct {
	Mode                   string   `json:"mode"`
	Base                   string   `json:"base"`
	MaxNewFiles            int      `json:"maxNewFiles"`
	MaxLargeFiles          int      `json:"maxLargeFiles"`
	LargeFileLineThreshold int      `json:"largeFileLineThreshold"`
	SmellPatterns          []string `json:"smellPatterns"`
}

func DefaultConfig() Config {
	return Config{
		AuditSets: AuditSets{Default: stableAudits, All: allAudits},
		LineAudit: LineAuditConfig{MaxLines: 300, Roots: []string{"cmd", "internal", "scripts", "packages", "infra"},
			Extensions: []string{".go", ".ts", ".tsx", ".mjs", ".js", ".rs", ".sql"}},
		Docs:     DocsConfig{Roots: []string{"docs", "README.md", "AGENTS.md", "CLAUDE.md"}},
		Fixtures: FixturesConfig{Roots: []string{"packages"}, TestRoots: []string{"packages"}},
		Env: EnvConfig{Example: ".env.example", Roots: []string{"packages", "scripts", "infra"},
			RequiredPrefixes: []string{}, Ignored: []string{"PATH", "NODE_ENV", "CI", "LINE_AUDIT_MAX"}},
		AgentInstructions: AgentConfig{Mode: "warn", Files: []string{"AGENTS.md", "CLAUDE.md"}, MinRepeatedWords: 24},
		DocsDuplication:   DupConfig{Mode: "warn", Roots: []string{"docs", "README.md"}, Threshold: 0.72, MinTokens: 80},
		DocsProposal: ProposalConfig{Mode: "fail", Roots: []string{"docs", "README.md"}, ProposalPath: ".hrns/doc-proposal.json",
			IndexPath: ".hrns/docs-index.json", Threshold: 0.52, TitleThreshold: 0.45, MinProposalTokens: 12},
		JSONDuplicateKeys: JSONKeysConfig{Mode: "fail", Roots: []string{"package.json", "tsconfig.json", ".github", "messages"}},
		ForbiddenRefs: ForbiddenConfig{Mode: "warn", Roots: []string{"cmd", "internal", "packages", "apps", "src", "docs"},
			Rules: []ForbiddenRule{}, AllowPaths: []string{}},
		MagicNumbers: MagicConfig{Mode: "warn", Roots: []string{"cmd", "internal", "packages", "apps", "src"},
			AllowPaths: []string{}, AllowedValues: []string{"0", "1", "2", "10", "100", "127", "255", "300", "404", "500", "1000", "1024"}},
		StructureRatchet:  RatchetConfig{Mode: "fail", Files: []RatchetFile{}},
		PlaceholderRoutes: PlaceholderConfig{Mode: "warn", Roots: []string{"apps", "src", "pages", "app"}},
		ScopeDrift:        ScopeDriftConfig{Mode: "warn", Base: "main"},
		Abstractions: AbstractionConfig{Mode: "warn", Base: "main", SingleUseThreshold: 1,
			Terms: []string{"Manager", "Factory", "Strategy", "Provider", "Registry", "Adapter", "Config"}},
		Regression: RegressionConfig{Mode: "warn", Base: "main",
			BugKeywords: []string{"fix", "bug", "regression", "crash", "broken", "error", "fail"},
			TestPaths:   []string{"test", "tests", "__tests__", ".test.", ".spec."}},
		MainDiff: MainDiffConfig{Mode: "warn", Base: "main", MaxFiles: 40, MaxChangedLines: 1200,
			RiskyPatterns: []string{"package-lock.json", "pnpm-lock.yaml", "^dist/", "^build/", "^coverage/"}},
		StopRule: StopRuleConfig{Mode: "warn", LogPaths: []string{".hrns/failures.log"}, RepeatedFailureThreshold: 2},
		Elegance: EleganceConfig{Mode: "warn", Base: "main", MaxNewFiles: 20, MaxLargeFiles: 4,
			LargeFileLineThreshold: 250, SmellPatterns: []string{"TODO", "FIXME", "temporary", "workaround", "hack"}},
	}
}

func LoadConfig(root string) (Config, error) {
	cfg := DefaultConfig()
	_ = mergeConfigFile(&cfg, filepath.Join(root, "package.json"), "hrns")
	path := os.Getenv("HRNS_CONFIG")
	if path == "" {
		path = filepath.Join(root, "hrns.config.json")
	}
	_ = mergeConfigFile(&cfg, path, "")
	return cfg, nil
}

func mergeConfigFile(cfg *Config, path, nested string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if nested != "" {
		n, ok := raw[nested].(map[string]any)
		if !ok {
			return nil
		}
		raw = n
	}
	base, _ := json.Marshal(cfg)
	var merged map[string]any
	_ = json.Unmarshal(base, &merged)
	deepMerge(merged, raw)
	out, _ := json.Marshal(merged)
	return json.Unmarshal(out, cfg)
}

func deepMerge(dst, src map[string]any) {
	for k, v := range src {
		if sm, ok := v.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = v
	}
}
