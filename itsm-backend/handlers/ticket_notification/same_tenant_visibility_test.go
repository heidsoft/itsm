// Package ticket_notification 的同租户行级可见性回归测试（N2 修复）。
//
// 此前 GET /api/v1/tickets/:id/notifications 的谓词只有 TicketID+TenantID：
// 同租户任一持 notification:read 的账号枚举工单 ID 即可读走**所有收件人**的
// 通知正文与邮箱/角色/部门——「同租户」不等于「全员可见」。修复后：
//   - 普通角色（end_user）只能看到 user_id=自己 的通知行；
//   - 管理角色（datascope.IsDataScopeAllRole 单一词表）可见该工单全部通知行；
//   - 通知正文对无关者零泄漏（与跨租户契约同级硬约束）。
package ticket_notification_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// visibilityFixture 扩展 cross_tenant_http_test 的 fixture：补三类同租户角色
// 与一条发给另一位收件人的通知。
type visibilityFixture struct {
	*fixture
	member    *ent.User // end_user，非收件人
	manager   *ent.User // manager，管理旁路
	recipient *ent.User // end_user，另一条通知的收件人
}

func newVisibilityFixture(t *testing.T) *visibilityFixture {
	t.Helper()
	fx := newFixture(t)
	ctx := context.Background()

	newUser := func(username, email, name, role string) *ent.User {
		return fx.client.User.Create().
			SetUsername(username).SetEmail(email).SetName(name).
			SetPasswordHash("hash").SetRole(user.Role(role)).SetActive(true).SetTenantID(fx.tenantA.ID).
			SaveX(ctx)
	}

	m := newUser("notif-a-member", "member-a@example.com", "Member A", "end_user")
	mg := newUser("notif-a-manager", "manager-a@example.com", "Manager A", "manager")
	rc := newUser("notif-a-recipient", "recipient-a@example.com", "Recipient A", "end_user")

	// owner 通知已由 newFixture 创建（userA 收，内容「机密通知内容-A」）。
	// 再补一条发给 recipient 的通知。
	fx.client.TicketNotification.Create().
		SetTicketID(fx.ticketA.ID).
		SetUserID(rc.ID).
		SetType("ticket_assigned").
		SetChannel("in_app").
		SetContent("仅收件人可见-指派通知").
		SetTenantID(fx.tenantA.ID).
		SetStatus("sent").
		SaveX(ctx)

	return &visibilityFixture{
		fixture:   fx,
		member:    m,
		manager:   mg,
		recipient: rc,
	}
}

func TestTicketNotificationSameTenantVisibility(t *testing.T) {
	vf := newVisibilityFixture(t)
	fx := vf.fixture
	path := fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID)

	t.Run("非收件人普通角色只见自己名下通知（他人正文零泄漏）", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, path, vf.member, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)
		// member 不是任何通知的收件人 → 只能看到 0 条
		require.Contains(t, w.Body.String(), `"total":0`)
		require.Contains(t, w.Body.String(), `"items":[]`)
		assert.NotContains(t, w.Body.String(), "机密通知内容-A")
		assert.NotContains(t, w.Body.String(), "仅收件人可见-指派通知")
	})

	t.Run("收件人本人可见自己的通知", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, path, vf.recipient, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"total":1`)
		require.Contains(t, w.Body.String(), "仅收件人可见-指派通知")
		assert.NotContains(t, w.Body.String(), "机密通知内容-A", "不得看到发给 owner 的通知正文")
	})

	t.Run("owner（super_admin）走管理旁路可见全部通知行", func(t *testing.T) {
		// userA 是 super_admin，属管理角色：按单一词表看全量。
		// 「收件人只看自己」的正控制由 recipient 用例承担。
		w := fx.do(t, http.MethodGet, path, fx.userA, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"total":2`)
		require.Contains(t, w.Body.String(), "机密通知内容-A")
		require.Contains(t, w.Body.String(), "仅收件人可见-指派通知")
	})

	t.Run("管理角色可见全部通知行", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, path, vf.manager, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"total":2`)
		require.Contains(t, w.Body.String(), "机密通知内容-A")
		require.Contains(t, w.Body.String(), "仅收件人可见-指派通知")
	})
}
