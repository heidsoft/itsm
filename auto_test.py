#!/usr/bin/env python3
"""
ITSM系统自动化测试脚本
用法: python auto_test.py
依赖: pip install playwright requests
"""
import asyncio
import requests
from playwright.async_api import async_playwright

async def main():
    results = []

    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=False)
        page = await browser.new_page()

        # 1. 登录测试
        print("\n=== 测试: 用户登录 ===")
        await page.goto('http://localhost:3000/login')
        await page.wait_for_load_state('networkidle')
        await page.wait_for_timeout(2000)

        await page.fill('input[name="username"]', 'admin')
        await page.fill('input[type="password"]', 'admin123')
        await page.click('button[type="submit"]')
        await page.wait_for_timeout(5000)

        token = await page.evaluate('localStorage.getItem("access_token")')
        login_ok = bool(token)
        print(f"登录: {'✓' if login_ok else '✗'}")
        results.append(("登录", login_ok))

        if not login_ok:
            print("登录失败，停止测试")
            return

        # 2. 测试各页面
        pages = [
            ('/dashboard', '仪表盘'),
            ('/tickets', '工单管理'),
            ('/incidents', '事件管理'),
            ('/problems', '问题管理'),
            ('/changes', '变更管理'),
            ('/knowledge', '知识库'),
            ('/service-catalog', '服务目录'),
            ('/sla', 'SLA管理'),
            ('/assets', '资产管理'),
            ('/cmdb', 'CMDB'),
            ('/workflow', '工作流'),
            ('/reports', '报表'),
            ('/admin', '系统管理'),
        ]

        print("\n=== 页面功能测试 ===")
        for path, name in pages:
            await page.goto(f'http://localhost:3000{path}')
            await page.wait_for_timeout(3000)
            text = await page.evaluate('document.body.innerText')
            ok = len(text) > 200 and '404' not in text
            print(f"{name}: {'✓' if ok else '✗'}")
            results.append((name, ok))

        # 3. API测试
        print("\n=== API集成测试 ===")
        resp = requests.post('http://localhost:8090/api/v1/auth/login',
                           json={'username': 'admin', 'password': 'admin123'}, timeout=5)
        api_ok = resp.status_code == 200
        print(f"登录API: {'✓' if api_ok else '✗'}")

        if api_ok:
            token = resp.json()['data']['access_token']
            headers = {'Authorization': f'Bearer {token}'}
            for path in ['/api/v1/tickets', '/api/v1/incidents', '/api/v1/sla/definitions']:
                r = requests.get(f'http://localhost:8090{path}', headers=headers, timeout=3)
                print(f"{path}: {'✓' if r.status_code == 200 else '✗'}")

        # 4. 结果汇总
        print("\n=== 测试结果 ===")
        passed = sum(1 for _, ok in results if ok)
        total = len(results)
        print(f"总计: {passed}/{total} 通过")

        await page.wait_for_timeout(30000)
        await browser.close()

if __name__ == '__main__':
    asyncio.run(main())
