package seeder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"

	"itsm-backend/internal/authz"
	"itsm-backend/service"
)

// componentChecksum hashes the JSON seed manifest together with the in-code
// definitions that produce this component's records.
func componentChecksum(payload []byte, component string) (string, error) {
	inputs, err := manifestDigestInputs(component)
	if err != nil {
		return "", err
	}
	return hashComponentInputs(payload, component, inputs), nil
}

func hashComponentInputs(payload []byte, component string, inputs []string) string {
	digest := sha256.New()
	digest.Write(payload)
	for _, input := range inputs {
		digest.Write([]byte{0})
		digest.Write([]byte(input))
	}
	digest.Write([]byte(":" + component))
	return hex.EncodeToString(digest.Sum(nil))
}

// manifestDigestInputs lists the in-code definitions that shape a component's
// records. They are invisible to the JSON seed config, so without them the
// version ledger could not notice a menu, permission or template edit.
func manifestDigestInputs(component string) ([]string, error) {
	switch component {
	case "identity-rbac":
		inputs := make([]string, 0, 256)
		for _, code := range AllDefinedPermissionCodes() {
			inputs = append(inputs, "permission="+code)
		}
		for _, spec := range menuDefinitions() {
			inputs = append(inputs, fmt.Sprintf("menu=%s|%s|%s|%s|%s|%d|%s",
				spec.Path, spec.Name, spec.ParentPath, spec.Icon, spec.PermissionCode, spec.SortOrder, spec.Description))
		}
		for _, role := range BuiltinRoles() {
			inputs = append(inputs, fmt.Sprintf("role=%s|%s|%s", role.Code, role.Name, role.Description))
		}
		for _, group := range BuiltinGroups() {
			inputs = append(inputs, "group="+group.Name)
		}
		inputs = append(inputs, builtinGrantDigests(authz.BuiltinRolePermissionCodes())...)
		return inputs, nil

	case "itil-core":
		inputs := make([]string, 0, len(ticketTypeDefinitions()))
		for _, tt := range ticketTypeDefinitions() {
			inputs = append(inputs, fmt.Sprintf("ticket-type=%s|%s|%s|%s|%s",
				tt.Code, tt.Name, tt.Description, tt.Icon, tt.Color))
		}
		return inputs, nil

	case "workflow-core":
		inputs := make([]string, 0, 16)
		for _, tpl := range workflowTemplateDefinitions() {
			inputs = append(inputs, fmt.Sprintf("workflow-template=%s|%s|%s|%s|%s",
				tpl.key, tpl.name, tpl.desc, tpl.domain, tpl.bpmnFile))
		}
		embedded, err := embedFSManifestDigest(seedWorkflowTemplateFS, "templates")
		if err != nil {
			return nil, fmt.Errorf("hash embedded workflow templates: %w", err)
		}
		inputs = append(inputs, "seed-bpmn-digest="+embedded)
		deployed, err := service.EmbeddedBPMNDigest()
		if err != nil {
			return nil, fmt.Errorf("hash deployed BPMN templates: %w", err)
		}
		inputs = append(inputs, "deployed-bpmn-digest="+deployed)
		return inputs, nil

	case "sla-core":
		inputs := make([]string, 0, len(defaultSLAPolicySeeds()))
		for _, policy := range defaultSLAPolicySeeds() {
			encoded := fmt.Sprintf("%+v", policy)
			inputs = append(inputs, "sla-policy-default="+encoded)
		}
		return inputs, nil

	case "cmdb-core", "extension-core":
		// Both components write exclusively from the JSON seed manifest.
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown component %q", component)
	}
}

func builtinGrantDigests(grants map[string][]string) []string {
	roles := make([]string, 0, len(grants))
	for code := range grants {
		roles = append(roles, code)
	}
	sort.Strings(roles)
	inputs := make([]string, 0, len(roles))
	for _, role := range roles {
		codes := append([]string(nil), grants[role]...)
		sort.Strings(codes)
		for _, code := range codes {
			inputs = append(inputs, "grant="+role+"|"+code)
		}
	}
	return inputs
}

func embedFSManifestDigest(fsys fs.FS, dir string) (string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", err
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
		content, err := fs.ReadFile(fsys, dir+"/"+name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(digest, "%s\x00%d\x00", name, len(content))
		digest.Write(content)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
