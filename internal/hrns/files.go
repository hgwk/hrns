package hrns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var textExts = map[string]struct{}{
	"": {}, ".css": {}, ".cjs": {}, ".js": {}, ".json": {}, ".md": {}, ".mjs": {},
	".rs": {}, ".sql": {}, ".ts": {}, ".tsx": {}, ".yaml": {}, ".yml": {},
}

var excludedParts = map[string]struct{}{
	".git": {}, ".next": {}, "archived": {}, "build": {}, "coverage": {}, "dist": {},
	"node_modules": {}, "playwright-report": {}, "reference": {}, "target": {},
}

func Exists(rel string) bool {
	_, err := os.Stat(rel)
	return err == nil
}

func ReadText(rel string) string {
	data, _ := os.ReadFile(rel)
	return string(data)
}

func ReadJSON(rel string, out any) error {
	data, err := os.ReadFile(rel)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func ListFiles(roots []string, exts map[string]struct{}) []string {
	if exts == nil {
		exts = textExts
	}
	seen := map[string]struct{}{}
	for _, root := range roots {
		walk(root, exts, seen)
	}
	out := make([]string, 0, len(seen))
	for file := range seen {
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}

func walk(rel string, exts map[string]struct{}, out map[string]struct{}) {
	rel = normalizePath(rel)
	if shouldExclude(rel) {
		return
	}
	info, err := os.Stat(rel)
	if err != nil {
		return
	}
	if info.IsDir() {
		entries, err := os.ReadDir(rel)
		if err != nil {
			return
		}
		for _, entry := range entries {
			walk(filepath.Join(rel, entry.Name()), exts, out)
		}
		return
	}
	if !info.Mode().IsRegular() {
		return
	}
	if _, ok := exts[filepath.Ext(rel)]; ok {
		out[rel] = struct{}{}
	}
}

func shouldExclude(rel string) bool {
	for _, part := range strings.Split(normalizePath(rel), "/") {
		if _, ok := excludedParts[part]; ok {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func stringSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
