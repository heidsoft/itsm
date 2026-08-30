# ent 代码生成操作手册

**实测数据（2026-08-30，macOS 16GB）**

| 项 | 数值 |
|---|---|
| schema 数量 | 132 |
| 生成代码规模 | 71.7 万行 |
| **峰值内存** | **4.1 GB** |
| **耗时** | **4 分 40 秒** |
| 结果 | exit=0，schema 未改时 `ent/` 零 diff（生成是确定性的） |

---

## 一、标准命令

### 方式 A：预编译二进制（推荐，避免每次重编译 entc）

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend

# 1. 构建 entc（只需做一次，约 20 秒，产物 22MB）
go build -o /tmp/entc entgo.io/ent/cmd/ent

# 2. 生成代码（必须在 ent/ 目录下执行）
cd ent
/tmp/entc generate ./schema
```

### 方式 B：项目标准入口

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
go generate ./ent
```

等价于 `ent/generate.go` 里的 `go run -mod=mod entgo.io/ent/cmd/ent generate ./schema`。
比方式 A 慢一些（每次都要编译 entc），但最贴近项目约定。

### 方式 C：后台运行（强烈推荐，可规避超时）

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend/ent
( /usr/bin/time -l /tmp/entc generate ./schema ) > /tmp/entc.log 2>&1
```

跑完看结果：

```bash
grep -E "maximum resident|peak memory" /tmp/entc.log
```

---

## 二、三条硬性前提（血泪教训）

### 1. 需要约 6GB 可用内存

实测峰值 4.1GB，留 2GB 余量。**可用内存不足是失败的唯一真实原因**。

跑之前先确认：

```bash
vm_stat | awk '/Pages free/{printf "free: %.0f MB\n",$3*4096/1048576} /Pages inactive/{printf "inactive: %.0f MB\n",$3*4096/1048576}'
```

`free + inactive` 应大于 6000 MB。不够就先关掉 Docker Desktop、IDE、浏览器等。

> Docker 容器路线不通：Docker 配额只有 5.8GB，扣掉自身开销后低于 entc 峰值需求。

### 2. 绝对不要用管道

```bash
# 错误！管道 + 超时会杀掉整个进程组，输出全空、报 137，极易误判为 OOM
/tmp/entc generate ./schema | tail -20
```

曾因此连续失败十几次，`/usr/bin/time` 的输出、stderr 全部丢失，看起来"毫无征兆地被杀"。
要捕获输出就重定向到**文件**，不要用管道。

### 3. 不要设大 GOMEMLIMIT

```bash
# 错误！GOMEMLIMIT=14GiB 会让 Go 以为内存充足而放任增长，反而更早被 kill
GOMEMLIMIT=14GiB /tmp/entc generate ./schema
```

**不设任何 GOGC/GOMEMLIMIT 反而成功了**。Go 1.19+ 默认的软性内存限制已经比较合理。
若一定要设，往小设（如 2GiB）用它换取更激进的 GC，不要往大设。

---

## 三、验证生成正确

```bash
cd /Users/heidsoft/Downloads/research/itsm

# 1. schema 未改动时，ent/ 应当零 diff（证明生成是确定性的、可复现的）
git status --porcelain -- itsm-backend/ent | wc -l    # 应为 0

# 2. 编译
cd itsm-backend && go build ./...

# 3. 跑测试
go test ./service/... ./handlers/... -count=1
```

---

## 四、改过 schema 之后的完整流程

```bash
# 1. 改 schema（例如给 knowledgearticle 加字段）
vim itsm-backend/ent/schema/knowledgearticle.go

# 2. 生成
cd itsm-backend/ent && ( /usr/bin/time -l /tmp/entc generate ./schema ) > /tmp/entc.log 2>&1

# 3. 确认编译
cd .. && go build ./...

# 4. 跑测试（重点看 service/ 与 handlers/）
go test ./service/... ./handlers/... -count=1

# 5. 检查 diff 范围是否合理（不应有大量无关文件变动）
cd .. && git status --porcelain -- itsm-backend/ent
```

**生产环境别忘了**：镜像上线前必须跑 `ent migrate`（见 AGENTS.md）。

---

## 五、故障排查

| 现象 | 原因 | 处理 |
|---|---|---|
| exit 137，无任何输出 | 可用内存不足，或用了管道 | 检查可用内存；改用重定向到文件 |
| exit 137，有部分输出 | 真 OOM | 关掉占内存的程序后重试；或换更大内存机器 |
| `ent/` 出现大量语法错误 | entc 被 kill 留下残片 | `git checkout -- itsm-backend/ent` 恢复后重跑 |
| 生成后编译报错 `undefined: xxx.FieldYyy` | schema 与生成代码不一致（生成中途被 kill） | 同上，恢复后重跑 |
| 卡住不动 | 正常，需要 4~5 分钟 | 耐心等；用后台方式跑 |

---

## 六、历史背景

此前长期认为「entc 在本机必然 OOM」，原因是：

1. 跑的时候可用内存只剩 590MB~1.2GB（Docker + 多个 worktree + IDE 占满）
2. 习惯性加管道（`| tail`），导致进程组被杀且输出丢失，无法定位原因
3. 错误地设置 `GOMEMLIMIT=14GiB`，适得其反

2026-08-30 后台运行（无管道、无 GOMEMLIMIT）实测成功，峰值 4.1GB / 4m40s，
且生成结果与 HEAD 逐字节一致。**该结论已修正。**

在此之前由于无法生成代码，项目采用了若干绕行方案（均无需改 schema），这些方案仍然有效：

- 工单 ↔ CI 绑定复用 schema 里已存在的 `edge.To("tickets")`
- 知识分类权限复用 `permission` 表的自由字符串 resource/action

后续若要加 L1 时效治理（`valid_until`）、L2 权威消歧（`authority_level`）等字段，
现在可以直接走正常的代码生成流程了。
