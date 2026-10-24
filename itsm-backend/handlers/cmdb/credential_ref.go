package cmdb

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func validateTenantCredentialRef(ref string, tenantID int) error {
	if ref == "" {
		return nil
	}
	// 阿里云试点的部署级环境凭据是固定 allowlist；只接受这一个名称，
	// 防止请求参数读取任意进程环境变量。
	if ref == "env://ITSM_ALIYUN" {
		return nil
	}
	if tenantID <= 0 {
		return fmt.Errorf("tenant id is required to validate credentialRef")
	}
	parsed, err := url.Parse(ref)
	if err != nil || parsed.Scheme != "secret" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("credentialRef must be an opaque tenant-scoped secret:// reference")
	}
	if parsed.Host != "tenant-"+strconv.Itoa(tenantID) {
		return fmt.Errorf("credentialRef must reference the current tenant")
	}
	if strings.ContainsAny(parsed.Host+parsed.Path, "{}\n\r\t") {
		return fmt.Errorf("credentialRef contains invalid characters")
	}
	return nil
}
