package aliyun

import (
	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"

	"itsm-backend/service/cloud"
)

// 只读探测工具（cmd/aliyun-connection-check 等）复用的导出包装。
// 不放进 ecs.go 是为了避开并行批次的同文件编辑；语义与适配器内部完全一致。

// NewECSClientForProbe 按 Region + 运行时凭据构造 ECS 客户端。
func NewECSClientForProbe(region string, cred *cloud.ResolvedCredential) (*ecs.Client, error) {
	return newECSClient(region, cred)
}

// ProbeRuntimeOptions 与发现链路同一套超时/不重试语义。
func ProbeRuntimeOptions() *dara.RuntimeOptions {
	return runtimeOptions()
}
