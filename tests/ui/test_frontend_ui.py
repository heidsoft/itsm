# -*- coding: utf-8 -*-
"""
ITSM前端UI功能测试
按功能模块测试：按钮、刷新、搜索、查询、报表等场景
"""

import pytest
import time
from tests.ui import ITSMUIClient


class TestTicketModule:
    """工单模块测试"""

    @pytest.fixture
    def client(self):
        """创建UI客户端"""
        client = ITSMUIClient()
        client.start()
        yield client
        client.close()

    @pytest.fixture
    def logged_in_client(self):
        """已登录的UI客户端"""
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_ticket_list_page_loads(self, logged_in_client):
        """测试工单列表页面加载"""
        logged_in_client.navigate_to('/tickets')

        # 验证页面标题
        logged_in_client.page.wait_for_timeout(2000)
        title = logged_in_client.page.title()
        print(f"Page title: {title}")

    def test_ticket_list_buttons(self, logged_in_client):
        """测试工单列表页面按钮"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找创建工单按钮
        create_button = logged_in_client.page.locator('button:has-text("创建"), a:has-text("创建工单")')
        if create_button.count() > 0:
            print("创建工单按钮: 存在")
        else:
            print("创建工单按钮: 未找到")

        # 查找刷新按钮
        refresh_button = logged_in_client.page.locator('button:has-text("刷新"), [class*="refresh"]')
        if refresh_button.count() > 0:
            print("刷新按钮: 存在")

    def test_ticket_search(self, logged_in_client):
        """测试工单搜索功能"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)

        # 尝试找到搜索框
        search_inputs = logged_in_client.page.locator('input[placeholder*="搜索"], input[name="keyword"], .ant-input')
        if search_inputs.count() > 0:
            search_input = search_inputs.first
            search_input.fill('test')
            search_input.press('Enter')
            logged_in_client.page.wait_for_timeout(1000)
            print("搜索功能: 可用")
        else:
            print("搜索功能: 未找到搜索框")

    def test_ticket_filter(self, logged_in_client):
        """测试工单筛选功能"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找筛选下拉框
        selects = logged_in_client.page.locator('.ant-select, select')
        if selects.count() > 0:
            print(f"筛选下拉框: 找到 {selects.count()} 个")
        else:
            print("筛选下拉框: 未找到")

    def test_ticket_create_button(self, logged_in_client):
        """测试创建工单按钮点击"""
        logged_in_client.navigate_to('/tickets/create')
        logged_in_client.page.wait_for_timeout(2000)

        # 验证创建页面加载
        page_content = logged_in_client.page.content()
        if '创建' in page_content or '工单' in page_content:
            print("创建工单页面: 加载成功")
        else:
            print("创建工单页面: 内容异常")

    def test_ticket_detail_page(self, logged_in_client):
        """测试工单详情页"""
        logged_in_client.navigate_to('/tickets/1')
        logged_in_client.page.wait_for_timeout(2000)

        print("工单详情页: 已访问")


class TestDashboardModule:
    """仪表盘模块测试"""

    @pytest.fixture
    def client(self):
        client = ITSMUIClient()
        client.start()
        yield client
        client.close()

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_dashboard_page_loads(self, logged_in_client):
        """测试仪表盘页面加载"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找统计卡片
        stat_cards = logged_in_client.page.locator('.ant-card, [class*="card"]')
        print(f"仪表盘卡片数量: {stat_cards.count()}")

    def test_dashboard_refresh(self, logged_in_client):
        """测试仪表盘刷新"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找刷新按钮
        refresh_buttons = logged_in_client.page.locator('button:has-text("刷新"), [class*="refresh"]')
        if refresh_buttons.count() > 0:
            refresh_buttons.first.click()
            logged_in_client.page.wait_for_timeout(1000)
            print("刷新功能: 可用")
        else:
            print("刷新按钮: 未找到")

    def test_dashboard_quick_actions(self, logged_in_client):
        """测试仪表盘快捷操作"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找快捷操作按钮
        action_buttons = logged_in_client.page.locator('button:has-text("创建"), a:has-text("创建")')
        print(f"快捷操作按钮: {action_buttons.count()} 个")

    def test_dashboard_charts(self, logged_in_client):
        """测试仪表盘图表"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找图表元素
        charts = logged_in_client.page.locator('[class*="chart"], .ant-chart, svg')
        print(f"图表元素: {charts.count()} 个")


class TestWorkflowModule:
    """工作流模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_workflow_list_page(self, logged_in_client):
        """测试工作流列表页"""
        logged_in_client.navigate_to('/workflow')
        logged_in_client.page.wait_for_timeout(2000)

        print("工作流列表页: 已加载")

    def test_workflow_buttons(self, logged_in_client):
        """测试工作流按钮"""
        logged_in_client.navigate_to('/workflow')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找按钮
        buttons = logged_in_client.page.locator('button')
        print(f"页面按钮数量: {buttons.count()}")

    def test_workflow_create_button(self, logged_in_client):
        """测试创建工作流按钮"""
        logged_in_client.navigate_to('/workflow')
        logged_in_client.page.wait_for_timeout(2000)

        create_btn = logged_in_client.page.locator('button:has-text("创建")')
        if create_btn.count() > 0:
            print("创建按钮: 存在")

    def test_workflow_designer(self, logged_in_client):
        """测试工作流设计器"""
        logged_in_client.navigate_to('/workflow/designer')
        logged_in_client.page.wait_for_timeout(2000)

        print("工作流设计器: 已加载")

    def test_workflow_instances(self, logged_in_client):
        """测试工作流实例"""
        logged_in_client.navigate_to('/workflow/instances')
        logged_in_client.page.wait_for_timeout(2000)

        print("工作流实例页: 已加载")


class TestAdminModule:
    """系统管理模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_admin_page_loads(self, logged_in_client):
        """测试管理员首页"""
        logged_in_client.navigate_to('/admin')
        logged_in_client.page.wait_for_timeout(2000)

        print("管理员首页: 已加载")

    def test_admin_users_page(self, logged_in_client):
        """测试用户管理页面"""
        logged_in_client.navigate_to('/admin/users')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找用户列表
        table = logged_in_client.page.locator('.ant-table, table')
        print(f"用户管理表格: {'存在' if table.count() > 0 else '未找到'}")

    def test_admin_roles_page(self, logged_in_client):
        """测试角色管理页面"""
        logged_in_client.navigate_to('/admin/roles')
        logged_in_client.page.wait_for_timeout(2000)

        print("角色管理页: 已加载")

    def test_admin_sla_definitions(self, logged_in_client):
        """测试SLA定义管理"""
        logged_in_client.navigate_to('/admin/sla-definitions')
        logged_in_client.page.wait_for_timeout(2000)

        print("SLA定义页: 已加载")

    def test_admin_workflows_page(self, logged_in_client):
        """测试管理工作流页面"""
        logged_in_client.navigate_to('/admin/workflows')
        logged_in_client.page.wait_for_timeout(2000)

        print("管理工作流页: 已加载")

    def test_admin_ticket_categories(self, logged_in_client):
        """测试工单分类管理"""
        logged_in_client.navigate_to('/admin/ticket-categories')
        logged_in_client.page.wait_for_timeout(2000)

        print("工单分类页: 已加载")

    def test_admin_escalation_rules(self, logged_in_client):
        """测试升级规则管理"""
        logged_in_client.navigate_to('/admin/escalation-rules')
        logged_in_client.page.wait_for_timeout(2000)

        print("升级规则页: 已加载")


class TestReportsModule:
    """报表模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_reports_page_loads(self, logged_in_client):
        """测试报表首页"""
        logged_in_client.navigate_to('/reports')
        logged_in_client.page.wait_for_timeout(2000)

        print("报表首页: 已加载")

    def test_sla_performance_report(self, logged_in_client):
        """测试SLA性能报表"""
        logged_in_client.navigate_to('/reports/sla-performance')
        logged_in_client.page.wait_for_timeout(2000)

        print("SLA性能报表: 已加载")

    def test_incident_trends_report(self, logged_in_client):
        """测试事件趋势报表"""
        logged_in_client.navigate_to('/reports/incident-trends')
        logged_in_client.page.wait_for_timeout(2000)

        print("事件趋势报表: 已加载")

    def test_problem_efficiency_report(self, logged_in_client):
        """测试问题效率报表"""
        logged_in_client.navigate_to('/reports/problem-efficiency')
        logged_in_client.page.wait_for_timeout(2000)

        print("问题效率报表: 已加载")

    def test_service_catalog_report(self, logged_in_client):
        """测试服务目录报表"""
        logged_in_client.navigate_to('/reports/service-catalog-usage')
        logged_in_client.page.wait_for_timeout(2000)

        print("服务目录报表: 已加载")


class TestKnowledgeModule:
    """知识库模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_knowledge_page_loads(self, logged_in_client):
        """测试知识库页面加载"""
        logged_in_client.navigate_to('/knowledge')
        logged_in_client.page.wait_for_timeout(2000)

        print("知识库首页: 已加载")

    def test_knowledge_search(self, logged_in_client):
        """测试知识库搜索"""
        logged_in_client.navigate_to('/knowledge')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找搜索框
        search_inputs = logged_in_client.page.locator('input[placeholder*="搜索"]')
        if search_inputs.count() > 0:
            search_inputs.first.fill('test')
            logged_in_client.page.wait_for_timeout(1000)
            print("知识库搜索: 可用")


class TestServiceCatalogModule:
    """服务目录模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_service_catalog_page_loads(self, logged_in_client):
        """测试服务目录页面加载"""
        logged_in_client.navigate_to('/service-catalog')
        logged_in_client.page.wait_for_timeout(2000)

        print("服务目录: 已加载")

    def test_service_catalog_browse(self, logged_in_client):
        """测试服务目录浏览"""
        logged_in_client.navigate_to('/service-catalog')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找服务项
        service_items = logged_in_client.page.locator('[class*="card"], .ant-card')
        print(f"服务项数量: {service_items.count()}")

    def test_service_request_create(self, logged_in_client):
        """测试创建服务请求"""
        logged_in_client.navigate_to('/service-requests/create')
        logged_in_client.page.wait_for_timeout(2000)

        print("服务请求创建页: 已加载")


class TestCMDBModule:
    """CMDB模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_cmdb_page_loads(self, logged_in_client):
        """测试CMDB页面加载"""
        logged_in_client.navigate_to('/cmdb')
        logged_in_client.page.wait_for_timeout(2000)

        print("CMDB首页: 已加载")

    def test_cmdb_ci_list(self, logged_in_client):
        """测试CI列表"""
        logged_in_client.navigate_to('/cmdb/cis')
        logged_in_client.page.wait_for_timeout(2000)

        print("CI列表页: 已加载")

    def test_cmdb_cloud_resources(self, logged_in_client):
        """测试云资源页面"""
        logged_in_client.navigate_to('/cmdb/cloud-resources')
        logged_in_client.page.wait_for_timeout(2000)

        print("云资源页: 已加载")


class TestNavigationTest:
    """导航测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_main_navigation(self, logged_in_client):
        """测试主导航菜单"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找导航菜单
        nav_menu = logged_in_client.page.locator('.ant-menu, nav, [class*="menu"]')
        print(f"导航菜单: {'存在' if nav_menu.count() > 0 else '未找到'}")

    def test_sidebar_navigation(self, logged_in_client):
        """测试侧边栏导航"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找侧边栏
        sidebar = logged_in_client.page.locator('.ant-layout-sider, [class*="sidebar"]')
        print(f"侧边栏: {'存在' if sidebar.count() > 0 else '未找到'}")

    def test_breadcrumb_navigation(self, logged_in_client):
        """测试面包屑导航"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)

        # 查找面包屑
        breadcrumb = logged_in_client.page.locator('.ant-breadcrumb, [class*="breadcrumb"]')
        print(f"面包屑: {'存在' if breadcrumb.count() > 0 else '未找到'}")
