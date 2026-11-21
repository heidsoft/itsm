package migration

// supersededMigration 登记「已发布、但内容被有意修正」的迁移版本。
//
// 背景（为什么需要这个白名单）：
//
//	schema_migrations.checksum 是发布制品的完整性锚点，改已应用迁移的 SQL 会在下一次
//	初始化时报 "migration checksum mismatch" 并硬失败——这条门禁本身必须保留，
//	它是「发布出去的东西没被偷偷改过」的唯一证据。
//
//	但有一类缺陷只能在文件本体上修：脚本在全新安装的早期阶段就 RAISE 中断整条
//	初始化链，任何「后置迁移」都来不及执行；而删掉文件又会让旧版本升级上来的实例
//	永久拿不到它负责的补码（MCP 通道会一直 403）。
//
// 因此这里放行「已经过评审的那一份内容」，而不是放行任意改动：
//   - 只有 checksum 精确等于 AllowedChecksum 时才降级为告警；
//   - 该版本后续任何再次改动都会重新变回硬失败。
//
// 新增条目的要求：必须写清缺陷、复现证据、以及对既有安装为何语义等价。
// 这不是「跳过校验」的通用开关，登记一项就要能回答上面三个问题。
type supersededMigration struct {
	// AllowedChecksum 是修正后文件内容的 sha256（与 checksumSQL 同算法、同输入）。
	AllowedChecksum string
	// Reason 供升级审计与 code review 阅读，不参与判定。
	Reason string
}

var supersededMigrations = map[string]supersededMigration{
	// 空卷首装被打断（fresh-install-gate run 44871855241）。
	//
	// 缺陷：Step 2 的 DO 块在 tenants 为空时 RAISE EXCEPTION，而 InitializeStorage
	// 先跑迁移流、再跑 seeder（internal/bootstrap/app.go），空卷上必然命中。
	// 证据：itsm-init 报 pq: mcp:read 权限码插入失败（tenant_id=1） (P0001)，
	// backend/worker/frontend 全部起不来。
	//
	// 语义等价性：修正只放宽「无租户」分支（此时 Step 1 本来就插入 0 行，
	// 旧版随后无条件抛错）；既有安装的租户早已存在，走的是与原脚本完全相同的
	// INSERT 路径，且该版本在既有安装上已记为 applied、不会再执行。
	// 同时把「按 tenant_id=1 判定」改为「按是否存在租户判定」，避免默认租户 id 不为 1
	// 的部署被误判为失败。
	"20261118_mcp_read_permission": {
		AllowedChecksum: "14a8d070150a859ed42637f75798e9f849074250723fcd2abe5dfc9ac64fcbbb",
		Reason: "空卷首装：tenants 为空时 DO 块 RAISE P0001 中断整条初始化链" +
			"（fresh-install-gate run 44871855241）；修正仅放宽无租户分支，既有安装语义不变",
	},
}
