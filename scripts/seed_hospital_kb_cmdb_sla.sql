-- =============================================================================
-- 医院信息科场景：知识库 SOP + CMDB + SLA 初始化数据
-- =============================================================================
-- 用法：
--   cat scripts/seed_hospital_kb_cmdb_sla.sql | \
--     docker exec -i itsm-postgres-prod psql -U itsm -d itsm_prod
--
-- 设计原则：
--   1. 全部使用 ON CONFLICT DO NOTHING，可重复运行
--   2. 租户隔离：全部 tenant_id=1
--   3. 复用现有数据：保留原 8 个 CI Types、3 个 SLA Policies
-- =============================================================================

\set ON_ERROR_STOP on

-- =============================================================================
-- 1. 知识库 SOP（医院信息科）
-- =============================================================================

-- 分类：临床信息系统 / 网络与基础设施 / 终端与外设 / 信息科管理
INSERT INTO knowledge_articles (title, content, category, tags, author_id, tenant_id, is_published, view_count, like_count, created_at, updated_at) VALUES
('HIS 系统升级标准操作 SOP',
'## 适用范围
HIS（医院信息系统）主版本升级、补丁升级。

## 前置准备
1. 升级窗口：02:00-06:00（凌晨维护时段）
2. 提前 3 天通知全院科室
3. 备份：数据库全量备份 + 配置备份 + 应用文件备份
4. 检查磁盘空间（>50GB）
5. 准备回滚方案（保留 7 天快照）

## 升级步骤
1. 切换负载均衡，将 HIS 应用从负载均衡器摘除
2. 停止 HIS 应用服务（systemctl stop his-app）
3. 停止 HIS 数据库连接池
4. 执行数据库迁移脚本（migrations/）
5. 部署新版本应用
6. 启动数据库连接池
7. 启动 HIS 应用服务
8. 验证登录、挂号、收费、医嘱四大主流程
9. 切回负载均衡

## 回滚条件
- 30 分钟内主流程未通过
- 数据库迁移报错
- 应用启动失败

## 联系人
- 厂商工程师：XXX-XXXXXXXX
- 信息科值班：0531-8888-ITSM',
'临床信息系统', 'HIS,升级,SOP,P0', 1, 1, true, 156, 23, NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 days'),

('护士站网络断网应急处理 SOP',
'## 触发条件
- 护士站 ≥3 台电脑同时无法访问 HIS/PACS
- 单台电脑故障转交桌面运维

## 应急响应（5 分钟内）
1. 信息科值班接到电话后立即启动应急
2. 通过 Zabbix 查看护士站交换机端口状态
3. 远程 ping 核心交换机，定位故障层
4. 通知护士站启动纸质医嘱流程
5. 通知科室主任和医务科

## 现场处理（30 分钟内）
1. 到达现场，戴好工牌
2. 检查网线、光模块状态
3. 备用 AP 启用（护士站临时无线方案）
4. 笔记本电脑 + 4G 网卡临时接入 HIS

## 恢复后
1. 填写故障报告
2. SLA 计时自动暂停
3. 创建 Problem 跟踪根因
4. 24 小时内提交 RCA 报告',
'网络与基础设施', '网络,应急,护士站,P1', 1, 1, true, 89, 18, NOW() - INTERVAL '25 days', NOW() - INTERVAL '3 days'),

('打印机脱机处理 SOP',
'## 常见原因
1. IP 地址冲突（最常见，70% 故障）
2. 驱动问题
3. 打印队列卡死
4. 网线松动
5. 硒鼓/碳粉耗尽

## 远程排查（5 分钟）
1. 远程桌面登录出问题的电脑
2. ping 打印机 IP
3. 检查打印队列（清除卡住的任务）
4. 查看打印机共享状态

## 现场处理（15 分钟）
1. 重启打印机
2. 更换网线
3. 重新分配 IP（DHCP 保留）
4. 重装驱动

## 预防措施
- 每季度巡检打印机 IP 分配
- 重要科室打印机做双机热备',
'终端与外设', '打印机,脱机,SOP,P3', 1, 1, true, 234, 45, NOW() - INTERVAL '60 days', NOW() - INTERVAL '10 days'),

('手术室 PACS 影像调阅故障处理 SOP',
'## 故障影响
P0 级故障：影响手术进行，必须立即响应。

## 应急方案（5 分钟内）
1. 启用备用影像服务器 PACS-BAK
2. 通知手术室使用术中影像离线缓存
3. 通知放射科协助调阅

## 排查步骤
1. 检查 PACS 应用服务器状态
2. 检查 PACS 数据库连接
3. 检查 DICOM 监听端口（104）
4. 检查存储空间
5. 查看 PACS 日志（/var/log/pacs/）

## 恢复验证
- 手术室 3 号、5 号、7 号终端测试调阅
- 调阅一张近期 CT 验证完整性
- 测试 1 小时内新建影像入库

## 联系
- PACS 厂商：XX 公司 7x24 电话
- 信息科值班：0531-8888-ITSM',
'临床信息系统', 'PACS,手术室,P0,影像', 1, 1, true, 67, 12, NOW() - INTERVAL '15 days', NOW() - INTERVAL '2 days'),

('LIS 检验报告系统故障处理 SOP',
'## 故障分级
- P0：急诊检验报告延迟 >15 分钟
- P1：普通检验报告延迟 >1 小时
- P2：历史报告查询失败

## 应急响应
1. P0 立即电话通知值班主任
2. 启用 LIS 备用数据库（只读模式）
3. 通知检验科手动出报告

## 排查
1. 检查 LIS 应用服务
2. 检查分析仪接口状态（ASTM/HL7）
3. 检查 LIS 数据库连接
4. 检查结果回传队列

## 恢复后
1. 重新执行分析仪接口对接
2. 补传延迟报告
3. 更新 SLA 状态',
'临床信息系统', 'LIS,检验,P0,P1', 1, 1, true, 78, 14, NOW() - INTERVAL '20 days', NOW() - INTERVAL '1 day'),

('心电监护仪故障应急处理 SOP',
'## 故障分级
- P0：术中/ICU 监护仪故障
- P1：普通病房监护仪故障

## 应急步骤（P0）
1. 立即电话响应（5 分钟内）
2. 启动备用监护仪
3. 通知设备科 + 临床工程师
4. 现场排查（30 分钟内）

## 排查要点
- 电源/电池状态
- 导联线连接
- 网关/中央站连接
- 固件版本

## 替代方案
- 启用便携式监护仪
- 人工定时监测 + 手工记录',
'临床信息系统', '监护仪,P0,ICU,生命体征', 1, 1, true, 56, 9, NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 day'),

('医院无线网络 AP 故障处理 SOP',
'## 故障特征
- 单 AP：覆盖区域无信号
- 多 AP：楼层/楼栋大面积断网
- 间歇性：患者投诉频繁掉线

## 应急响应
1. 远程检查 AC 控制器状态
2. 查看 AP 上线状态
3. 重启故障 AP（远程 PoE 断电）
4. 启用相邻 AP 增大功率

## 现场处理
1. 检查 PoE 交换机端口
2. 检查 AP 物理连接
3. 检查 AP 固件版本
4. 必要时更换 AP

## 关键 AP 列表（影响业务的）
- 手术室 3/5/7 号
- ICU 全部
- 急诊抢救室
- 护士站',
'网络与基础设施', '无线,AP,网络,P1', 1, 1, true, 145, 32, NOW() - INTERVAL '40 days', NOW() - INTERVAL '7 days'),

('信息科值班交接 SOP',
'## 交接时间
每日 08:30 / 17:30（两班倒）
节假日不间断

## 交接内容
1. 在处理工单列表
2. SLA 即将超期工单
3. 已知问题（监控告警、供应商故障）
4. 待跟进事项
5. 备品备件库存

## 交接清单
- [ ] 查看 ITSM 工单看板
- [ ] 查看 Zabbix 告警
- [ ] 查看值班手机
- [ ] 检查备件柜
- [ ] 上签交接记录表

## 注意事项
- 不得擅自关闭 P0/P1 工单
- 升级超过 30 分钟未响应必须报备科主任',
'信息科管理', '值班,交接,SOP,管理', 1, 1, true, 312, 78, NOW() - INTERVAL '90 days', NOW() - INTERVAL '15 days'),

('患者数据备份与恢复 SOP',
'## 备份策略
- 全量备份：每周日 02:00
- 增量备份：每日 02:00（除周日）
- binlog 备份：实时同步到异地
- 备份保留：本地 7 天，异地 30 天

## 备份介质
- 本地：NAS（RAID6）
- 异地：腾讯云对象存储 COS
- 离线：磁带机（每月一次）

## 恢复流程
1. 申请《数据恢复审批单》
2. 信息科主任 + 医务科主任双签
3. 在测试环境先验证备份完整性
4. 选择恢复时间点
5. 执行恢复 + 完整性验证

## 演练
每季度一次恢复演练（最近一次：2026-06-15）',
'信息科管理', '备份,恢复,数据,SOP', 1, 1, true, 89, 21, NOW() - INTERVAL '50 days', NOW() - INTERVAL '8 days'),

('医保接口异常处理 SOP',
'## 故障分级
- P0：医保实时结算完全中断
- P1：部分交易失败或延迟
- P2：医保对账异常

## 应急响应
1. P0：启动手工结算过渡（10 分钟内）
2. 联系医保接口厂商工程师
3. 查看医保前置机日志
4. 通知财务科和门诊办

## 排查
1. 网络连通性（医保专线）
2. 前置机服务状态
3. 数字证书有效期
4. 接口版本匹配

## 恢复后
1. 重新上传延迟交易
2. 与医保中心对账
3. 故障报告归档',
'临床信息系统', '医保,接口,P0,P1', 1, 1, true, 67, 11, NOW() - INTERVAL '8 days', NOW() - INTERVAL '1 day'),

('EMR 电子病历系统故障处理 SOP',
'## 故障分级
- P0：全院 EMR 无法访问
- P1：特定科室 EMR 故障
- P2：部分功能异常（医嘱/病历书写）

## 应急响应
1. 切换到 EMR 备用集群（异地灾备）
2. 通知医务科启动纸质病历过渡
3. 排查 EMR 主集群故障原因

## 排查
1. EMR 应用服务器集群状态
2. EMR 数据库主从状态
3. 文件存储（NFS）状态
4. 电子签名服务

## 恢复后
1. 病历数据补录（电子化扫描归档）
2. 医嘱补录
3. 与纸质病历对照复核',
'临床信息系统', 'EMR,病历,P0,灾备', 1, 1, true, 98, 19, NOW() - INTERVAL '12 days', NOW() - INTERVAL '2 days'),

('医院信息系统账号开通回收 SOP',
'## 开通流程
1. 申请人填写《系统账号申请单》
2. 申请人所在科室主任审批
3. 信息科审核权限范围
4. 创建账号（最小权限原则）
5. 通知申请人初始密码（强制首次登录修改）
6. 培训系统使用

## 必开系统
- HIS（医生/护士）
- PACS（医生）
- LIS（医生/护士）
- EMR（医生）
- OA（行政）

## 回收流程
1. 离职/转岗申请单
2. 关闭账号（不删除，保留审计）
3. 30 天后归档
4. 归档保留 2 年

## 密码策略
- 长度 ≥10 位
- 包含大小写+数字+特殊字符
- 90 天强制更换
- 5 次错误锁定 30 分钟',
'信息科管理', '账号,权限,SOP,合规', 1, 1, true, 256, 67, NOW() - INTERVAL '120 days', NOW() - INTERVAL '20 days')
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 2. CI Types 补充（医疗设备 + 无线 AP）
-- =============================================================================

INSERT INTO ci_types (name, description, icon, color, attribute_schema, tenant_id, is_active, created_at, updated_at) VALUES
('medical_device', '医疗设备（监护仪、输液泵、呼吸机等）', 'heart-pulse', '#ff4d4f',
 '{"fields": [{"name": "device_type", "type": "enum", "options": ["心电监护仪", "输液泵", "呼吸机", "麻醉机", "除颤仪"]}, {"name": "risk_level", "type": "enum", "options": ["生命支持", "治疗辅助", "监测辅助"]}, {"name": "pm_cycle_days", "type": "number", "default": 90}]}',
 1, true, NOW(), NOW()),
('wireless_ap', '无线接入点', 'wifi', '#52c41a',
 '{"fields": [{"name": "ssid", "type": "string"}, {"name": "frequency_band", "type": "enum", "options": ["2.4G", "5G", "双频"]}, {"name": "max_clients", "type": "number", "default": 50}]}',
 1, true, NOW(), NOW())
ON CONFLICT (tenant_id, name) DO NOTHING;

-- =============================================================================
-- 3. Configuration Items（医院实例）
-- =============================================================================
-- 注意：保留已有 1 条 CI，按名称去重（不做 id 冲突）

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT 'HIS 应用服务器', '医院信息系统核心应用', id, 'server', 'active', 'production', 'critical',
 'SRV-HIS-001', 'SN-HIS-2024-001', 'Dell PowerEdge R750', 'Dell',
 '中心机房-U1', '信息科', '信息科', 'managed',
 '{"ip": "10.1.1.10", "cpu_cores": 32, "memory_gb": 128, "os": "CentOS 7.9"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='server' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT 'HIS 数据库服务器', 'Oracle RAC 集群节点1', id, 'database', 'active', 'production', 'critical',
 'DB-HIS-001', 'SN-DB-HIS-001', 'Oracle Exadata X8', 'Oracle',
 '中心机房-U2', '信息科', '信息科', 'managed',
 '{"ip": "10.1.2.10", "db_version": "Oracle 19c", "storage_tb": 10, "backup_strategy": "RMAN+daily"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='database' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT 'PACS 应用服务器', '影像归档与通信系统', id, 'application', 'active', 'production', 'critical',
 'APP-PACS-001', 'SN-PACS-001', 'Dell PowerEdge R650', 'Dell',
 '中心机房-U3', '信息科', '信息科', 'managed',
 '{"ip": "10.1.3.10", "dicom_port": 104, "storage_tb": 50}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='application' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT 'LIS 应用服务器', '检验信息系统', id, 'application', 'active', 'production', 'high',
 'APP-LIS-001', 'SN-LIS-001', 'HPE ProLiant DL380', 'HPE',
 '中心机房-U4', '信息科', '信息科', 'managed',
 '{"ip": "10.1.3.11", "analyzer_connections": 12}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='application' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT 'EMR 应用服务器', '电子病历系统', id, 'application', 'active', 'production', 'critical',
 'APP-EMR-001', 'SN-EMR-001', 'Dell PowerEdge R750', 'Dell',
 '中心机房-U5', '信息科', '信息科', 'managed',
 '{"ip": "10.1.3.12", "storage_mode": "NFS集群"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='application' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '核心交换机', '医院核心网络交换（万兆骨干）', id, 'network', 'active', 'production', 'critical',
 'NET-CORE-001', 'SN-CORE-001', 'Cisco Catalyst 9500', 'Cisco',
 '中心机房-机柜A', '信息科', '信息科', 'managed',
 '{"ip": "10.1.0.1", "ports": 48, "uplink": "10G×4", "vlan_count": 32}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='network' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '防火墙', '医院边界安全防护', id, 'network', 'active', 'production', 'high',
 'NET-FW-001', 'SN-FW-001', '深信服 AF-2000-FH2130', '深信服',
 '中心机房-机柜A', '信息科', '信息科', 'managed',
 '{"ip": "10.1.0.254", "throughput_gbps": 8, "session_count": 2000000}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='network' LIMIT 1;

-- 医疗设备（medical_device 类型）
INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '心电监护仪-ICU-01', 'ICU 中央监护站 1 号', id, 'medical_device', 'active', 'production', 'critical',
 'MD-MON-ICU-01', 'SN-MON-2024-001', 'Mindray ePM12M', '迈瑞',
 'ICU 中央站', 'ICU 护士长', '设备科', 'managed',
 '{"device_type": "心电监护仪", "risk_level": "生命支持", "pm_cycle_days": 90, "ip": "10.2.1.10", "install_date": "2024-03-15"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='medical_device' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '心电监护仪-CCU-01', 'CCU 中央监护站 1 号', id, 'medical_device', 'active', 'production', 'critical',
 'MD-MON-CCU-01', 'SN-MON-2024-002', 'Philips IntelliVue MX450', 'Philips',
 'CCU 中央站', 'CCU 护士长', '设备科', 'managed',
 '{"device_type": "心电监护仪", "risk_level": "生命支持", "pm_cycle_days": 90, "ip": "10.2.1.11"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='medical_device' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '呼吸机-ICU-01', 'ICU 呼吸机 1 号', id, 'medical_device', 'active', 'production', 'critical',
 'MD-VENT-ICU-01', 'SN-VENT-2024-001', 'Mindray SV300', '迈瑞',
 'ICU 病床 3 号', 'ICU 护士长', '设备科', 'managed',
 '{"device_type": "呼吸机", "risk_level": "生命支持", "pm_cycle_days": 60}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='medical_device' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '输液泵-心内-01', '心内科输液泵', id, 'medical_device', 'active', 'production', 'high',
 'MD-PUMP-XN-01', 'SN-PUMP-2024-001', 'Braun Infusomat Space', '贝朗',
 '心内科护士站', '心内护士长', '设备科', 'managed',
 '{"device_type": "输液泵", "risk_level": "治疗辅助", "pm_cycle_days": 180}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='medical_device' LIMIT 1;

-- 无线 AP
INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '无线AP-手术室-3', '3号手术室无线 AP', id, 'wireless_ap', 'active', 'production', 'critical',
 'AP-OR-3', 'SN-AP-OR-3', 'Aruba AP-535', 'Aruba',
 '3号手术室天花板', '信息科', '信息科', 'managed',
 '{"ssid": "Hospital-Internal", "frequency_band": "双频", "max_clients": 100, "ip": "10.3.1.30"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='wireless_ap' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '无线AP-护士站-心内', '心内科护士站无线 AP', id, 'wireless_ap', 'active', 'production', 'high',
 'AP-NS-XN', 'SN-AP-NS-XN', 'Aruba AP-535', 'Aruba',
 '心内科护士站', '信息科', '信息科', 'managed',
 '{"ssid": "Hospital-Internal", "frequency_band": "双频", "max_clients": 80, "ip": "10.3.2.10"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='wireless_ap' LIMIT 1;

INSERT INTO configuration_items (name, description, ci_type_id, ci_type, status, environment, criticality, asset_tag, serial_number, model, vendor, location, assigned_to, owned_by, ownership_mode, attributes, tenant_id, created_at, updated_at)
SELECT '无线AP-急诊', '急诊科无线 AP', id, 'wireless_ap', 'active', 'production', 'critical',
 'AP-ER', 'SN-AP-ER', 'Aruba AP-535', 'Aruba',
 '急诊抢救室', '信息科', '信息科', 'managed',
 '{"ssid": "Hospital-Internal", "frequency_band": "双频", "max_clients": 100, "ip": "10.3.3.10"}'::jsonb,
 1, NOW(), NOW()
FROM ci_types WHERE tenant_id=1 AND name='wireless_ap' LIMIT 1;

-- =============================================================================
-- 4. CI 关系（建立依赖图）
-- =============================================================================
-- 关系：HIS 应用 → HIS 数据库、PACS 应用 → HIS 数据库、心电监护 → 无线 AP

-- HIS 应用 → HIS 数据库
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'depends_on',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='HIS 应用服务器'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='HIS 数据库服务器'),
  'critical', 'critical', true, 'HIS 应用强依赖 Oracle 数据库',
  '{"dependency_type": "data_layer", "failover": "RAC切换"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='HIS 应用服务器')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='HIS 数据库服务器');

-- PACS 应用 → HIS 数据库
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'depends_on',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='PACS 应用服务器'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='HIS 数据库服务器'),
  'high', 'high', true, 'PACS 影像元数据存储在 HIS DB',
  '{"dependency_type": "metadata"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='PACS 应用服务器')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='HIS 数据库服务器');

-- 核心交换机 → 防火墙
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'connects_to',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='核心交换机'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='防火墙'),
  'critical', 'critical', true, '核心交换机上行连接防火墙',
  '{"link_type": "uplink"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='核心交换机')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='防火墙');

-- HIS 服务器 → 核心交换机
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'connects_to',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='HIS 应用服务器'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='核心交换机'),
  'critical', 'critical', true, 'HIS 服务器接入核心交换机',
  '{"port": "Gi0/1", "vlan": 10}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='HIS 应用服务器')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='核心交换机');

-- 心电监护 ICU → 无线AP护士站心内（通过无线网络）
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'depends_on',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='心电监护仪-ICU-01'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='无线AP-护士站-心内'),
  'high', 'high', true, '监护数据通过无线网络上传中央站',
  '{"protocol": "HL7"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='心电监护仪-ICU-01')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='无线AP-护士站-心内');

-- 无线AP → 核心交换机
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'connects_to',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='无线AP-手术室-3'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='核心交换机'),
  'high', 'high', true, '手术室 AP 上联核心交换机',
  '{"port": "Gi0/10"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='无线AP-手术室-3')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='核心交换机');

INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'connects_to',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='无线AP-护士站-心内'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='核心交换机'),
  'high', 'high', true, '心内护士站 AP 上联核心交换机',
  '{"port": "Gi0/11"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='无线AP-护士站-心内')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='核心交换机');

-- EMR 应用 → HIS 数据库（共享部分数据）
INSERT INTO ci_relationships (tenant_id, relationship_type, source_ci_id, target_ci_id, strength, impact_level, is_active, description, metadata)
SELECT 1, 'depends_on',
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='EMR 应用服务器'),
  (SELECT id FROM configuration_items WHERE tenant_id=1 AND name='HIS 数据库服务器'),
  'high', 'high', true, 'EMR 调阅 HIS 中的患者基础信息',
  '{"dependency_type": "patient_master"}'::jsonb
WHERE EXISTS (SELECT 1 FROM configuration_items WHERE name='EMR 应用服务器')
  AND EXISTS (SELECT 1 FROM configuration_items WHERE name='HIS 数据库服务器');

-- =============================================================================
-- 5. SLA 策略补充（医院分级响应）
-- =============================================================================
-- 保留现有 3 条，新增 1 条 critical（医疗场景 P0 级）和 1 条 request 类型

INSERT INTO sla_policies (name, description, customer_tier, ticket_type, priority, response_time_minutes, resolution_time_minutes, business_hours, exclude_weekends, exclude_holidays, escalation_rules, is_active, priority_score, tenant_id, created_at, updated_at) VALUES
('医疗P0紧急SLA-24x7', '生命体征相关系统（PACS/HIS/LIS/监护）紧急故障专用，全天候响应', 'platinum', 'incident', 'critical', 5, 30,
 '{"work_days": [1,2,3,4,5,6,7], "start_time": "00:00", "end_time": "23:59", "time_zone": "Asia/Shanghai"}'::jsonb,
 false, false,
 '{"level1": {"minutes": 5, "role": "值班工程师"}, "level2": {"minutes": 15, "role": "信息科主任"}, "level3": {"minutes": 30, "role": "分管院长"}}'::jsonb,
 true, 100, 1, NOW(), NOW()),

('医疗P1高优SLA', '护士站/医生站等核心业务故障，8 小时内解决', 'gold', 'incident', 'high', 15, 120,
 '{"work_days": [1,2,3,4,5], "start_time": "08:00", "end_time": "18:00", "time_zone": "Asia/Shanghai"}'::jsonb,
 false, true,
 '{"level1": {"minutes": 15, "role": "值班工程师"}, "level2": {"minutes": 60, "role": "信息科主任"}}'::jsonb,
 true, 80, 1, NOW(), NOW()),

('医疗P2中优SLA', '内部办公系统故障', 'silver', 'incident', 'medium', 60, 480,
 '{"work_days": [1,2,3,4,5], "start_time": "08:00", "end_time": "18:00", "time_zone": "Asia/Shanghai"}'::jsonb,
 true, true,
 '{"level1": {"minutes": 60, "role": "工程师"}, "level2": {"minutes": 240, "role": "信息科主任"}}'::jsonb,
 true, 50, 1, NOW(), NOW()),

('医疗P3低优SLA', '非紧急请求、培训类需求', 'bronze', 'request', 'low', 240, 1440,
 '{"work_days": [1,2,3,4,5], "start_time": "09:00", "end_time": "17:00", "time_zone": "Asia/Shanghai"}'::jsonb,
 true, true,
 '{"level1": {"minutes": 240, "role": "工程师"}}'::jsonb,
 true, 20, 1, NOW(), NOW()),

('变更类工单SLA', '所有变更（Change）类工单的评估和实施时限', '', 'change', 'medium', 240, 2880,
 '{"work_days": [1,2,3,4,5], "start_time": "09:00", "end_time": "17:00", "time_zone": "Asia/Shanghai"}'::jsonb,
 true, true,
 '{"level1": {"minutes": 240, "role": "CAB 审批"}, "level2": {"minutes": 1440, "role": "实施"}}'::jsonb,
 true, 30, 1, NOW(), NOW()),

('问题管理SLA', 'Problem 工单的根因分析时限', '', 'problem', 'medium', 480, 7200,
 '{"work_days": [1,2,3,4,5], "start_time": "09:00", "end_time": "17:00", "time_zone": "Asia/Shanghai"}'::jsonb,
 true, true,
 '{"level1": {"minutes": 480, "role": "高级工程师"}, "level2": {"minutes": 2880, "role": "信息科主任"}}'::jsonb,
 true, 40, 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 验证
-- =============================================================================

SELECT 'knowledge_articles' as table_name, count(*) as total, count(*) FILTER (WHERE is_published=true) as published FROM knowledge_articles WHERE tenant_id=1
UNION ALL
SELECT 'ci_types', count(*), count(*) FILTER (WHERE is_active=true) FROM ci_types WHERE tenant_id=1
UNION ALL
SELECT 'configuration_items', count(*), count(*) FILTER (WHERE status='active') FROM configuration_items WHERE tenant_id=1
UNION ALL
SELECT 'ci_relationships', count(*), count(*) FILTER (WHERE is_active=true) FROM ci_relationships WHERE tenant_id=1
UNION ALL
SELECT 'sla_policies', count(*), count(*) FILTER (WHERE is_active=true) FROM sla_policies WHERE tenant_id=1;
