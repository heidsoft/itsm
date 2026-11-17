package capability

// 能力矩阵的 readinessEnv 与部署契约必须保持同步。
//
// 背景：registry 是这些变量的唯一声明处，但如果 .env.example 与
// docker-compose.prod.yml 里没有它们，运维根本无从得知开关存在。本轮实测
// 5 个变量里只有 ITSM_AI_READY 在 compose 出现过，其余 4 个（云发现与三个
// 三方连接器）在两处都没有——于是 prod 上这些能力永远停在 deploymentReady=false，
// 而「该怎么打开」在部署产物里查不到任何线索。
//
// 本守卫反向扫描（从 registry 出发，而不是从文件出发）：新增 readinessEnv
// 而忘记接线时会直接失败。变量名必须精确出现在文件里，不做模糊匹配。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// readinessEnvRefPattern 从 registry 字面量里取 readinessEnv 的变量名。
var readinessEnvRefPattern = regexp.MustCompile(`readinessEnv: "([A-Z0-9_]+)"`)

// declaredReadinessEnvs 返回 registry 声明过的全部就绪申报变量名（去重）。
func declaredReadinessEnvs() []string {
	seen := make(map[string]struct{})
	for _, item := range registry {
		if item.readinessEnv == "" {
			continue
		}
		seen[item.readinessEnv] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	return out
}

// mustReadRepoFile 读取仓库根目录下的文件。
//
// 测试运行在 itsm-backend/handlers/capability，向上三级是仓库根。
// 文件缺失时直接失败：放宽成「跳过」会让守卫在路径变化后静默失效。
func mustReadRepoFile(t *testing.T, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", relative))
	require.NoError(t, err, "部署契约文件必须存在：%s", relative)
	return string(content)
}

func TestReadinessEnvVariablesExistInDeploymentContract(t *testing.T) {
	envExample := mustReadRepoFile(t, ".env.example")
	compose := mustReadRepoFile(t, "docker-compose.prod.yml")

	declared := declaredReadinessEnvs()
	require.NotEmpty(t, declared, "registry 至少要声明一个 readinessEnv，否则本守卫无意义")

	for _, key := range declared {
		key := key
		t.Run(key, func(t *testing.T) {
			// .env.example 要有以该变量名开头的赋值行（示例值可为空）
			assignment := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `=`)
			require.True(t, assignment.MatchString(envExample),
				"%s 必须在 .env.example 中列出，否则运维不知道要申报", key)

			// compose 要把宿主机变量透传进容器，且默认留空以保持既有行为
			expected := key + "=${" + key + ":-}"
			require.Contains(t, compose, expected,
				"docker-compose.prod.yml 必须透传 %s（默认留空）", key)
		})
	}
}

// registry 里的变量名一旦写错，矩阵会永远读到空值并静默降级。
// 这里锁住「声明处的字面量」本身：拼写必须与部署契约里的一致，
// 且不得出现 snake_case 之类的非约定写法。
func TestReadinessEnvLiteralSpelling(t *testing.T) {
	content, err := os.ReadFile("handler.go")
	require.NoError(t, err)

	matches := readinessEnvRefPattern.FindAllStringSubmatch(string(content), -1)
	require.NotEmpty(t, matches, "handler.go 中应当能解析出 readinessEnv 字面量")

	for _, m := range matches {
		key := m[1]
		require.True(t, strings.HasPrefix(key, "ITSM_"),
			"%s 必须以 ITSM_ 开头，否则与其余部署变量不成体系", key)
		require.True(t, strings.HasSuffix(key, "_READY"),
			"%s 必须以 _READY 结尾，语义是就绪申报而不是开关", key)
		require.Equal(t, strings.ToUpper(key), key, "%s 必须全大写", key)
	}
}
