// Package credential 是凭据强度判定的唯一手写源。
//
// 背景：管理员密码在两条路径上使用——
//  1. internal/bootstrap 的启动守卫（弱口令 fatal 拒启动）；
//  2. pkg/seeder 的引导建号 / 运维显式重置。
//
// 两条路径过去各写一份弱口令列表，改一处就会漂移。这里收敛为唯一源，
// 两个包都只调用本包，禁止再内联列表。
package credential

import "strings"

// MinAdminPasswordLength 是管理员密码的最小长度。仅用于「显式重置/新建」路径，
// 不用于启动守卫——启动守卫只拦截已知弱口令，避免扩大既有部署的 fatal 面。
const MinAdminPasswordLength = 12

// weakAdminPasswords 是已知的默认/弱管理员口令。命中即视为不可用。
var weakAdminPasswords = []string{
	"admin", "admin123", "password", "123456", "itsm123", "changeme",
}

// IsWeakAdminPassword 判定口令是否属于已知弱口令。
// 空口令返回 true：生产环境「没设 ADMIN_PASSWORD」等同于使用默认值。
func IsWeakAdminPassword(password string) bool {
	if strings.TrimSpace(password) == "" {
		return true
	}
	lower := strings.ToLower(password)
	for _, weak := range weakAdminPasswords {
		if lower == weak {
			return true
		}
	}
	return false
}

// IsShortAdminPassword 判定口令是否短于管理员最小长度要求。
// 空口令返回 true，便于调用方一次性得到「不可用」结论。
func IsShortAdminPassword(password string) bool {
	return len(password) < MinAdminPasswordLength
}
