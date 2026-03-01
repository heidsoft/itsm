# SEO 提交流程

**执行时间**: 2026-03-01  
**状态**: 准备执行

---

## 🔍 搜索引擎提交

### 1. Google Search Console

**提交地址**: https://search.google.com/search-console

**步骤**:
1. 登录 Google 账号
2. 点击"添加属性"
3. 输入域名：cloudmesh.top
4. 选择验证方式（推荐 DNS 验证）
5. 添加 TXT 记录到 DNS
6. 等待验证通过（5-10 分钟）
7. 提交网站地图：https://cloudmesh.top/sitemap.xml
8. 监控索引状态

**预期时间线**:
- 验证：5-10 分钟
- 开始索引：1-3 天
- 完整索引：1-2 周

---

### 2. 百度搜索资源平台

**提交地址**: https://ziyuan.baidu.com/

**步骤**:
1. 注册百度账号
2. 登录百度站长平台
3. 添加网站：cloudmesh.top
4. 验证所有权（文件验证或 DNS 验证）
5. 提交网站地图
6. 使用 API 主动推送页面

**主动推送代码**:
```bash
curl -H 'Content-Type:text/plain' \
--data-binary @urls.txt \
"http://data.zz.baidu.com/urls?site=cloudmesh.top&token=YOUR_TOKEN"
```

**预期时间线**:
- 验证：10-30 分钟
- 开始收录：1-3 天
- 完整收录：1-2 周

---

### 3. Bing Webmaster Tools

**提交地址**: https://www.bing.com/webmasters

**步骤**:
1. 登录 Microsoft 账号
2. 添加网站
3. 验证所有权
4. 提交网站地图

---

## 📝 提交后监控

### 每日检查
- [ ] 索引页面数
- [ ] 抓取错误
- [ ] 搜索展示次数
- [ ] 点击率

### 每周报告
- [ ] 关键词排名变化
- [ ] 自然流量趋势
- [ ] 外部链接增长
- [ ] 移动端可用性

---

**SEO 提交流程已准备就绪！**
