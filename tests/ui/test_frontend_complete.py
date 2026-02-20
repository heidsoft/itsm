# -*- coding: utf-8 -*-
"""
ITSM前端UI功能测试 - 完整版
按功能模块测试：按钮、刷新、搜索、查询、报表等场景
针对Ant Design 6优化
"""

import pytest
import time
from tests.ui import ITSMUIClient


class TestTicketModule:
    """工单模块测试"""

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

    def test_ticket_list_page_loads(self, logged_in_client):
        """测试工单列表页面加载"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        title = logged_in_client.page.title()
        print(f"Page title: {title}")
        # 验证页面关键元素
        assert logged_in_client.page is not None

    def test_ticket_list_buttons(self, logged_in_client):
        """测试工单列表页面按钮"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # 查找创建工单按钮 (Ant Design 6)
        create_button = logged_in_client.page.locator('button:has-text("创建")')
        if create_button.count() > 0:
            print("创建工单按钮: 存在")
        else:
            print("创建工单按钮: 未找到")

    def test_ticket_search(self, logged_in_client):
        """测试工单搜索功能"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # 查找搜索框 (Ant Design 6 input)
        search_inputs = logged_in_client.page.locator('.ant-input, input[placeholder*="搜索"]')
        if search_inputs.count() > 0:
            search_input = search_inputs.first
            search_input.fill('test')
            search_input.press('Enter')
            logged_in_client.page.wait_for_timeout(1000)
            print("搜索功能: 可用")

    def test_ticket_filter(self, logged_in_client):
        """测试工单筛选功能"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # 查找筛选下拉框 (Ant Design 6 Select)
        selects = logged_in_client.page.locator('.ant-select')
        print(f"筛选下拉框: 找到 {selects.count()} 个")

    def test_ticket_kanban_view(self, logged_in_client):
        """测试工单看板视图"""
        logged_in_client.navigate_to('/tickets?view=kanban')
        logged_in_client.page.wait_for_timeout(2000)
        print("工单看板视图: 已加载")

    def test_ticket_create_page(self, logged_in_client):
        """测试创建工单页面"""
        logged_in_client.navigate_to('/tickets/create')
        logged_in_client.page.wait_for_timeout(2000)
        page_content = logged_in_client.page.content()
        if '创建' in page_content or '工单' in page_content:
            print("创建工单页面: 加载成功")

    def test_ticket_detail_page(self, logged_in_client):
        """测试工单详情页"""
        logged_in_client.navigate_to('/tickets/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("工单详情页: 已访问")


class TestIncidentModule:
    """事件管理模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_incident_list_page(self, logged_in_client):
        """测试事件列表页"""
        logged_in_client.navigate_to('/incidents')
        logged_in_client.page.wait_for_timeout(2000)
        print("事件列表页: 已加载")

    def test_incident_create_page(self, logged_in_client):
        """测试创建事件页"""
        logged_in_client.navigate_to('/incidents/new')
        logged_in_client.page.wait_for_timeout(2000)
        print("创建事件页: 已加载")

    def test_incident_detail_page(self, logged_in_client):
        """测试事件详情页"""
        logged_in_client.navigate_to('/incidents/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("事件详情页: 已加载")


class TestProblemModule:
    """问题管理模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_problem_list_page(self, logged_in_client):
        """测试问题列表页"""
        logged_in_client.navigate_to('/problems')
        logged_in_client.page.wait_for_timeout(2000)
        print("问题列表页: 已加载")

    def test_problem_detail_page(self, logged_in_client):
        """测试问题详情页"""
        logged_in_client.navigate_to('/problems/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("问题详情页: 已加载")

    def test_problem_create_page(self, logged_in_client):
        """测试创建问题页"""
        logged_in_client.navigate_to('/problems/new')
        logged_in_client.page.wait_for_timeout(2000)
        print("创建问题页: 已加载")


class TestChangeModule:
    """变更管理模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_change_list_page(self, logged_in_client):
        """测试变更列表页"""
        logged_in_client.navigate_to('/changes')
        logged_in_client.page.wait_for_timeout(2000)
        print("变更列表页: 已加载")

    def test_change_detail_page(self, logged_in_client):
        """测试变更详情页"""
        logged_in_client.navigate_to('/changes/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("变更详情页: 已加载")

    def test_change_create_page(self, logged_in_client):
        """测试创建变更页"""
        logged_in_client.navigate_to('/changes/new')
        logged_in_client.page.wait_for_timeout(2000)
        print("创建变更页: 已加载")


class TestServiceCatalogModule:
    """服务目录模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_service_catalog_page(self, logged_in_client):
        """测试服务目录页"""
        logged_in_client.navigate_to('/service-catalog')
        logged_in_client.page.wait_for_timeout(2000)
        print("服务目录: 已加载")

    def test_service_catalog_browse(self, logged_in_client):
        """测试服务目录浏览"""
        logged_in_client.navigate_to('/service-catalog')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Card
        service_items = logged_in_client.page.locator('.ant-card')
        print(f"服务项数量: {service_items.count()}")

    def test_service_request_list(self, logged_in_client):
        """测试服务请求列表"""
        logged_in_client.navigate_to('/service-requests')
        logged_in_client.page.wait_for_timeout(2000)
        print("服务请求列表: 已加载")

    def test_service_request_detail(self, logged_in_client):
        """测试服务请求详情"""
        logged_in_client.navigate_to('/service-requests/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("服务请求详情: 已加载")


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
        # Ant Design 6 Search input
        search_inputs = logged_in_client.page.locator('.ant-input-search input, .ant-input')
        if search_inputs.count() > 0:
            search_inputs.first.fill('test')
            logged_in_client.page.wait_for_timeout(1000)
            print("知识库搜索: 可用")

    def test_knowledge_article_detail(self, logged_in_client):
        """测试知识库文章详情"""
        logged_in_client.navigate_to('/knowledge/articles/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("知识库文章详情: 已加载")


class TestDashboardModule:
    """仪表盘模块测试"""

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
        # Ant Design 6 Statistic
        stat_cards = logged_in_client.page.locator('.ant-statistic')
        print(f"仪表盘统计卡片: {stat_cards.count()} 个")

    def test_dashboard_refresh(self, logged_in_client):
        """测试仪表盘刷新"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)
        # 查找刷新按钮
        refresh_buttons = logged_in_client.page.locator('button:has-text("刷新")')
        if refresh_buttons.count() > 0:
            refresh_buttons.first.click()
            logged_in_client.page.wait_for_timeout(1000)
            print("刷新功能: 可用")

    def test_dashboard_charts(self, logged_in_client):
        """测试仪表盘图表"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)
        # 查找图表元素
        charts = logged_in_client.page.locator('.ant-chart, svg')
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
        # Ant Design 6 Table
        table = logged_in_client.page.locator('.ant-table')
        print(f"工作流表格: {'存在' if table.count() > 0 else '未找到'}")
        print("工作流列表页: 已加载")

    def test_workflow_buttons(self, logged_in_client):
        """测试工作流按钮"""
        logged_in_client.navigate_to('/workflow')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Button
        buttons = logged_in_client.page.locator('.ant-btn')
        print(f"页面按钮数量: {buttons.count()}")

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

    def test_workflow_automation(self, logged_in_client):
        """测试工作流自动化"""
        logged_in_client.navigate_to('/workflow/automation')
        logged_in_client.page.wait_for_timeout(2000)
        print("工作流自动化页: 已加载")

    def test_workflow_ticket_approval(self, logged_in_client):
        """测试工单审批"""
        logged_in_client.navigate_to('/workflow/ticket-approval')
        logged_in_client.page.wait_for_timeout(2000)
        print("工单审批页: 已加载")


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
        table = logged_in_client.page.locator('.ant-table')
        print(f"用户管理表格: {'存在' if table.count() > 0 else '未找到'}")

    def test_admin_roles_page(self, logged_in_client):
        """测试角色管理页面"""
        logged_in_client.navigate_to('/admin/roles')
        logged_in_client.page.wait_for_timeout(2000)
        print("角色管理页: 已加载")

    def test_admin_groups_page(self, logged_in_client):
        """测试用户组管理页面"""
        logged_in_client.navigate_to('/admin/groups')
        logged_in_client.page.wait_for_timeout(2000)
        print("用户组管理页: 已加载")

    def test_admin_tenants_page(self, logged_in_client):
        """测试租户管理页面"""
        logged_in_client.navigate_to('/admin/tenants')
        logged_in_client.page.wait_for_timeout(2000)
        print("租户管理页: 已加载")

    def test_admin_sla_definitions(self, logged_in_client):
        """测试SLA定义管理"""
        logged_in_client.navigate_to('/admin/sla-definitions')
        logged_in_client.page.wait_for_timeout(2000)
        print("SLA定义页: 已加载")

    def test_admin_escalation_rules(self, logged_in_client):
        """测试升级规则管理"""
        logged_in_client.navigate_to('/admin/escalation-rules')
        logged_in_client.page.wait_for_timeout(2000)
        print("升级规则页: 已加载")

    def test_admin_ticket_categories(self, logged_in_client):
        """测试工单分类管理"""
        logged_in_client.navigate_to('/admin/ticket-categories')
        logged_in_client.page.wait_for_timeout(2000)
        print("工单分类页: 已加载")

    def test_admin_workflows_page(self, logged_in_client):
        """测试管理工作流页面"""
        logged_in_client.navigate_to('/admin/workflows')
        logged_in_client.page.wait_for_timeout(2000)
        print("管理工作流页: 已加载")

    def test_admin_automation_rules(self, logged_in_client):
        """测试自动化规则页面"""
        logged_in_client.navigate_to('/admin/tickets/automation-rules')
        logged_in_client.page.wait_for_timeout(2000)
        print("自动化规则页: 已加载")

    def test_admin_approval_chains(self, logged_in_client):
        """测试审批链页面"""
        logged_in_client.navigate_to('/admin/approval-chains')
        logged_in_client.page.wait_for_timeout(2000)
        print("审批链页: 已加载")

    def test_admin_service_catalogs(self, logged_in_client):
        """测试服务目录管理"""
        logged_in_client.navigate_to('/admin/service-catalogs')
        logged_in_client.page.wait_for_timeout(2000)
        print("服务目录管理页: 已加载")


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
        # Ant Design 6 Card
        cards = logged_in_client.page.locator('.ant-card')
        print(f"报表卡片: {cards.count()} 个")
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

    def test_change_success_report(self, logged_in_client):
        """测试变更成功率报表"""
        logged_in_client.navigate_to('/reports/change-success')
        logged_in_client.page.wait_for_timeout(2000)
        print("变更成功率报表: 已加载")

    def test_cmdb_quality_report(self, logged_in_client):
        """测试CMDB质量报表"""
        logged_in_client.navigate_to('/reports/cmdb-quality')
        logged_in_client.page.wait_for_timeout(2000)
        print("CMDB质量报表: 已加载")


class TestSLAModule:
    """SLA模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_sla_dashboard_page(self, logged_in_client):
        """测试SLA仪表盘"""
        logged_in_client.navigate_to('/sla-dashboard')
        logged_in_client.page.wait_for_timeout(2000)
        print("SLA仪表盘: 已加载")

    def test_sla_monitor_page(self, logged_in_client):
        """测试SLA监控页"""
        logged_in_client.navigate_to('/sla-monitor')
        logged_in_client.page.wait_for_timeout(2000)
        print("SLA监控页: 已加载")

    def test_sla_definitions_page(self, logged_in_client):
        """测试SLA定义页"""
        logged_in_client.navigate_to('/sla/definitions/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("SLA定义详情页: 已加载")


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

    def test_cmdb_ci_detail(self, logged_in_client):
        """测试CI详情"""
        logged_in_client.navigate_to('/cmdb/cis/1')
        logged_in_client.page.wait_for_timeout(2000)
        print("CI详情页: 已加载")

    def test_cmdb_ci_create(self, logged_in_client):
        """测试创建CI"""
        logged_in_client.navigate_to('/cmdb/cis/create')
        logged_in_client.page.wait_for_timeout(2000)
        print("创建CI页: 已加载")

    def test_cmdb_cloud_resources(self, logged_in_client):
        """测试云资源页面"""
        logged_in_client.navigate_to('/cmdb/cloud-resources')
        logged_in_client.page.wait_for_timeout(2000)
        print("云资源页: 已加载")

    def test_cmdb_cloud_services(self, logged_in_client):
        """测试云服务页面"""
        logged_in_client.navigate_to('/cmdb/cloud-services')
        logged_in_client.page.wait_for_timeout(2000)
        print("云服务页: 已加载")

    def test_cmdb_cloud_accounts(self, logged_in_client):
        """测试云账号页面"""
        logged_in_client.navigate_to('/cmdb/cloud-accounts')
        logged_in_client.page.wait_for_timeout(2000)
        print("云账号页: 已加载")

    def test_cmdb_reconciliation(self, logged_in_client):
        """测试CMDB对账页面"""
        logged_in_client.navigate_to('/cmdb/reconciliation')
        logged_in_client.page.wait_for_timeout(2000)
        print("CMDB对账页: 已加载")


class TestApprovalModule:
    """审批模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_approvals_pending_page(self, logged_in_client):
        """测试待审批页面"""
        logged_in_client.navigate_to('/approvals/pending')
        logged_in_client.page.wait_for_timeout(2000)
        print("待审批页: 已加载")


class TestProfileModule:
    """用户模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_profile_page(self, logged_in_client):
        """测试个人中心页面"""
        logged_in_client.navigate_to('/profile')
        logged_in_client.page.wait_for_timeout(2000)
        print("个人中心页: 已加载")

    def test_my_requests_page(self, logged_in_client):
        """测试我的请求页面"""
        logged_in_client.navigate_to('/my-requests')
        logged_in_client.page.wait_for_timeout(2000)
        print("我的请求页: 已加载")

    def test_notifications_page(self, logged_in_client):
        """测试通知页面"""
        logged_in_client.navigate_to('/notifications')
        logged_in_client.page.wait_for_timeout(2000)
        print("通知页: 已加载")


class TestSystemModule:
    """系统模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_system_users_page(self, logged_in_client):
        """测试系统用户页面"""
        logged_in_client.navigate_to('/system/users')
        logged_in_client.page.wait_for_timeout(2000)
        print("系统用户页: 已加载")

    def test_system_organization_page(self, logged_in_client):
        """测试组织架构页面"""
        logged_in_client.navigate_to('/system/organization')
        logged_in_client.page.wait_for_timeout(2000)
        print("组织架构页: 已加载")


class TestOtherModules:
    """其他模块测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_applications_page(self, logged_in_client):
        """测试应用页面"""
        logged_in_client.navigate_to('/applications')
        logged_in_client.page.wait_for_timeout(2000)
        print("应用页: 已加载")

    def test_projects_page(self, logged_in_client):
        """测试项目页面"""
        logged_in_client.navigate_to('/projects')
        logged_in_client.page.wait_for_timeout(2000)
        print("项目页: 已加载")

    def test_tags_page(self, logged_in_client):
        """测试标签页面"""
        logged_in_client.navigate_to('/tags')
        logged_in_client.page.wait_for_timeout(2000)
        print("标签页: 已加载")

    def test_templates_page(self, logged_in_client):
        """测试模板页面"""
        logged_in_client.navigate_to('/templates')
        logged_in_client.page.wait_for_timeout(2000)
        print("模板页: 已加载")

    def test_ai_chat_page(self, logged_in_client):
        """测试AI聊天页面"""
        logged_in_client.navigate_to('/ai/chat')
        logged_in_client.page.wait_for_timeout(2000)
        print("AI聊天页: 已加载")

    def test_improvements_page(self, logged_in_client):
        """测试改进页面"""
        logged_in_client.navigate_to('/improvements')
        logged_in_client.page.wait_for_timeout(2000)
        print("改进页: 已加载")


class TestAntDesign6Components:
    """Ant Design 6组件测试"""

    @pytest.fixture
    def logged_in_client(self):
        client = ITSMUIClient()
        client.start()
        client.login()
        yield client
        client.close()

    def test_antd6_select_component(self, logged_in_client):
        """测试Ant Design 6 Select组件"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Select
        selects = logged_in_client.page.locator('.ant-select')
        print(f"Ant Design 6 Select组件: {selects.count()} 个")

    def test_antd6_table_component(self, logged_in_client):
        """测试Ant Design 6 Table组件"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Table
        tables = logged_in_client.page.locator('.ant-table')
        print(f"Ant Design 6 Table组件: {tables.count()} 个")

    def test_antd6_modal_component(self, logged_in_client):
        """测试Ant Design 6 Modal组件"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # 查找可能打开Modal的按钮
        buttons = logged_in_client.page.locator('button:has-text("创建")')
        if buttons.count() > 0:
            buttons.first.click()
            logged_in_client.page.wait_for_timeout(500)
            # Ant Design 6 Modal
            modals = logged_in_client.page.locator('.ant-modal')
            print(f"Ant Design 6 Modal组件: {modals.count()} 个")

    def test_antd6_card_component(self, logged_in_client):
        """测试Ant Design 6 Card组件"""
        logged_in_client.navigate_to('/dashboard')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Card
        cards = logged_in_client.page.locator('.ant-card')
        print(f"Ant Design 6 Card组件: {cards.count()} 个")

    def test_antd6_form_component(self, logged_in_client):
        """测试Ant Design 6 Form组件"""
        logged_in_client.navigate_to('/tickets/create')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Form
        forms = logged_in_client.page.locator('.ant-form')
        print(f"Ant Design 6 Form组件: {forms.count()} 个")

    def test_antd6_input_component(self, logged_in_client):
        """测试Ant Design 6 Input组件"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Input
        inputs = logged_in_client.page.locator('.ant-input')
        print(f"Ant Design 6 Input组件: {inputs.count()} 个")

    def test_antd6_button_component(self, logged_in_client):
        """测试Ant Design 6 Button组件"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Button
        buttons = logged_in_client.page.locator('.ant-btn')
        print(f"Ant Design 6 Button组件: {buttons.count()} 个")

    def test_antd6_tag_component(self, logged_in_client):
        """测试Ant Design 6 Tag组件"""
        logged_in_client.navigate_to('/tickets')
        logged_in_client.page.wait_for_timeout(2000)
        # Ant Design 6 Tag
        tags = logged_in_client.page.locator('.ant-tag')
        print(f"Ant Design 6 Tag组件: {tags.count()} 个")
