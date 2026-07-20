'use client';

import { useMemo, useState } from 'react';
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  Bot,
  Check,
  CheckCircle2,
  ChevronRight,
  CircleDot,
  Clock3,
  Database,
  GitBranch,
  LockKeyhole,
  Network,
  Play,
  Radio,
  Server,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
  UserCheck,
  X,
  Zap,
} from 'lucide-react';
import styles from './prototype.module.css';

type ActionState = 'pending' | 'approved' | 'executed';

const evidence = [
  { time: '14:03:12', label: 'Prometheus', detail: 'checkout-api P95 延迟超过 2.5s', tone: 'danger' },
  { time: '14:03:19', label: '日志关联', detail: '支付连接池耗尽，错误率 18.7%', tone: 'danger' },
  { time: '14:03:27', label: 'CMDB', detail: '影响订单中心及 3 个下游服务', tone: 'warning' },
  { time: '14:03:41', label: '历史事件', detail: '匹配 INC-2025-0817，相似度 92%', tone: 'info' },
];

const guardrails = [
  ['身份', 'SRE Copilot · on behalf of 刘洋'],
  ['授权范围', 'production / checkout-api'],
  ['风险等级', 'L3 · 必须人工批准'],
  ['有效窗口', '单次授权 · 10 分钟'],
];

export default function AgentOpsDemoPage() {
  const [actionState, setActionState] = useState<ActionState>('pending');
  const [showTrace, setShowTrace] = useState(false);

  const status = useMemo(() => {
    if (actionState === 'executed') return { label: '执行成功', detail: '连接池已恢复，错误率持续下降', icon: CheckCircle2 };
    if (actionState === 'approved') return { label: '等待执行', detail: '授权已签发，执行窗口剩余 09:42', icon: Play };
    return { label: '等待批准', detail: '高风险动作已被策略引擎拦截', icon: LockKeyhole };
  }, [actionState]);

  const StatusIcon = status.icon;

  const approve = () => setActionState('approved');
  const execute = () => setActionState('executed');
  const reset = () => {
    setActionState('pending');
    setShowTrace(false);
  };

  return (
    <main className={styles.shell}>
      <div className={styles.gridGlow} />
      <header className={styles.topbar}>
        <div className={styles.brand}>
          <span className={styles.brandMark}><CircleDot size={18} /></span>
          <span>HEID<span>CONTROL</span></span>
          <em>PROTOTYPE 01</em>
        </div>
        <nav className={styles.nav} aria-label="原型导航">
          <span className={styles.live}><i /> LIVE INCIDENT</span>
          <span>Agent Control Plane</span>
          <span>审计中心</span>
        </nav>
        <button className={styles.operator}><span>刘</span> 值班负责人</button>
      </header>

      <section className={styles.hero}>
        <div>
          <div className={styles.eyebrow}><Radio size={14} /> SEV-1 · 进行中 · 14:03 触发</div>
          <h1>支付链路异常<span>，AI 正在受控处置</span></h1>
          <p>INC-2026-0719 · checkout-api · 华东生产集群</p>
        </div>
        <div className={styles.heroMetrics}>
          <div><span>当前影响</span><strong>18.7%</strong><small>支付失败率</small></div>
          <div><span>已用时间</span><strong>08:42</strong><small>目标 30 分钟内恢复</small></div>
          <div><span>AI 置信度</span><strong>92%</strong><small>4 项证据已交叉验证</small></div>
        </div>
      </section>

      <section className={styles.workspace}>
        <div className={styles.timelinePanel}>
          <div className={styles.panelHead}>
            <div><span>01</span><h2>证据时间线</h2></div>
            <button onClick={reset}>重置演示</button>
          </div>
          <div className={styles.timeline}>
            {evidence.map((item, index) => (
              <article key={item.time} className={styles.timelineItem}>
                <time>{item.time}</time>
                <i className={styles[item.tone]} />
                <div>
                  <span>{item.label}</span>
                  <p>{item.detail}</p>
                </div>
                {index === evidence.length - 1 && <Sparkles size={16} />}
              </article>
            ))}
          </div>
          <div className={styles.topology}>
            <div className={styles.topologyTitle}><Network size={16} /> CMDB 影响拓扑</div>
            <div className={styles.nodes}>
              <span><Server size={15} /> API 网关</span><ChevronRight size={15} />
              <span className={styles.hotNode}><Activity size={15} /> checkout-api</span><ChevronRight size={15} />
              <span><Database size={15} /> payment-db</span>
            </div>
            <div className={styles.impact}>订单中心、会员权益、发票服务受到间接影响</div>
          </div>
        </div>

        <div className={styles.reasoningPanel}>
          <div className={styles.panelHead}>
            <div><span>02</span><h2>Agent 判断</h2></div>
            <div className={styles.agentBadge}><Bot size={15} /> SRE COPILOT</div>
          </div>
          <div className={styles.hypothesis}>
            <div className={styles.confidence}><strong>92</strong><span>%<br />置信度</span></div>
            <div>
              <span>最可能根因</span>
              <h3>支付数据库连接池未释放</h3>
              <p>最近发布将连接超时从 30s 调整为 120s；流量峰值下，大量异常请求持续占用连接。</p>
            </div>
          </div>
          <div className={styles.reasonList}>
            <div><Check size={15} /><span>日志特征与历史事件一致</span><b>+34%</b></div>
            <div><Check size={15} /><span>变更 CHG-0719-04 时间相关</span><b>+27%</b></div>
            <div><Check size={15} /><span>数据库 CPU 与 IO 正常</span><b>+18%</b></div>
            <div><Check size={15} /><span>连接使用率达到 99.6%</span><b>+13%</b></div>
          </div>
          <div className={styles.planLabel}><GitBranch size={15} /> 建议处置计划</div>
          <ol className={styles.plan}>
            <li><span>1</span><div><b>动态扩容连接池</b><small>从 200 调整至 320 · 可自动回滚</small></div></li>
            <li><span>2</span><div><b>滚动重启 2 个异常实例</b><small>每批 1 个 · 保持 80% 服务容量</small></div></li>
            <li><span>3</span><div><b>观察 5 分钟并验证</b><small>错误率 &lt; 1%，P95 &lt; 500ms</small></div></li>
          </ol>
        </div>

        <aside className={styles.controlPanel}>
          <div className={styles.panelHead}>
            <div><span>03</span><h2>执行控制</h2></div>
            <ShieldCheck size={19} />
          </div>
          <div className={`${styles.controlStatus} ${styles[actionState]}`}>
            <StatusIcon size={22} />
            <div><b>{status.label}</b><small>{status.detail}</small></div>
          </div>
          <div className={styles.guardrails}>
            {guardrails.map(([label, value]) => (
              <div key={label}><span>{label}</span><p>{value}</p></div>
            ))}
          </div>
          <div className={styles.command}>
            <span><TerminalSquare size={14} /> 待执行工具</span>
            <code>k8s.scale_connection_pool</code>
            <p>参数已脱敏 · 变更可逆 · 审计已开启</p>
          </div>

          {actionState === 'pending' && (
            <div className={styles.actions}>
              <button className={styles.reject}><X size={16} /> 驳回</button>
              <button className={styles.approve} onClick={approve}><UserCheck size={16} /> 批准一次执行</button>
            </div>
          )}
          {actionState === 'approved' && (
            <button className={styles.execute} onClick={execute}><Zap size={17} /> 执行处置计划</button>
          )}
          {actionState === 'executed' && (
            <button className={styles.traceButton} onClick={() => setShowTrace(true)}><ShieldCheck size={17} /> 查看审计证据</button>
          )}
          <div className={styles.policy}><LockKeyhole size={14} /> 策略 AGT-PROD-L3 已强制生效</div>
        </aside>
      </section>

      <section className={styles.outcomes}>
        <div><Clock3 size={18} /><span>预计恢复时间</span><strong>6 分钟</strong><small>较人工流程缩短 68%</small></div>
        <div><ShieldCheck size={18} /><span>受控执行率</span><strong>100%</strong><small>无越权、无静默操作</small></div>
        <div><Network size={18} /><span>已连接系统</span><strong>7</strong><small>监控、日志、CMDB、K8s、IM…</small></div>
        <div className={styles.thesis}><Sparkles size={18} /><span>产品价值</span><p>让 Agent 真正进入生产环境，同时保留企业需要的控制权。</p><ArrowRight size={18} /></div>
      </section>

      {showTrace && (
        <div className={styles.modalBackdrop} onClick={() => setShowTrace(false)}>
          <section className={styles.auditModal} onClick={(event) => event.stopPropagation()}>
            <div className={styles.auditHead}>
              <div><ShieldCheck size={22} /><span><b>执行证据包</b><small>AUD-20260719-140742</small></span></div>
              <button onClick={() => setShowTrace(false)} aria-label="关闭"><X size={18} /></button>
            </div>
            {[
              ['14:07:42.031', '审批签名', '刘洋 · MFA 已验证'],
              ['14:07:42.108', '授权签发', 'scope: checkout-api / action: scale'],
              ['14:07:43.921', '工具执行', 'connector: kubernetes-prod · success'],
              ['14:07:49.404', '状态验证', 'pool 320/320 · error rate 2.1% ↓'],
            ].map(([time, title, detail]) => (
              <div className={styles.auditRow} key={time}><time>{time}</time><i /><b>{title}</b><span>{detail}</span></div>
            ))}
            <div className={styles.auditFoot}><CheckCircle2 size={16} /> 证据链完整，哈希已写入租户审计日志</div>
          </section>
        </div>
      )}
    </main>
  );
}
