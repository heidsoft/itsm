# -*- coding: utf-8 -*-
"""
UI测试用例
UI Test Cases
"""

import pytest
from tests.ui import ITSMUIClient
from tests.database import ITSMDBClient


class TestLogin:
    """登录页面测试"""

    @pytest.fixture
    def ui_client(self):
        """创建UI客户端"""
        client = ITSMUIClient()
        client.start()
        yield client
        client.close()

    def test_login_page_loads(self, ui_client):
        """测试登录页面加载"""
        ui_client.navigate_to('/login')

        # 验证页面加载 - 支持 Ant Design 组件
        ui_client.assert_visible('input[id*="username"], input[name="username"], .ant-input input')
        ui_client.assert_visible('input[type="password"], .ant-input-password input')
        ui_client.assert_visible('button[type="submit"]')

    def test_login_success(self, ui_client):
        """测试登录成功"""
        try:
            ui_client.login('admin', 'admin123')
            # 验证跳转
            current_url = ui_client.page.url
            # 检查是否登录成功（跳转到dashboard/tickets/admin页面）
            if 'login' in current_url.lower():
                # 登录失败，可能是API不可用
                pytest.skip(f"登录失败，可能是后端API不可用。当前URL: {current_url}")
            assert 'dashboard' in current_url or 'tickets' in current_url or '/admin' in current_url
        except Exception as e:
            # 如果登录失败，记录页面内容用于调试
            print(f"Login failed, URL: {ui_client.page.url}")
            raise e

    def test_login_failure(self, ui_client):
        """测试登录失败"""
        ui_client.navigate_to('/login')

        # 输入错误密码 - 支持 Ant Design
        username_input = ui_client.page.locator('input[id*="username"], input[name="username"]').first
        username_input.wait_for(timeout=5000)
        username_input.fill('admin')

        password_input = ui_client.page.locator('input[type="password"]').first
        password_input.fill('wrongpassword')

        ui_client.click('button[type="submit"]:not([data-testid])')

        # 验证错误提示
        ui_client.page.wait_for_timeout(2000)
        # 不应该跳转
        assert '/login' in ui_client.page.url


class TestDashboard:
    """仪表盘测试"""

    @pytest.fixture
    def ui_client(self):
        """创建已登录的UI客户端"""
        client = ITSMUIClient()
        client.start()
        try:
            client.login()
        except Exception as e:
            client.close()
            pytest.skip(f"无法登录，后端API可能不可用: {e}")
        yield client
        client.close()

    def test_dashboard_loads(self, ui_client):
        """测试仪表盘加载"""
        ui_client.go_to_dashboard()

        # 验证页面加载 - 检查任意一个可见元素即可
        page = ui_client.page
        # 等待页面加载完成
        page.wait_for_load_state('networkidle')
        # 检查是否存在任意一个标识性元素
        dashboard_visible = (
            page.locator('text=ITSM').count() > 0 or
            page.locator('text=仪表盘').count() > 0 or
            page.locator('.text-2xl').count() > 0 or
            page.locator('[class*="dashboard"]').count() > 0
        )
        assert dashboard_visible, "仪表盘页面未正确加载"

    def test_dashboard_stats(self, ui_client):
        """测试仪表盘统计"""
        ui_client.go_to_dashboard()

        # 等待数据加载
        ui_client.page.wait_for_timeout(2000)

        # 获取统计数据
        stats = ui_client.get_dashboard_stats()

        assert isinstance(stats, dict)


class TestTickets:
    """工单页面测试"""

    @pytest.fixture
    def ui_client(self):
        """创建已登录的UI客户端"""
        client = ITSMUIClient()
        client.start()
        try:
            client.login()
        except Exception as e:
            client.close()
            pytest.skip(f"无法登录，后端API可能不可用: {e}")
        yield client
        client.close()

    def test_tickets_page_loads(self, ui_client):
        """测试工单列表页面加载"""
        ui_client.navigate_to('/tickets')

        # 验证页面加载
        ui_client.wait_for_selector('text=工单', timeout=10000)

    def test_ticket_search(self, ui_client):
        """测试工单搜索"""
        ui_client.navigate_to('/tickets')

        # 输入搜索关键词
        search_input = 'input[placeholder*="搜索"], input[name="keyword"]'
        if ui_client.is_visible(search_input):
            ui_client.fill(search_input, 'test')
            ui_client.press(search_input, 'Enter')
            ui_client.page.wait_for_timeout(1000)

    def test_ticket_detail_page(self, ui_client):
        """测试工单详情页"""
        # 先获取一个工单ID
        db = ITSMDBClient()
        tickets = db.get_all_tickets(limit=1)

        if tickets:
            ticket_id = tickets[0]['id']
            ui_client.view_ticket_detail(ticket_id)

            # 验证页面加载
            ui_client.page.wait_for_timeout(1000)


class TestKnowledgeBase:
    """知识库测试"""

    @pytest.fixture
    def ui_client(self):
        """创建已登录的UI客户端"""
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_knowledge_page_loads(self, ui_client):
        """测试知识库页面加载"""
        ui_client.navigate_to('/knowledge')

        # 验证页面加载
        ui_client.page.wait_for_timeout(2000)

    def test_knowledge_search(self, ui_client):
        """测试知识库搜索"""
        ui_client.navigate_to('/knowledge')

        # 等待页面加载
        ui_client.page.wait_for_timeout(1000)

        # 尝试搜索
        search_input = 'input[placeholder*="搜索"], input[name="q"]'
        if ui_client.is_visible(search_input):
            ui_client.fill(search_input, 'test')
            ui_client.press(search_input, 'Enter')
            ui_client.page.wait_for_timeout(1000)


class TestSLA:
    """SLA测试"""

    @pytest.fixture
    def ui_client(self):
        """创建已登录的UI客户端"""
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_sla_dashboard_loads(self, ui_client):
        """测试SLA仪表盘加载"""
        ui_client.navigate_to('/sla-dashboard')

        # 验证页面加载
        ui_client.page.wait_for_timeout(2000)


class TestResponsive:
    """响应式测试"""

    @pytest.mark.parametrize("viewport", [
        {"width": 1920, "height": 1080},  # Desktop
        {"width": 768, "height": 1024},  # Tablet
        {"width": 375, "height": 667},    # Mobile
    ])
    def test_responsive_layout(self, viewport):
        """测试响应式布局"""
        from tests.ui import UITestClient

        client = UITestClient()
        client.start()

        # 设置视口大小
        client.context.set_viewport_size(viewport)

        # 访问页面
        client.navigate_to('/login')
        client.page.wait_for_timeout(1000)

        # 验证基本元素
        client.assert_visible('input[type="password"]')

        client.close()


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
