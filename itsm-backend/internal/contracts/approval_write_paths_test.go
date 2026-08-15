package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type approvalWriteManifest struct {
	Version       int                  `json:"version"`
	TargetRuntime string               `json:"targetRuntime"`
	Entries       []approvalWriteEntry `json:"entries"`
}

type approvalWriteEntry struct {
	File   string   `json:"file"`
	Owner  string   `json:"owner"`
	Models []string `json:"models"`
	Mode   string   `json:"mode"`
}

var approvalWritePattern = regexp.MustCompile(
	`\.(ApprovalChain|ApprovalWorkflow|ServiceRequestApproval|ProcessApprovalDecision|ChangeApproval)\.(Create|Update|Delete)|INSERT\s+INTO\s+(change_approvals|change_approval_chains)`,
)

func TestApprovalWritePathInventoryHasNoDrift(t *testing.T) {
	backendRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(backendRoot, "internal", "contracts", "approval_write_paths.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest approvalWriteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version < 1 || manifest.TargetRuntime != "bpmn" {
		t.Fatalf("invalid approval contract metadata: version=%d target=%q", manifest.Version, manifest.TargetRuntime)
	}

	registered := make(map[string]approvalWriteEntry, len(manifest.Entries))
	allowedModes := map[string]bool{
		"authoritative": true, "template-only": true, "legacy-runtime": true, "compatibility": true,
	}
	for _, entry := range manifest.Entries {
		if entry.File == "" || entry.Owner == "" || len(entry.Models) == 0 || !allowedModes[entry.Mode] {
			t.Fatalf("incomplete approval writer entry: %+v", entry)
		}
		if _, duplicate := registered[entry.File]; duplicate {
			t.Fatalf("duplicate approval writer entry: %s", entry.File)
		}
		if _, err := os.Stat(filepath.Join(backendRoot, entry.File)); err != nil {
			t.Fatalf("registered approval writer %s: %v", entry.File, err)
		}
		registered[entry.File] = entry
	}

	discovered := make(map[string]bool)
	err = filepath.WalkDir(backendRoot, func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(backendRoot, path)
		if err != nil {
			return err
		}
		if item.IsDir() {
			if rel == "ent" || rel == "migrations" || rel == "migration" || strings.HasPrefix(rel, "ent"+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if approvalWritePattern.Match(content) {
			discovered[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var unregistered []string
	for file := range discovered {
		if _, ok := registered[file]; !ok {
			unregistered = append(unregistered, file)
		}
	}
	sort.Strings(unregistered)
	if len(unregistered) > 0 {
		t.Fatalf("unregistered approval runtime writers: %s", strings.Join(unregistered, ", "))
	}
}
