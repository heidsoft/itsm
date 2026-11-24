package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"

	_ "github.com/mattn/go-sqlite3"
)

// ---------------------------------------------------------------------------
// P2-04 守卫：候选字段查询必须是「整 token 相等」，而不是子串 Contains。
//
// 审计发现：ListUserTasks 用 CandidateUsersContains/CandidateGroupsContains 做
// 子串匹配，候选 "12" 会被用户 "2" 命中（前缀/中缀碰撞），且与执行侧
// authorizeTaskActorWithClient 的 split+trim 精确口径不一致——查询面比执行面
// 更宽，等于查询泄露了执行时会被拒绝的任务。
//
// 本文件所有用例走 ListUserTasks 全路径（而非直接调 csvTokenEQ），
// 保证「调用点接线」本身被守卫：把调用点改回 Contains 时必须红。
// ---------------------------------------------------------------------------

type candidateFixture struct {
	client     *ent.Client
	tenantID   int
	instanceID int
	defKey     string
	ctx        context.Context
}

func seedCandidateFixture(t *testing.T) *candidateFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:bpmn_candidate_exact_match?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("cand-match").SetDomain("cand.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-CAND").SetDeploymentName("Cand").SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	definition, err := client.ProcessDefinition.Create().
		SetKey("cand_match").SetName("Cand").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-CAND-1").SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).SetStatus("running").
		SetVariables(map[string]interface{}{}).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	return &candidateFixture{
		client:     client,
		tenantID:   tenant.ID,
		instanceID: instance.ID,
		defKey:     definition.Key,
		ctx:        ctx,
	}
}

func (f *candidateFixture) createTask(t *testing.T, taskID, candidateUsers, candidateGroups string) {
	t.Helper()
	b := f.client.ProcessTask.Create().
		SetTaskID(taskID).SetTaskDefinitionKey("T").SetTaskName(taskID).
		SetProcessDefinitionKey(f.defKey).SetProcessInstanceID(f.instanceID).
		SetStatus("created").SetTenantID(f.tenantID).
		SetTaskVariables(map[string]interface{}{})
	if candidateUsers != "" {
		b = b.SetCandidateUsers(candidateUsers)
	}
	if candidateGroups != "" {
		b = b.SetCandidateGroups(candidateGroups)
	}
	_, err := b.Save(f.ctx)
	require.NoError(t, err)
}

// listByCandidate 走 ListUserTasks 的 AllTasks 筛选分支。
func (f *candidateFixture) listByCandidate(t *testing.T, field, token string) []string {
	t.Helper()
	svc := &bpmnTaskService{client: f.client, logger: zaptest.NewLogger(t).Sugar()}
	req := &dto.ListUserTasksRequest{TenantID: f.tenantID, AllTasks: true}
	if field == "users" {
		req.CandidateUsers = token
	} else {
		req.CandidateGroups = token
	}
	tasks, _, err := svc.ListUserTasks(f.ctx, req)
	require.NoError(t, err)
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.TaskID)
	}
	return ids
}

// TestListUserTasks_CandidateUserIDNoSubstringCollision 覆盖 ID 前/中/后缀碰撞：
// 查询 "2" 不得命中候选 "12" / "20" / "x2y"，查询 "12" 不得命中 "2"。
func TestListUserTasks_CandidateUserIDNoSubstringCollision(t *testing.T) {
	f := seedCandidateFixture(t)
	f.createTask(t, "T-12", "12", "")
	f.createTask(t, "T-2", "2", "")
	f.createTask(t, "T-20", "20", "")
	f.createTask(t, "T-X2Y", "x2y", "")
	f.createTask(t, "T-MID", "1,2,3", "")

	// token "2"：只命中整串 "2" 与中间 token 形态 "1,2,3"。
	require.ElementsMatch(t, []string{"T-2", "T-MID"}, f.listByCandidate(t, "users", "2"))
	// token "12"：只命中整串 "12"。
	require.ElementsMatch(t, []string{"T-12"}, f.listByCandidate(t, "users", "12"))
	// token "20"：只命中整串 "20"。
	require.ElementsMatch(t, []string{"T-20"}, f.listByCandidate(t, "users", "20"))
}

// TestListUserTasks_CandidateUsernameNoSubstringCollision username/email 形态碰撞：
// 查询 "admin" 不得命中 "admin2" / "2admin" / "xadminx"。
func TestListUserTasks_CandidateUsernameNoSubstringCollision(t *testing.T) {
	f := seedCandidateFixture(t)
	f.createTask(t, "T-ADMIN", "admin", "")
	f.createTask(t, "T-ADMIN2", "admin2", "")
	f.createTask(t, "T-2ADMIN", "2admin", "")
	f.createTask(t, "T-XADMINX", "xadminx", "")

	require.ElementsMatch(t, []string{"T-ADMIN"}, f.listByCandidate(t, "users", "admin"))
}

// TestListUserTasks_CandidateGroupNoSubstringCollision 组名碰撞：
// 查询 "ops" 不得命中 "ops-team"；逗号与 ", " 两种分隔形态都要命中。
func TestListUserTasks_CandidateGroupNoSubstringCollision(t *testing.T) {
	f := seedCandidateFixture(t)
	f.createTask(t, "T-OPS", "", "ops")
	f.createTask(t, "T-OPS-TEAM", "", "ops-team")
	f.createTask(t, "T-GROUP-LIST", "", "a,ops,b")
	f.createTask(t, "T-GROUP-SPACED", "", "a, ops, b")

	require.ElementsMatch(t, []string{"T-OPS", "T-GROUP-LIST", "T-GROUP-SPACED"},
		f.listByCandidate(t, "groups", "ops"))
}

// TestListUserTasks_CandidateTokenBoundaryForms 正例矩阵：整串 / 首 token / 末 token /
// 中间 token，以及 ", " 空格分隔变体，都必须命中（不能把精确匹配收紧成只认无空格形态）。
func TestListUserTasks_CandidateTokenBoundaryForms(t *testing.T) {
	f := seedCandidateFixture(t)
	f.createTask(t, "T-SOLO", "2", "")
	f.createTask(t, "T-HEAD", "2,3", "")
	f.createTask(t, "T-TAIL", "1,2", "")
	f.createTask(t, "T-MID", "1,2,3", "")
	f.createTask(t, "T-HEAD-SP", "2, 3", "")
	f.createTask(t, "T-TAIL-SP", "1, 2", "")
	f.createTask(t, "T-MID-SP", "1, 2, 3", "")

	require.ElementsMatch(t, []string{
		"T-SOLO", "T-HEAD", "T-TAIL", "T-MID", "T-HEAD-SP", "T-TAIL-SP", "T-MID-SP",
	}, f.listByCandidate(t, "users", "2"))
}

// TestCSVTokenEQ_EmptyTokenIsFalse 空 token 恒假：空串经 Contains 会命中所有非空行，
// 属越权放大而非筛选（调用点已过滤空串，此处守卫 helper 自身的防御语义）。
func TestCSVTokenEQ_EmptyTokenIsFalse(t *testing.T) {
	f := seedCandidateFixture(t)
	f.createTask(t, "T-A", "anything", "")
	f.createTask(t, "T-B", "other", "")

	var matched []string
	err := f.client.ProcessTask.Query().
		Where(csvTokenEQ(processtask.FieldCandidateUsers, "  ")).
		Select(processtask.FieldTaskID).
		Scan(f.ctx, &matched)
	require.NoError(t, err)
	require.Empty(t, matched, "空 token 必须恒假，不得命中任何行")
}
