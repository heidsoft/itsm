package rls

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readinessTestDB 构造 sqlmock 连接，返回 mock 句柄。
//
// CheckReadiness 只做两次只读查询，不依赖真实数据库，因此用 sqlmock 覆盖
// 全部 unready 分支比开容器集成测试更快、更稳定。
func readinessTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// expectRole 声明第一次查询（当前角色属性）的返回。
func expectRole(mock sqlmock.Sqlmock, user string, super, bypass bool) {
	mock.ExpectQuery(`SELECT current_user, rolsuper, rolbypassrls`).
		WillReturnRows(sqlmock.NewRows([]string{"current_user", "rolsuper", "rolbypassrls"}).
			AddRow(user, super, bypass))
}

// expectPosture 声明第二次查询（public schema RLS 姿态计数）的返回。
func expectPosture(mock sqlmock.Sqlmock, rlsTables, withoutPolicy, ownedNonForce int) {
	mock.ExpectQuery(`count\(\*\) FILTER`).
		WillReturnRows(sqlmock.NewRows([]string{"rls_tables", "without_policy", "owned_non_force"}).
			AddRow(rlsTables, withoutPolicy, ownedNonForce))
}

func TestCheckReadinessReadyPath(t *testing.T) {
	db, mock := readinessTestDB(t)
	expectRole(mock, "itsm_app", false, false)
	expectPosture(mock, 26, 0, 0)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.True(t, r.Ready(), "non-superuser, non-bypass, policies present => ready")
	assert.Empty(t, r.Reasons)
	assert.Equal(t, "itsm_app", r.CurrentUser)
	assert.Equal(t, 26, r.RLSTables)
	assert.NoError(t, EnforceReadinessError(r))
	assert.Contains(t, r.String(), "state=ready")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessRejectsSuperuser(t *testing.T) {
	db, mock := readinessTestDB(t)
	// 默认 dev/prod 连接账号就是 superuser——这是 enforce 无效的头号原因。
	expectRole(mock, "itsm_user", true, false)
	expectPosture(mock, 26, 0, 0)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.False(t, r.Ready())
	require.Len(t, r.Reasons, 1)
	assert.Contains(t, r.Reasons[0], "超级用户")
	assert.Contains(t, r.Reasons[0], "itsm_app", "reason must tell the operator what to do")

	err = EnforceReadinessError(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RLS_MODE=enforce")
	assert.Contains(t, err.Error(), "超级用户")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessRejectsBypassRLS(t *testing.T) {
	db, mock := readinessTestDB(t)
	expectRole(mock, "itsm_admin", false, true)
	expectPosture(mock, 26, 0, 0)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.False(t, r.Ready())
	require.Len(t, r.Reasons, 1)
	assert.Contains(t, r.Reasons[0], "BYPASSRLS")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessRejectsEmptyPolicyLayer(t *testing.T) {
	db, mock := readinessTestDB(t)
	expectRole(mock, "itsm_app", false, false)
	expectPosture(mock, 0, 0, 0)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.False(t, r.Ready())
	require.Len(t, r.Reasons, 1)
	assert.Contains(t, r.Reasons[0], "没有任何表启用 RLS")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessRejectsTablesWithoutPolicy(t *testing.T) {
	db, mock := readinessTestDB(t)
	expectRole(mock, "itsm_app", false, false)
	expectPosture(mock, 26, 2, 0)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.False(t, r.Ready())
	require.Len(t, r.Reasons, 1)
	assert.Contains(t, r.Reasons[0], "2 张表启用了 RLS 却没有 policy")
	// 没有 policy 的 RLS 表默认拒绝所有行——必须说清楚后果，否则会被当成
	// 「隔离很强」而不是「业务读不到数据」。
	assert.Contains(t, r.Reasons[0], "0 行")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessRejectsOwnerNonForce(t *testing.T) {
	db, mock := readinessTestDB(t)
	expectRole(mock, "itsm_user", false, false)
	expectPosture(mock, 26, 0, 26)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.False(t, r.Ready())
	require.Len(t, r.Reasons, 1)
	assert.Contains(t, r.Reasons[0], "未 FORCE")
	assert.Contains(t, r.Reasons[0], "FORCE ROW LEVEL SECURITY")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessAggregatesAllReasons(t *testing.T) {
	db, mock := readinessTestDB(t)
	expectRole(mock, "itsm_user", true, true)
	expectPosture(mock, 26, 1, 25)

	r, err := CheckReadiness(context.Background(), db)
	require.NoError(t, err)
	assert.False(t, r.Ready())
	// 五个维度同时不满足时应全部列出，而不是只报第一条——否则运维会
	// 一次只修一个问题，反复重启。
	assert.Len(t, r.Reasons, 4, "superuser + bypassrls + no-policy + owner-non-force")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckReadinessPropagatesQueryErrors(t *testing.T) {
	t.Run("role query", func(t *testing.T) {
		db, mock := readinessTestDB(t)
		mock.ExpectQuery(`SELECT current_user, rolsuper, rolbypassrls`).
			WillReturnError(errors.New("boom"))
		_, err := CheckReadiness(context.Background(), db)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "current role attributes")
	})

	t.Run("posture query", func(t *testing.T) {
		db, mock := readinessTestDB(t)
		expectRole(mock, "itsm_app", false, false)
		mock.ExpectQuery(`count\(\*\) FILTER`).WillReturnError(errors.New("boom"))
		_, err := CheckReadiness(context.Background(), db)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RLS posture")
	})

	t.Run("nil db", func(t *testing.T) {
		_, err := CheckReadiness(context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-nil")
	})
}

func TestEnforceReadinessErrorRejectsUnevaluated(t *testing.T) {
	// nil Readiness 不能被当成「没有原因 = 就绪」——那会把「探测根本没跑」
	// 静默解释成「隔离已生效」。
	err := EnforceReadinessError(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not evaluated")

	// 零值 Readiness 同理：Reasons 为空不代表检查通过，只代表没检查。
	err = EnforceReadinessError(&Readiness{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not evaluated")
	assert.False(t, (&Readiness{}).Ready(), "zero value must not be considered ready")
}

func TestReadinessStringMarksUnready(t *testing.T) {
	r := &Readiness{CurrentUser: "itsm_user", Superuser: true, RLSTables: 0}
	s := r.String()
	assert.True(t, strings.HasPrefix(s, "rls-readiness(state=unready"), s)
	assert.Contains(t, s, "user=itsm_user")
	assert.Contains(t, s, "superuser=true")
}
