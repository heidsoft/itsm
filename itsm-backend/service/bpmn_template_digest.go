package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
)

// EmbeddedBPMNDigest returns a deterministic digest over the embedded BPMN
// template files. Initialization component checksums include it so editing a
// template is visible to the version ledger without a hand-maintained bump.
func EmbeddedBPMNDigest() (string, error) {
	entries, err := fs.ReadDir(bpmnTemplates, "bpmn")
	if err != nil {
		return "", fmt.Errorf("read embedded bpmn templates: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	digest := sha256.New()
	for _, name := range names {
		content, err := bpmnTemplates.ReadFile("bpmn/" + name)
		if err != nil {
			return "", fmt.Errorf("read embedded bpmn template %s: %w", name, err)
		}
		fmt.Fprintf(digest, "%s\x00%d\x00", name, len(content))
		digest.Write(content)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
