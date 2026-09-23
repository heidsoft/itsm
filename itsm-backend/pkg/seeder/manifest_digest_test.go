package seeder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The version ledger must notice an in-code manifest edit: a menu, permission,
// ticket type or BPMN template change cannot leave the recorded checksum
// unchanged.
func TestComponentChecksumCoversInCodeManifests(t *testing.T) {
	payload, err := json.Marshal(getProductDefaultConfig())
	require.NoError(t, err)

	for _, component := range ProductionComponentNames {
		baseline, err := componentChecksum(payload, component)
		require.NoError(t, err, component)
		require.Len(t, baseline, 64, component)

		repeat, err := componentChecksum(payload, component)
		require.NoError(t, err)
		assert.Equal(t, baseline, repeat, "checksum must be deterministic")

		inputs, err := manifestDigestInputs(component)
		require.NoError(t, err)
		if len(inputs) == 0 {
			// cmdb-core and extension-core seed purely from the JSON manifest.
			assert.Equal(t, configOnlyChecksum(payload, component), baseline, component)
			continue
		}
		assert.NotEqual(t, configOnlyChecksum(payload, component), baseline,
			"%s checksum must include in-code manifest inputs", component)
	}

	_, err = manifestDigestInputs("not-a-component")
	require.ErrorContains(t, err, `unknown component "not-a-component"`)
}

func TestManifestDigestInputsCoverTheirComponents(t *testing.T) {
	identity, err := manifestDigestInputs("identity-rbac")
	require.NoError(t, err)
	require.NotEmpty(t, identity)
	assert.Contains(t, identity, "permission=ticket:read")
	assert.True(t, hasPrefixFrom(identity, "menu=/dashboard|"))
	assert.True(t, hasPrefixFrom(identity, "role="))
	assert.True(t, hasPrefixFrom(identity, "group=approvers-"))
	assert.True(t, hasPrefixFrom(identity, "grant=sysadmin|"))

	itil, err := manifestDigestInputs("itil-core")
	require.NoError(t, err)
	assert.Len(t, itil, len(ticketTypeDefinitions())+len(standardChangeDefinitions())+
		len(incidentCategoryDefinitions())+len(ticketTagDefinitions()))

	cmdb, err := manifestDigestInputs("cmdb-core")
	require.NoError(t, err)
	assert.Len(t, cmdb, len(ciTypeDefinitions()))

	workflow, err := manifestDigestInputs("workflow-core")
	require.NoError(t, err)
	assert.True(t, hasPrefixFrom(workflow, "seed-bpmn-digest="))
	assert.True(t, hasPrefixFrom(workflow, "deployed-bpmn-digest="))

	sla, err := manifestDigestInputs("sla-core")
	require.NoError(t, err)
	require.NotEmpty(t, sla)
}

// Appending one menu entry must change the digest, which is what a real
// manifest edit does at build time.
func TestHashComponentInputsIsSensitiveToSingleEntry(t *testing.T) {
	base := hashComponentInputs([]byte("payload"), "identity-rbac", []string{"menu=/a", "permission=x:read"})
	added := hashComponentInputs([]byte("payload"), "identity-rbac", []string{"menu=/a", "permission=x:read", "permission=y:read"})
	changed := hashComponentInputs([]byte("payload"), "identity-rbac", []string{"menu=/a", "permission=x:write"})
	other := hashComponentInputs([]byte("payload"), "itil-core", []string{"menu=/a", "permission=x:read"})

	assert.NotEqual(t, base, added)
	assert.NotEqual(t, base, changed)
	assert.NotEqual(t, base, other)
	assert.Equal(t, base, hashComponentInputs([]byte("payload"), "identity-rbac", []string{"menu=/a", "permission=x:read"}))
}

func configOnlyChecksum(payload []byte, component string) string {
	sum := sha256.Sum256(append(append([]byte(nil), payload...), []byte(":"+component)...))
	return hex.EncodeToString(sum[:])
}

func hasPrefixFrom(values []string, prefix string) bool {
	for _, value := range values {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
