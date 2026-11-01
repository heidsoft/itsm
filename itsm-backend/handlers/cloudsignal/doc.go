// Package cloudsignal 负责云监控告警与云安全中心漏洞的采集、落库与查询。
//
// 采集由 worker 定时轮询完成（不依赖云厂商推送）：告警复用 alerts 表，
// 漏洞写入 security_findings，每个账号每种信号的游标与健康状态记在
// cloud_signal_sync_states。两类数据都不走 Ent：本机 Ent codegen 不可用，
// 新增表一律用版本化迁移 + 原生 SQL。
package cloudsignal
