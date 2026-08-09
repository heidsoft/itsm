package cmdb

import (
	"fmt"
	"net/url"
	"strings"
)

func validateTenantCredentialRef(ref string) error {
	if ref == "" {
		return nil
	}
	parsed, err := url.Parse(ref)
	if err != nil || parsed.Scheme != "secret" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("credentialRef must be an opaque tenant-scoped secret:// reference")
	}
	if strings.ContainsAny(parsed.Host+parsed.Path, "{}\n\r\t") {
		return fmt.Errorf("credentialRef contains invalid characters")
	}
	return nil
}
