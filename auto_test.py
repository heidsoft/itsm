#!/usr/bin/env python3
"""
ITSM系统自动化测试脚本 - 完整操作测试版
用法: python auto_test.py
依赖: pip install playwright requests
"""
import asyncio
import requests
import random
import string
from datetime import datetime
from playwright.async_api import async_playwright, expect

BASE_URL = "http://localhost:3000"
API_URL = "http://localhost:8090/api/v1"

# 测试配置
TEST_USER = {
    "username": "admin",
    "password": "admin123"
}

def random_string(length=8):
    """生成随机字符串"""
    return ''.join(random.choices(string.ascii_lowercase + string.digits, k=length))

def log_test(name, passed, details=""):
    """打印测试结果"""
    status = "✓" if passed else "✗"
    detail_str = f" - {details}" if details else ""
    print(f"  {status} {name}{detail_str}")
    return passed

async def test_login(page) -> bool:
    """测试登录功能"""
    print("\n=== 测试: 用户登录 ===")

    await page.goto(f'{BASE_URL}/login')
    await page.wait_for_load_state('networkidle')

    # 等待表单加载
    await page.wait_for_selector('.ant-input, .ant-input-password', timeout=10000)
    await page.wait_for_timeout(1000)

    # 打印页面标题用于调试
    title = await page.title()
    print(f"  页面标题: {title}")

    # 查找用户名输入框 - 尝试多种选择器
    username_input = None
    for selector in [
        'input[name="username"]',
        'input#username',
        'input[placeholder*="用户"]',
        '.ant-input:has(~ input)',
        'input.ant-input'
    ]:
        try:
            username_input = await page.query_selector(selector)
            if username_input:
                print(f"  找到用户名输入框: {selector}")
                break
        except:
            pass

    if username_input:
        await username_input.fill(TEST_USER["username"])
    else:
        # 使用更通用的方法：获取所有input并填充第一个
        inputs = await page.query_selector_all('input.ant-input')
        if len(inputs) >= 1:
            await inputs[0].fill(TEST_USER["username"])
            print("  使用备用方法填写用户名")
        else:
            print("  ✗ 未找到用户名输入框")
            return False

    # 查找密码输入框
    password_input = None
    for selector in [
        'input[name="password"]',
        'input#password',
        'input.ant-input-password-input',
        'input[type="password"]'
    ]:
        try:
            password_input = await page.query_selector(selector)
            if password_input:
                print(f"  找到密码输入框: {selector}")
                break
        except:
            pass

    if password_input:
        await password_input.fill(TEST_USER["password"])
    else:
        # 使用备用方法
        inputs = await page.query_selector_all('input')
        if len(inputs) >= 2:
            await inputs[1].fill(TEST_USER["password"])
            print("  使用备用方法填写密码")
        else:
            print("  ✗ 未找到密码输入框")
            return False

    # 点击登录按钮 - 尝试多种选择器
    submit_clicked = False
    for selector in [
        'button[type="submit"]',
        'button.ant-btn-primary',
        'button:has-text("登录")',
        'button:has-text("登 录")'
    ]:
        try:
            submit_btn = await page.query_selector(selector)
            if submit_btn:
                await submit_btn.click()
                submit_clicked = True
                print(f"  点击登录按钮: {selector}")
                break
        except:
            pass

    if not submit_clicked:
        # 最后一个备用方法
        buttons = await page.query_selector_all('button')
        for btn in buttons:
            text = await btn.text_content()
            if text and '登录' in text:
                await btn.click()
                submit_clicked = True
                break

    if not submit_clicked:
        print("  ✗ 未找到登录按钮")
        return False

    await page.wait_for_timeout(5000)

    # 检查是否登录成功
    token = await page.evaluate('localStorage.getItem("access_token")')
    if token:
        print(f"  ✓ 登录成功 - Token已保存")
        return True
    else:
        # 尝试检查是否有错误提示
        error_text = await page.text_content('.ant-message-error, .ant-form-item-explain-error, [class*="error"]').catch(lambda: None)
        print(f"  ✗ 登录失败 {error_text or ''}")
        return False

async def test_dashboard(page) -> bool:
    """测试仪表盘页面"""
    print("\n=== 测试: 仪表盘 (/dashboard) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/dashboard')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    # 检查页面内容
    text = await page.evaluate('document.body.innerText')

    # 检查关键元素
    checks = [
        ("统计数据区域", len(text) > 200 and ('统计' in text or 'Dashboard' in text or '总' in text or '工单' in text)),
        ("页面加载", '404' not in text and 'Not Found' not in text),
    ]

    for name, result in checks:
        all_passed &= log_test(name, result)

    return all_passed

async def test_tickets(page) -> bool:
    """测试工单管理功能"""
    print("\n=== 测试: 工单管理 (/tickets) ===")
    all_passed = True

    # 1. 访问工单列表页
    await page.goto(f'{BASE_URL}/tickets')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("工单列表加载", list_ok)

    # 2. 点击新建工单按钮
    print("  尝试点击新建工单按钮...")
    # 尝试多种选择器
    create_clicked = False

    # 方法1: 使用包含"新建"或"创建"的按钮
    create_buttons = await page.query_selector_all('button')
    for btn in create_buttons:
        btn_text = await btn.text_content()
        if btn_text and ('新建' in btn_text or '创建' in btn_text or '创建工单' in btn_text):
            await btn.click()
            create_clicked = True
            log_test("点击新建工单按钮", True)
            break

    # 方法2: 使用PlusOutlined图标按钮
    if not create_clicked:
        plus_btn = await page.query_selector('button:has(svg.lucide-plus), button:has(svg.anticon-plus)')
        if plus_btn:
            await plus_btn.click()
            create_clicked = True
            log_test("点击新建工单按钮(icon)", True)

    # 3. 如果成功点击，进入创建页面
    if create_clicked:
        await page.wait_for_timeout(2000)
        current_url = page.url
        if '/tickets/create' in current_url or '/create' in current_url:
            all_passed &= log_test("跳转到创建页面", True)

            # 4. 填写工单表单
            await page.wait_for_timeout(1000)

            # 填写标题
            title_input = await page.query_selector('input[id*="title"], input[name="title"], input[aria-label*="标题"]')
            if title_input:
                test_title = f"自动化测试工单-{random_string()}"
                await title_input.fill(test_title)
                log_test("填写工单标题", True, test_title[:20])
                all_passed &= True
            else:
                log_test("填写工单标题", False, "未找到标题输入框")

            # 填写描述
            desc_input = await page.query_selector('textarea[id*="description"], textarea[name="description"]')
            if desc_input:
                test_desc = f"这是自动化测试工单的详细描述，用于测试工单创建功能。描述内容至少需要10个字符。"
                await desc_input.fill(test_desc)
                log_test("填写工单描述", True, test_desc[:20])
                all_passed &= True
            else:
                log_test("填写工单描述", False, "未找到描述输入框")

            # 选择优先级 (如果有)
            priority_select = await page.query_selector('div[id*="priority"], div[name="priority"], .ant-select:has-text("优先级")')
            if priority_select:
                await priority_select.click()
                await page.wait_for_timeout(500)
                # 选择第一个选项
                menu_item = await page.query_selector('.ant-select-dropdown .ant-select-item')
                if menu_item:
                    await menu_item.click()
                    log_test("选择优先级", True)

            # 5. 提交工单
            submit_buttons = await page.query_selector_all('button')
            submit_clicked = False
            for btn in submit_buttons:
                btn_text = await btn.text_content()
                if btn_text and ('提交' in btn_text or '创建工单' in btn_text or '保存' in btn_text):
                    await btn.click()
                    submit_clicked = True
                    log_test("点击提交按钮", True)
                    break

            if submit_clicked:
                await page.wait_for_timeout(3000)
                # 检查是否创建成功（跳转到详情页或显示成功消息）
                success_text = await page.evaluate('document.body.innerText')
                if '成功' in success_text or 'success' in success_text.lower() or '/tickets/' in page.url:
                    all_passed &= log_test("工单创建成功", True)
                else:
                    all_passed &= log_test("工单创建成功", False, "未检测到成功提示")
            else:
                log_test("点击提交按钮", False, "未找到提交按钮")
        else:
            all_passed &= log_test("跳转到创建页面", False, f"URL: {current_url}")
    else:
        all_passed &= log_test("点击新建工单按钮", False, "未找到按钮")

    return all_passed

async def test_incidents(page) -> bool:
    """测试事件管理功能"""
    print("\n=== 测试: 事件管理 (/incidents) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/incidents')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("事件列表加载", list_ok)

    # 尝试点击新建事件
    create_buttons = await page.query_selector_all('button')
    for btn in create_buttons:
        btn_text = await btn.text_content()
        if btn_text and ('新建' in btn_text or '创建' in btn_text):
            await btn.click()
            await page.wait_for_timeout(2000)
            log_test("点击新建事件按钮", True)
            break

    return all_passed

async def test_problems(page) -> bool:
    """测试问题管理功能"""
    print("\n=== 测试: 问题管理 (/problems) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/problems')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("问题列表加载", list_ok)

    return all_passed

async def test_changes(page) -> bool:
    """测试变更管理功能"""
    print("\n=== 测试: 变更管理 (/changes) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/changes')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("变更列表加载", list_ok)

    return all_passed

async def test_knowledge(page) -> bool:
    """测试知识库功能"""
    print("\n=== 测试: 知识库 (/knowledge) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/knowledge')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("知识库列表加载", list_ok)

    # 尝试搜索功能
    search_input = await page.query_selector('input[placeholder*="搜索"], input[aria-label*="搜索"]')
    if search_input:
        await search_input.fill("test")
        await page.wait_for_timeout(1000)
        log_test("搜索功能", True)

    # 尝试新建文章
    create_buttons = await page.query_selector_all('button')
    for btn in create_buttons:
        btn_text = await btn.text_content()
        if btn_text and ('新建' in btn_text or '创建' in btn_text):
            await btn.click()
            await page.wait_for_timeout(2000)
            log_test("点击新建文章按钮", True)
            break

    return all_passed

async def test_service_catalog(page) -> bool:
    """测试服务目录功能"""
    print("\n=== 测试: 服务目录 (/service-catalog) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/service-catalog')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("服务目录加载", list_ok)

    # 尝试查看服务详情
    service_cards = await page.query_selector_all('[class*="card"], [class*="Card"], .ant-card')
    if len(service_cards) > 1:  # 跳过页面头部卡片
        log_test("服务列表渲染", True, f"{len(service_cards)} 个卡片")

    return all_passed

async def test_sla(page) -> bool:
    """测试SLA管理功能"""
    print("\n=== 测试: SLA管理 (/sla) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/sla')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("SLA页面加载", list_ok)

    return all_passed

async def test_assets(page) -> bool:
    """测试资产管理功能"""
    print("\n=== 测试: 资产管理 (/assets) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/assets')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("资产列表加载", list_ok)

    return all_passed

async def test_cmdb(page) -> bool:
    """测试CMDB功能"""
    print("\n=== 测试: CMDB (/cmdb) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/cmdb')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("CMDB页面加载", list_ok)

    return all_passed

async def test_workflow(page) -> bool:
    """测试工作流功能"""
    print("\n=== 测试: 工作流 (/workflow) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/workflow')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("工作流列表加载", list_ok)

    return all_passed

async def test_my_requests(page) -> bool:
    """测试我的请求功能"""
    print("\n=== 测试: 我的请求 (/my-requests) ===")
    all_passed = True

    await page.goto(f'{BASE_URL}/my-requests')
    await page.wait_for_load_state('networkidle')
    await page.wait_for_timeout(3000)

    text = await page.evaluate('document.body.innerText')
    list_ok = '404' not in text
    all_passed &= log_test("我的请求页面加载", list_ok)

    return all_passed

async def test_api_integration() -> bool:
    """测试后端API集成"""
    print("\n=== API集成测试 ===")
    all_passed = True

    try:
        # 登录API
        resp = requests.post(f'{API_URL}/auth/login',
                           json=TEST_USER, timeout=5)
        api_ok = resp.status_code == 200
        all_passed &= log_test("登录API (/auth/login)", api_ok, f"状态码: {resp.status_code}")

        if api_ok and 'data' in resp.json():
            token = resp.json()['data'].get('access_token', '')
            headers = {'Authorization': f'Bearer {token}'}

            # 测试各个API端点
            endpoints = [
                ('/tickets', '工单列表'),
                ('/incidents', '事件列表'),
                ('/problems', '问题列表'),
                ('/changes', '变更列表'),
                ('/sla/definitions', 'SLA定义'),
                ('/knowledge/articles', '知识库文章'),
            ]

            for path, name in endpoints:
                try:
                    r = requests.get(f'{API_URL}{path}', headers=headers, timeout=3)
                    all_passed &= log_test(f"{name} API ({path})", r.status_code == 200, f"状态码: {r.status_code}")
                except Exception as e:
                    log_test(f"{name} API ({path})", False, str(e)[:30])
        else:
            log_test("获取Token", False, resp.text[:50])

    except Exception as e:
        log_test("API测试", False, str(e)[:50])

    return all_passed

async def main():
    """主测试函数"""
    results = []
    start_time = datetime.now()

    print("=" * 60)
    print("ITSM系统 - 完整自动化测试")
    print(f"开始时间: {start_time.strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)

    async with async_playwright() as p:
        # 启动浏览器 (headless=False 以便观察测试过程)
        browser = await p.chromium.launch(headless=False, slow_mo=100)
        context = await browser.new_context(
            viewport={'width': 1280, 'height': 720},
            locale='zh-CN'
        )
        page = await context.new_page()

        try:
            # 1. 登录测试
            login_ok = await test_login(page)
            results.append(("用户登录", login_ok))

            if not login_ok:
                print("\n登录失败，停止后续测试")
                # 仍然尝试API测试
                await test_api_integration()
                return

            # 2. 仪表盘测试
            results.append(("仪表盘", await test_dashboard(page)))

            # 3. 工单管理测试
            results.append(("工单管理", await test_tickets(page)))

            # 4. 事件管理测试
            results.append(("事件管理", await test_incidents(page)))

            # 5. 问题管理测试
            results.append(("问题管理", await test_problems(page)))

            # 6. 变更管理测试
            results.append(("变更管理", await test_changes(page)))

            # 7. 知识库测试
            results.append(("知识库", await test_knowledge(page)))

            # 8. 服务目录测试
            results.append(("服务目录", await test_service_catalog(page)))

            # 9. SLA管理测试
            results.append(("SLA管理", await test_sla(page)))

            # 10. 资产管理测试
            results.append(("资产管理", await test_assets(page)))

            # 11. CMDB测试
            results.append(("CMDB", await test_cmdb(page)))

            # 12. 工作流测试
            results.append(("工作流", await test_workflow(page)))

            # 13. 我的请求测试
            results.append(("我的请求", await test_my_requests(page)))

            # 14. API集成测试
            results.append(("API集成", await test_api_integration()))

        except Exception as e:
            print(f"\n测试过程发生错误: {e}")
            import traceback
            traceback.print_exc()
        finally:
            # 等待用户查看结果
            await page.wait_for_timeout(5000)
            await browser.close()

    # 结果汇总
    end_time = datetime.now()
    duration = (end_time - start_time).total_seconds()

    print("\n" + "=" * 60)
    print("测试结果汇总")
    print("=" * 60)

    passed = sum(1 for _, ok in results if ok)
    total = len(results)

    for name, ok in results:
        status = "✓" if ok else "✗"
        print(f"  {status} {name}")

    print("-" * 60)
    print(f"总计: {passed}/{total} 通过")
    print(f"耗时: {duration:.1f} 秒")
    print("=" * 60)

    if passed == total:
        print("\n🎉 所有测试通过!")
    elif passed > total * 0.7:
        print("\n✓ 大部分测试通过")
    else:
        print("\n✗ 较多测试失败，请检查")

if __name__ == '__main__':
    asyncio.run(main())
