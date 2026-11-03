package alicloud

import "testing"

// 错误码分类守卫：永久错误必须命中，可重试错误绝不能误判。
// 误判方向两边都危险——永久错误重投会刷假告警，临时错误判永久会让一次
// 网络抖动变成「交付永久失败」。
func TestIsPermanentAliyunCode(t *testing.T) {
	permanent := []string{
		"InvalidAccountStatus.NotEnoughBalance",
		"InvalidAccountStatus.Forbidden",
		"QuotaExceeded.Instance",
		"InvalidVSwitchId.NotFound",
		"InvalidSecurityGroupId.NotFound",
		"InvalidImageId.NotFound",
		"InvalidInstanceType.ValueNotSupported",
		"InvalidParameter.Mismatch",
		"Forbidden.RAM",
		"OperationDenied.NoStock",
		// 2026-11-03 真实账号验证实测：地域配错时云侧返回 400
		// 「The specified endpoint can't operate this region」，此前被判可重试。
		"InvalidOperation.NotSupportedEndpoint",
		"InvalidOperation.RegionNotSupported",
	}
	for _, code := range permanent {
		if !isPermanentAliyunCode(code) {
			t.Errorf("%s 必须判为永久错误（重试不会成功）", code)
		}
	}

	retryable := []string{
		"",
		"Throttling",
		"Throttling.User",
		"ServiceUnavailable",
		"InternalError",
		"RequestTimeout",
		"UnknownError",
		// InvalidOperation 命名空间下还有「实例状态冲突、资源正在变更」这类
		// 临时拒绝，等状态变化后重试有意义，不能整段判永久。
		"InvalidOperation.IncorrectInstanceStatus",
		"InvalidOperation.ResourceInUse",
	}
	for _, code := range retryable {
		if isPermanentAliyunCode(code) {
			t.Errorf("%q 不得判为永久错误（重试有意义）", code)
		}
	}
}
