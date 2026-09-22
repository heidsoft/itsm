package seeder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoadSeedConfigMergesGroupsOverride(t *testing.T) {
	// 相对路径解析依赖包目录 cwd，不经过 ITSM_SEED_CONFIG 分支。
	require.Empty(t, os.Getenv("ITSM_SEED_CONFIG"))
	embedded := getEmbeddedConfig()
	require.NotEmpty(t, embedded.Groups)
	require.NotEmpty(t, embedded.Departments)

	t.Run("json groups replace builtin", func(t *testing.T) {
		custom := `{"groups":[{"name":"approvers-custom","description":"tenant group"}],"teams":[{"name":"Team X","code":"team-x"}]}`
		path := filepath.Join(t.TempDir(), "seed.json")
		require.NoError(t, os.WriteFile(path, []byte(custom), 0o600))
		t.Setenv("ITSM_SEED_CONFIG", path)

		loaded := loadSeedConfig(zap.NewNop().Sugar())

		require.Len(t, loaded.Groups, 1)
		assert.Equal(t, "approvers-custom", loaded.Groups[0].Name)
		assert.Equal(t, embedded.Departments, loaded.Departments,
			"sections absent from both override files must survive the merge")
	})

	t.Run("absent groups keep builtin", func(t *testing.T) {
		custom := `{"teams":[{"name":"Only Teams","code":"only-teams"}]}`
		path := filepath.Join(t.TempDir(), "seed.json")
		require.NoError(t, os.WriteFile(path, []byte(custom), 0o600))
		t.Setenv("ITSM_SEED_CONFIG", path)

		loaded := loadSeedConfig(zap.NewNop().Sugar())

		assert.Equal(t, embedded.Groups, loaded.Groups)
	})
}
