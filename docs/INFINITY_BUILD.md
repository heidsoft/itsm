# Infinity Docker 构建笔记

> 日期: 2026-04-01

## 概述

Infinity 是一个 Go 语言开发的向量数据库服务，用于 AI 语义搜索场景。

## 容器信息

| 属性 | 值 |
|------|-----|
| 镜像名 | `my-infinity:custom` |
| 容器名 | `my-infinity` |
| 状态 | 运行中 |
| 运行时长 | 10+ 小时 |

## 端口映射

| 容器端口 | 主机端口 | 协议 |
|----------|----------|------|
| 23817 | 23817 | TCP |
| 23820 | 23820 | TCP |
| 23850 | 23850 | TCP |

## 环境变量

```
HTTP_PROXY=http://127.0.0.1:33210
http_proxy=http://127.0.0.1:33210
HTTPS_PROXY=http://127.0.0.1:33210
https_proxy=http://127.0.0.1:33210
NO_PROXY=localhost,127.0.0.1,10.96.0.0/12,192.168.59.0/24,192.168.49.0/24,192.168.39.0/24
no_proxy=localhost,127.0.0.1,10.96.0.0/12,192.168.59.0/24,192.168.49.0/24,192.168.39.0/24
```

## 基础镜像

- Ubuntu 22.04
- Entrypoint: `/usr/bin/infinity`

## 常用命令

```bash
# 查看容器状态
docker ps | grep infinity

# 查看容器日志
docker logs my-infinity

# 重启容器
docker restart my-infinity

# 停止容器
docker stop my-infinity

# 删除容器（如需重新构建）
docker rm my-infinity
```

## 构建注意事项

### 容器名冲突

构建时如遇到以下错误：
```
docker: Error response from daemon: Conflict. The container name "/my-infinity" is already in use
```

解决方法：
1. 先停止并删除现有容器: `docker rm -f my-infinity`
2. 或者使用不同的容器名: `docker run --name my-infinity-new ...`

### 端口占用

确保以下端口未被占用：
- 23817
- 23820
- 23850

```bash
# 检查端口占用
lsof -i :23817
lsof -i :23820
lsof -i :23850
```

## 测试连接

```bash
# 检查服务是否可访问
curl http://localhost:23817/health
curl http://localhost:23820/health
```