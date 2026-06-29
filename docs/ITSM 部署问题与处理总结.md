# ITSM 部署问题与处理总结

# ITSM 部署问题与处理总结

> 整理时间：2026\-06\-29
项目地址：https://github\.com/heidsoft/itsm（main 分支）
部署方式：Docker Compose 开发环境

---

## 一、问题总览

|\#|问题现象|类别|根因|状态|
|---|---|---|---|---|
|1|`unknown shorthand flag: 'f' in -f`|脚本兼容性|`docker compose` v2 命令在 v1 环境不存在|✅ 已修复|
|2|`dial tcp proxy.golang.org:443: i/o timeout`|网络/配置|国内服务器无法访问 Go 官方代理|✅ 已修复|
|3|`':ciTypeId' conflicts with existing wildcard ':id'`|代码 Bug|Gin 路由同级下 wildcard 参数名冲突|✅ 已修复|
|4|`KeyError: 'ContainerConfig'` \+ 容器名冲突|残留状态|历史容器未完全清理导致重建失败|✅ 已修复|

---

## 二、问题一：docker\-compose 命令不存在

### 错误信息

```
unknown shorthand flag: 'f' in -f
Usage: docker [OPTIONS] COMMAND [ARG...]
```

### 根因

系统安装的是 `docker-compose` v1\.29\.2（独立二进制），但 `scripts/lib/common.sh` 中的 `dc()` 函数调用的是 `docker compose`（Docker CLI 内置插件，v2 才支持）。

### 修复

**文件：** `scripts/lib/common.sh`

**修改内容：** `dc()` 函数中 `docker compose` → `docker-compose`

```Bash
# 修改前
dc() {
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        docker compose "$@"
    else
        docker compose "$@" 2>&1 | { ... }
    fi
}

# 修改后
dc() {
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        docker-compose "$@"
    else
        docker-compose "$@" 2>&1 | { ... }
    fi
}
```

### 判断方法

```Bash
# 检查 docker-compose 版本
docker-compose version

# 检查 docker compose 插件是否可用
docker compose version
```

---

## 三、问题二：Go 模块代理拉取超时

### 错误信息

```
go: ariga.io/atlas@v0.32.1-0.20250325101103-175b25e1c1b9:
Get "https://proxy.golang.org/ariga.io/atlas/@v/v0.32.1-0.20250325101103-175b25e1c1b9.mod":
dial tcp 142.250.73.145:443: i/o timeout
```

### 根因

容器构建时通过 `GOPROXY=https://proxy.golang.org,direct` 拉取 Go 模块，国内网络访问 `proxy.golang.org` 超时。

### 修复

**文件：** `docker-compose.dev.yml`

在 `itsm-init` 和 `itsm-backend` 服务的 build args 中添加国内镜像：

```YAML
services:
  itsm-init:
    build:
      args:
        - BUILDKIT_INLINE_CACHE=1
        - HTTP_PROXY=
        - HTTPS_PROXY=
        - http_proxy=
        - https_proxy=
        - GOPROXY=https://goproxy.cn,direct   # ← 新增

  itsm-backend:
    build:
      args:
        - BUILDKIT_INLINE_CACHE=1
        - HTTP_PROXY=
        - HTTPS_PROXY=
        - http_proxy=
        - https_proxy=
        - GOPROXY=https://goproxy.cn,direct   # ← 新增
```

---

## 四、问题三：Gin 路由 wildcard 参数冲突（代码 Bug）

### 错误信息

```
panic: ':ciTypeId' in new path '/api/v1/cmdb/ci-types/:ciTypeId/attributes'
conflicts with existing wildcard ':id' in existing prefix '/api/v1/cmdb/ci-types/:id'
```

### 根因

Gin 框架在同一级路由路径下，不能同时注册两个不同名字的 wildcard 参数。例如以下代码会冲突：

```Go
ciTypes.GET("/:id", ...)                    // 先注册了 :id
ciTypes.GET("/:ciTypeId/attributes", ...)   // 再注册 :ciTypeId → 冲突
```

### 修复涉及的文件

|文件|修改内容|
|---|---|
|`itsm-backend/router/router.go`|`/:ciTypeId/attributes` → `/:id/attributes`|
|`itsm-backend/router/cmdb_routes.go`|同上|
|`itsm-backend/controller/cmdb_controller.go`|`ctx.Param("ciTypeId")` → `ctx.Param("id")`|

### 具体修改

**1\. router/router\.go**

```Go
// 修改前
ciTypes.GET("/:ciTypeId/attributes", config.CMDBController.ListCIAttributeDefinitions)

// 修改后
ciTypes.GET("/:id/attributes", config.CMDBController.ListCIAttributeDefinitions)
```

**2\. router/cmdb\_routes\.go**

```Go
// 修改前
cis.GET("/:ciTypeId/relationships", cmdbController.ListCIRelationshipsByCIID)
cis.GET("/:ciTypeId/impact-analysis", cmdbController.GetCIImpactAnalysis)

// 修改后
cis.GET("/:id/relationships", cmdbController.ListCIRelationshipsByCIID)
cis.GET("/:id/impact-analysis", cmdbController.GetCIImpactAnalysis)
```

**3\. controller/cmdb\_controller\.go**

```Go
// 修改前
ciTypeIDStr := ctx.Param("ciTypeId")
ciTypeID, err := strconv.Atoi(ciTypeIDStr)

// 修改后
ciTypeIDStr := ctx.Param("id")
ciTypeID, err := strconv.Atoi(ciTypeIDStr)
```

### 受影响的路由汇总

|原路由路径|修改后路径|HTTP 方法|
|---|---|---|
|`/api/v1/cmdb/ci-types/:ciTypeId/attributes`|`/api/v1/cmdb/ci-types/:id/attributes`|GET|
|`/api/v1/cmdb/cis/:ciId/relationships`|`/api/v1/cmdb/cis/:id/relationships`|GET|
|`/api/v1/cmdb/cis/:ciId/impact-analysis`|`/api/v1/cmdb/cis/:id/impact-analysis`|GET|

---

## 五、问题四：Docker 残留容器干扰

### 错误信息

```
KeyError: 'ContainerConfig'
Conflict. The container name "/itsm-init-dev" is already in use
```

### 根因

多次构建过程中，历史容器实例未完全清理。docker\-compose 尝试重建容器时，读取旧容器镜像配置的 `ContainerConfig` 字段（新版 API 已移除）失败，导致报错。

### 修复

手动清理残留容器：

```Bash
# 查看残留容器
docker ps -a | grep itsm

# 强制删除
docker rm -f itsm-init-dev

# 重新启动
docker-compose -f docker-compose.dev.yml --profile dev up -d --build
```

---

## 六、验证结果

启动命令：

```Bash
cd /service/home/openclaw/itsm
docker-compose -f docker-compose.dev.yml --profile dev up -d --build
```

服务状态：

|服务|状态|端口|
|---|---|---|
|itsm\-backend\-dev|✅ Up \(healthy\)|8090|
|itsm\-frontend\-dev|✅ Up|3000|
|itsm\-postgres\-dev|✅ Up \(healthy\)|5432|
|itsm\-redis\-dev|✅ Up \(healthy\)|6379|
|itsm\-minio\-dev|✅ Up \(healthy\)|9000/9001|

健康检查：

```Bash
curl http://localhost:8090/api/v1/health
# {"status":"ok","timestamp":"..."}
```

---

## 七、处理方法总结

**核心原则：先定位，后修复，每步验证**

1. **读报错信息** — 从错误输出提取关键词定位根因

2. **定位文件** — 错误指向哪里就查哪里，不猜测

3. **小范围验证** — 修改前确认上下文范围，避免改错

4. **精准修改** — 用精确替换，不动无关代码

5. **验证结果** — 重新构建后用 `docker ps` \+ health endpoint 确认服务在线

---

## 八、后续建议

1. **路由规范**：后续开发中避免在同级路由下混用不同名字的 wildcard 参数，统一使用 `:id`

2. **构建代理**：将 `GOPROXY=https://goproxy.cn,direct` 作为默认 build arg 写入 Dockerfile，避免每次部署都要修改 docker\-compose

3. **清理脚本**：在 `deploy-dev.sh` 的 `reset` 命令中增加强制清理残留容器的逻辑

