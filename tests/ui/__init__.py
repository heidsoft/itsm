# -*- coding: utf-8 -*-
"""
UI自动化测试工具
UI Automation Test Utilities
"""

import os
import time
from pathlib import Path
from typing import Optional, Dict, Any
from datetime import datetime

from playwright.sync_api import sync_playwright, Browser, BrowserContext, Page, expect

from .. import get_config, get_logger


class UITestClient:
    """UI自动化测试客户端"""

    def __init__(self, browser_type: str = 'chromium'):
        self.config = get_config()
        self.logger = get_logger('ui_test')
        self.browser_type = browser_type
        self.browser: Optional[Browser] = None
        self.context: Optional[BrowserContext] = None
        self.page: Optional[Page] = None
        self._playwright = None
        self.screenshots_dir = Path('./tests/reports/screenshots')
        self.screenshots_dir.mkdir(parents=True, exist_ok=True)

    def start(self):
        """启动浏览器"""
        self._playwright = sync_playwright().start()

        # 获取浏览器
        if self.browser_type == 'chromium':
            self.browser = self._playwright.chromium.launch(
                headless=self.config.get_bool('ui', 'headless', True),
                args=['--no-sandbox', '--disable-dev-shm-usage']
            )
        elif self.browser_type == 'firefox':
            self.browser = self._playwright.firefox.launch(headless=True)
        elif self.browser_type == 'webkit':
            self.browser = self._playwright.webkit.launch(headless=True)
        else:
            self.browser = self._playwright.chromium.launch(headless=True)

        # 创建上下文
        self.context = self.browser.new_context(
            viewport={'width': 1920, 'height': 1080},
            ignore_https_errors=True
        )

        # 创建页面
        self.page = self.context.new_page()
        self.logger.info(f"Browser started: {self.browser_type}")

    def close(self):
        """关闭浏览器"""
        if self.page:
            self.page.close()
        if self.context:
            self.context.close()
        if self.browser:
            self.browser.close()
        if self._playwright:
            self._playwright.stop()
        self.logger.info("Browser closed")

    def navigate_to(self, url: str):
        """导航到URL"""
        base_url = self.config.get('ui', 'base_url', 'http://localhost:3000')
        if not url.startswith('http'):
            url = f"{base_url.rstrip('/')}/{url.lstrip('/')}"

        self.page.goto(url, wait_until='networkidle', timeout=30000)
        self.logger.info(f"Navigated to: {url}")

    def take_screenshot(self, name: str = None) -> str:
        """截图"""
        if name is None:
            name = f"screenshot_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        # 确保目录存在
        self.screenshots_dir.mkdir(parents=True, exist_ok=True)

        filepath = self.screenshots_dir / f"{name}.png"
        self.page.screenshot(path=str(filepath), full_page=True)
        self.logger.info(f"Screenshot saved: {filepath}")

        return str(filepath)

    def take_failure_screenshot(self, test_name: str) -> str:
        """失败时截图"""
        return self.take_screenshot(f"failure_{test_name}")

    # ==================== 元素操作 ====================

    def click(self, selector: str, timeout: int = 30000):
        """点击元素"""
        self.page.click(selector, timeout=timeout)

    def fill(self, selector: str, value: str):
        """填充输入框"""
        self.page.fill(selector, value)

    def type_slowly(self, selector: str, value: str, delay: int = 50):
        """慢速输入"""
        self.page.type(selector, value, delay=delay)

    def select_option(self, selector: str, value: str):
        """选择下拉选项"""
        self.page.select_option(selector, value)

    def check(self, selector: str):
        """勾选复选框"""
        self.page.check(selector)

    def uncheck(self, selector: str):
        """取消勾选"""
        self.page.unchecked(selector)

    def hover(self, selector: str):
        """悬停"""
        self.page.hover(selector)

    # ==================== 元素获取 ====================

    def get_text(self, selector: str) -> str:
        """获取元素文本"""
        return self.page.text_content(selector)

    def get_attribute(self, selector: str, attribute: str) -> str:
        """获取元素属性"""
        return self.page.get_attribute(selector, attribute)

    def is_visible(self, selector: str) -> bool:
        """元素是否可见"""
        return self.page.is_visible(selector)

    def is_enabled(self, selector: str) -> bool:
        """元素是否可用"""
        return self.page.is_enabled(selector)

    def count_elements(self, selector: str) -> int:
        """获取元素数量"""
        return len(self.page.query_selector_all(selector))

    # ==================== 等待操作 ====================

    def wait_for_selector(self, selector: str, timeout: int = 30000):
        """等待元素出现"""
        self.page.wait_for_selector(selector, timeout=timeout)

    def wait_for_load_state(self, state: str = 'networkidle'):
        """等待页面加载完成"""
        self.page.wait_for_load_state(state)

    def wait_for_url(self, url: str, timeout: int = 30000):
        """等待URL变化"""
        self.page.wait_for_url(url, timeout=timeout)

    # ==================== 断言 ====================

    def assert_title(self, expected: str):
        """断言页面标题"""
        expect(self.page).to_have_title(expected)

    def assert_url(self, expected: str):
        """断言页面URL"""
        expect(self.page).to_have_url(expected)

    def assert_visible(self, selector: str):
        """断言元素可见"""
        expect(self.page.locator(selector)).to_be_visible()

    def assert_hidden(self, selector: str):
        """断言元素隐藏"""
        expect(self.page.locator(selector)).to_be_hidden()

    def assert_enabled(self, selector: str):
        """断言元素可用"""
        expect(self.page.locator(selector)).to_be_enabled()

    def assert_disabled(self, selector: str):
        """断言元素禁用"""
        expect(self.page.locator(selector)).to_be_disabled()

    def assert_text(self, selector: str, expected: str):
        """断言元素文本"""
        expect(self.page.locator(selector)).to_have_text(expected)

    def assert_value(self, selector: str, expected: str):
        """断言输入框值"""
        expect(self.page.locator(selector)).to_have_value(expected)

    def assert_count(self, selector: str, count: int):
        """断言元素数量"""
        expect(self.page.locator(selector)).to_have_count(count)


class ITSMUIClient(UITestClient):
    """ITSM UI测试客户端 - 封装常用业务操作"""

    def __init__(self):
        super().__init__()
        self.base_url = self.config.get('ui', 'base_url', 'http://localhost:3000')

    # ==================== 认证操作 ====================

    def login(self, username: str = None, password: str = None):
        """登录"""
        if username is None:
            username = self.config.get('api.auth', 'username', 'admin')
        if password is None:
            password = self.config.get('api.auth', 'password', 'admin123')

        self.navigate_to('/login')

        # 填写登录表单 - 支持 Ant Design Input 组件
        # 使用 #username 因为 Ant Design 会生成 id
        username_input = self.page.locator('#username').first
        username_input.wait_for(timeout=5000)
        username_input.fill(username)

        password_input = self.page.locator('#password').first
        password_input.wait_for(timeout=5000)
        password_input.fill(password)

        # 点击登录按钮 - 使用更精确的selector避免匹配到SSO按钮
        self.page.click('button[type="submit"]')

        # 等待登录成功跳转
        try:
            self.page.wait_for_url(f'{self.base_url}/**', timeout=15000)
            # 确认不在登录页
            if '/login' in self.page.url:
                raise Exception('Login failed - still on login page')
            self.logger.info(f"Login successful: {username}")
        except Exception as e:
            self.logger.error(f"Login failed: {e}")
            raise

    def logout(self):
        """登出"""
        # 点击用户菜单
        self.page.click('button:has-text("退出")')
        # 等待跳转
        self.page.wait_for_url(f'{self.base_url}/login')
        self.logger.info("Logout successful")

    # ==================== 工单操作 ====================

    def create_ticket(self, title: str, description: str, priority: str = 'medium'):
        """创建工单"""
        self.navigate_to('/tickets/create')

        # 填写工单信息
        self.page.fill('input[name="title"], input[id*="title"]', title)
        self.page.fill('textarea[name="description"]', description)
        self.page.select_option('select[name="priority"]', priority)

        # 提交
        self.page.click('button[type="submit"], button:has-text("提交")')

        # 等待创建成功
        self.page.wait_for_selector('text=创建成功', timeout=5000)
        self.logger.info(f"Ticket created: {title}")

    def search_ticket(self, keyword: str):
        """搜索工单"""
        self.navigate_to('/tickets')

        # 输入搜索关键词
        search_input = 'input[placeholder*="搜索"], input[name="keyword"]'
        self.page.fill(search_input, keyword)
        self.page.press(search_input, 'Enter')

        # 等待搜索结果
        self.page.wait_for_load_state('networkidle')
        self.logger.info(f"Ticket searched: {keyword}")

    def view_ticket_detail(self, ticket_id: int):
        """查看工单详情"""
        self.navigate_to(f'/tickets/{ticket_id}')

        # 等待页面加载
        self.wait_for_selector('text=工单详情')
        self.logger.info(f"Viewing ticket: {ticket_id}")

    # ==================== 仪表盘操作 ====================

    def go_to_dashboard(self):
        """访问仪表盘"""
        self.navigate_to('/dashboard')
        self.wait_for_selector('text=仪表盘', timeout=10000)
        self.logger.info("Dashboard loaded")

    def get_dashboard_stats(self) -> Dict[str, int]:
        """获取仪表盘统计数据"""
        stats = {}

        # 获取各个统计卡片的数据
        stat_cards = self.page.query_selector_all('.ant-statistic-content-value, [class*="statistic"]')

        for card in stat_cards:
            text = card.text_content()
            if text:
                try:
                    value = int(text.strip())
                    stats[len(stats)] = value
                except ValueError:
                    pass

        return stats


# 上下文管理器
class BrowserContext:
    """浏览器上下文管理器"""

    def __init__(self, client: UITestClient):
        self.client = client

    def __enter__(self):
        self.client.start()
        return self.client

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.client.close()


def create_ui_client() -> ITSMUIClient:
    """创建UI测试客户端"""
    return ITSMUIClient()
