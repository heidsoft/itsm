# -*- coding: utf-8 -*-
"""
API测试客户端
API Test Client
"""

import json
import time
import requests
from typing import Any, Dict, Optional, List
from urllib.parse import urljoin
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

from .. import get_config, get_logger, context


class APIClient:
    """API测试客户端"""

    def __init__(self, base_url: str = None, token: str = None):
        self.config = get_config()
        self.logger = get_logger('api_client')

        # 获取base_url
        if base_url is None:
            base_url = self.config.get('api', 'base_url', fallback='http://localhost:8080')

        self.base_url = base_url.rstrip('/')
        self.token = token
        self.session = self._create_session()
        self.last_response = None
        self.last_request = None

    def _create_session(self) -> requests.Session:
        """创建带重试的session"""
        session = requests.Session()

        # 配置重试策略
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )

        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("http://", adapter)
        session.mount("https://", adapter)

        return session

    def _get_headers(self) -> Dict[str, str]:
        """获取请求头"""
        headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }

        if self.token:
            headers['Authorization'] = f'Bearer {self.token}'

        tenant_id = context.get_tenant_id()
        if tenant_id:
            headers['X-Tenant-ID'] = str(tenant_id)

        return headers

    def _build_url(self, endpoint: str) -> str:
        """构建完整URL"""
        if endpoint.startswith('http'):
            return endpoint
        return urljoin(self.base_url, endpoint)

    def _log_request(self, method: str, url: str, **kwargs):
        """记录请求信息"""
        self.last_request = {
            'method': method,
            'url': url,
            'headers': kwargs.get('headers', {}),
            'json': kwargs.get('json'),
            'params': kwargs.get('params')
        }
        self.logger.debug(f"Request: {method} {url}")

    def _log_response(self, response: requests.Response):
        """记录响应信息"""
        self.last_response = response
        self.logger.debug(
            f"Response: {response.status_code} "
            f"Time: {response.elapsed.total_seconds():.3f}s"
        )

    def request(
        self,
        method: str,
        endpoint: str,
        params: Dict = None,
        json: Dict = None,
        data: Any = None,
        headers: Dict = None,
        files: Dict = None,
        timeout: int = None,
        allow_redirects: bool = True,
        **kwargs
    ) -> requests.Response:
        """发送HTTP请求"""

        url = self._build_url(endpoint)

        if timeout is None:
            timeout = self.config.get_int('api', 'timeout', 30)

        # 合并headers
        request_headers = self._get_headers()
        if headers:
            request_headers.update(headers)

        self._log_request(method, url, headers=request_headers, json=json, params=params)

        try:
            response = self.session.request(
                method=method,
                url=url,
                params=params,
                json=json,
                data=data,
                headers=request_headers,
                files=files,
                timeout=timeout,
                allow_redirects=allow_redirects,
                **kwargs
            )

            self._log_response(response)
            return response

        except requests.exceptions.Timeout:
            self.logger.error(f"Request timeout: {url}")
            raise
        except requests.exceptions.ConnectionError as e:
            self.logger.error(f"Connection error: {e}")
            raise
        except requests.exceptions.RequestException as e:
            self.logger.error(f"Request error: {e}")
            raise

    def get(self, endpoint: str, params: Dict = None, **kwargs) -> requests.Response:
        """GET请求"""
        return self.request('GET', endpoint, params=params, **kwargs)

    def post(self, endpoint: str, json: Dict = None, **kwargs) -> requests.Response:
        """POST请求"""
        return self.request('POST', endpoint, json=json, **kwargs)

    def put(self, endpoint: str, json: Dict = None, **kwargs) -> requests.Response:
        """PUT请求"""
        return self.request('PUT', endpoint, json=json, **kwargs)

    def patch(self, endpoint: str, json: Dict = None, **kwargs) -> requests.Response:
        """PATCH请求"""
        return self.request('PATCH', endpoint, json=json, **kwargs)

    def delete(self, endpoint: str, **kwargs) -> requests.Response:
        """DELETE请求"""
        return self.request('DELETE', endpoint, **kwargs)

    def upload(self, endpoint: str, files: Dict, data: Dict = None, **kwargs) -> requests.Response:
        """文件上传"""
        url = self._build_url(endpoint)
        headers = self._get_headers()
        # 上传文件时不需要Content-Type，让requests自动处理

        self._log_request('POST', url, files=files, data=data)

        response = self.session.post(
            url,
            files=files,
            data=data,
            headers=headers,
            **kwargs
        )

        self._log_response(response)
        return response

    def download(self, endpoint: str, **kwargs) -> bytes:
        """文件下载"""
        response = self.get(endpoint, stream=True, **kwargs)
        response.raise_for_status()
        return response.content


class ITSMAPIClient(APIClient):
    """ITSM API客户端 - 封装常用业务接口"""

    def __init__(self, base_url: str = None, token: str = None):
        super().__init__(base_url, token)
        self.logger = get_logger('itsm_api')

    # ==================== 认证接口 ====================

    def login(self, username: str = None, password: str = None) -> Dict:
        """用户登录"""
        if username is None:
            username = self.config.get('api.auth', 'username', 'admin')
        if password is None:
            password = self.config.get('api.auth', 'password', 'admin123')

        response = self.post('/api/v1/auth/login', json={
            'username': username,
            'password': password
        })

        if response.status_code == 200:
            data = response.json()
            if data.get('code') == 0:
                self.token = data.get('data', {}).get('access_token')
                context.set_token(self.token)

                # 设置租户ID
                tenant_id = data.get('data', {}).get('tenant_id', 1)
                context.set_tenant_id(tenant_id)

                self.logger.info(f"Login successful: {username}")
                return data.get('data', {})

        self.logger.error(f"Login failed: {response.text}")
        response.raise_for_status()
        return {}

    def logout(self) -> bool:
        """用户登出"""
        response = self.post('/api/v1/auth/logout')
        self.token = None
        context.clear()
        return response.status_code == 200

    # ==================== 工单接口 ====================

    def get_tickets(self, params: Dict = None) -> Dict:
        """获取工单列表"""
        response = self.get('/api/v1/tickets', params=params)
        return response.json()

    def get_ticket(self, ticket_id: int) -> Dict:
        """获取工单详情"""
        response = self.get(f'/api/v1/tickets/{ticket_id}')
        return response.json()

    def create_ticket(self, data: Dict) -> Dict:
        """创建工单"""
        response = self.post('/api/v1/tickets', json=data)
        return response.json()

    def update_ticket(self, ticket_id: int, data: Dict) -> Dict:
        """更新工单"""
        response = self.put(f'/api/v1/tickets/{ticket_id}', json=data)
        return response.json()

    def delete_ticket(self, ticket_id: int) -> bool:
        """删除工单"""
        response = self.delete(f'/api/v1/tickets/{ticket_id}')
        return response.status_code == 200

    def get_ticket_comments(self, ticket_id: int) -> Dict:
        """获取工单评论"""
        response = self.get(f'/api/v1/tickets/{ticket_id}/comments')
        return response.json()

    def add_ticket_comment(self, ticket_id: int, content: str) -> Dict:
        """添加工单评论"""
        response = self.post(f'/api/v1/tickets/{ticket_id}/comments', json={
            'content': content
        })
        return response.json()

    def get_ticket_sla(self, ticket_id: int) -> Dict:
        """获取工单SLA信息"""
        response = self.get(f'/api/v1/tickets/{ticket_id}/sla')
        return response.json()

    # ==================== 用户接口 ====================

    def get_users(self, params: Dict = None) -> Dict:
        """获取用户列表"""
        response = self.get('/api/v1/users', params=params)
        return response.json()

    def get_user(self, user_id: int) -> Dict:
        """获取用户详情"""
        response = self.get(f'/api/v1/users/{user_id}')
        return response.json()

    # ==================== 仪表盘接口 ====================

    def get_dashboard_overview(self) -> Dict:
        """获取仪表盘概览"""
        response = self.get('/api/v1/dashboard/overview')
        return response.json()

    def get_ticket_stats(self) -> Dict:
        """获取工单统计"""
        response = self.get('/api/v1/tickets/stats')
        return response.json()

    # ==================== SLA接口 ====================

    def get_sla_definitions(self, params: Dict = None) -> Dict:
        """获取SLA定义列表"""
        response = self.get('/api/v1/sla/definitions', params=params)
        return response.json()

    def get_sla_violations(self, params: Dict = None) -> Dict:
        """获取SLA违规列表"""
        response = self.get('/api/v1/sla/violations', params=params)
        return response.json()

    # ==================== 知识库接口 ====================

    def get_knowledge_articles(self, params: Dict = None) -> Dict:
        """获取知识库文章列表"""
        response = self.get('/api/v1/knowledge/articles', params=params)
        return response.json()

    def get_knowledge_article(self, article_id: int) -> Dict:
        """获取知识库文章详情"""
        response = self.get(f'/api/v1/knowledge/articles/{article_id}')
        return response.json()

    def search_knowledge(self, keyword: str) -> Dict:
        """搜索知识库"""
        response = self.post('/api/v1/knowledge/search', json={'keyword': keyword})
        return response.json()

    # ==================== CMDB接口 ====================

    def get_cmdb_cis(self, params: Dict = None) -> Dict:
        """获取配置项列表"""
        response = self.get('/api/v1/cmdb/cis', params=params)
        return response.json()

    def get_cmdb_ci(self, ci_id: int) -> Dict:
        """获取配置项详情"""
        response = self.get(f'/api/v1/cmdb/cis/{ci_id}')
        return response.json()

    def get_cmdb_relationships(self, params: Dict = None) -> Dict:
        """获取配置项关系列表"""
        response = self.get('/api/v1/configuration-items/relationships', params=params)
        return response.json()

    def get_cmdb_cloud_resources(self, params: Dict = None) -> Dict:
        """获取云资源列表"""
        response = self.get('/api/v1/cmdb/cloud-resources', params=params)
        return response.json()

    def get_cmdb_cloud_services(self, params: Dict = None) -> Dict:
        """获取云服务列表"""
        response = self.get('/api/v1/cmdb/cloud-services', params=params)
        return response.json()

    # ==================== BPMN工作流接口 ====================

    def get_bpmn_processes(self, params: Dict = None) -> Dict:
        """获取BPMN流程定义列表"""
        response = self.get('/api/v1/bpmn/process-definitions', params=params)
        return response.json()

    def get_bpmn_process(self, process_key: str) -> Dict:
        """获取BPMN流程定义详情"""
        response = self.get(f'/api/v1/bpmn/process-definitions/{process_key}')
        return response.json()

    def get_bpmn_tasks(self, params: Dict = None) -> Dict:
        """获取BPMN任务列表"""
        response = self.get('/api/v1/bpmn/tasks', params=params)
        return response.json()

    def get_bpmn_instances(self, params: Dict = None) -> Dict:
        """获取BPMN实例列表"""
        response = self.get('/api/v1/bpmn/process-instances', params=params)
        return response.json()

    # ==================== 通知接口 ====================

    def get_notifications(self, params: Dict = None) -> Dict:
        """获取通知列表"""
        response = self.get('/api/v1/notifications', params=params)
        return response.json()

    def get_notification_unread_count(self) -> Dict:
        """获取未读通知数量"""
        response = self.get('/api/v1/notifications/unread-count')
        return response.json()

    def mark_notification_read(self, notification_id: int) -> Dict:
        """标记通知为已读"""
        response = self.put(f'/api/v1/notifications/{notification_id}/read')
        return response.json()

    # ==================== 工单分类接口 ====================

    def get_ticket_categories(self, params: Dict = None) -> Dict:
        """获取工单分类列表"""
        response = self.get('/api/v1/ticket-categories', params=params)
        return response.json()

    def get_ticket_category_tree(self) -> Dict:
        """获取工单分类树"""
        response = self.get('/api/v1/ticket-categories/tree')
        return response.json()

    # ==================== 工单标签接口 ====================

    def get_ticket_tags(self, params: Dict = None) -> Dict:
        """获取工单标签列表"""
        response = self.get('/api/v1/ticket-tags', params=params)
        return response.json()

    # ==================== 工单视图接口 ====================

    def get_ticket_views(self, params: Dict = None) -> Dict:
        """获取工单视图列表"""
        response = self.get('/api/v1/tickets/views', params=params)
        return response.json()

    # ==================== 自动化规则接口 ====================

    def get_automation_rules(self, params: Dict = None) -> Dict:
        """获取自动化规则列表"""
        response = self.get('/api/v1/tickets/automation-rules', params=params)
        return response.json()

    # ==================== 报表接口 ====================

    def get_ticket_analytics(self, params: Dict = None) -> Dict:
        """获取工单分析数据"""
        response = self.get('/api/v1/tickets/analytics', params=params)
        return response.json()

    def export_tickets(self, params: Dict = None) -> Dict:
        """导出工单"""
        response = self.post('/api/v1/tickets/export', json=params or {})
        return response.json()

    def get_ticket_stats(self) -> Dict:
        """获取工单统计"""
        response = self.get('/api/v1/tickets/stats')
        return response.json()

    # ==================== 问题调查接口 ====================

    def get_problem_investigations(self, params: Dict = None) -> Dict:
        """获取问题调查列表（按ID获取）"""
        # 问题调查通过问题ID关联，这里返回空列表占位
        response = self.get('/api/v1/problem-investigation/investigations/1', params=params)
        return response.json()

    # ==================== 服务请求接口 ====================

    def get_service_requests(self, params: Dict = None) -> Dict:
        """获取服务请求列表"""
        response = self.get('/api/v1/service-requests', params=params)
        return response.json()

    # ==================== 组织架构接口 ====================

    def get_departments(self, params: Dict = None) -> Dict:
        """获取部门列表（树形结构）"""
        response = self.get('/api/v1/org/departments/tree', params=params)
        return response.json()

    def get_teams(self, params: Dict = None) -> Dict:
        """获取团队列表"""
        response = self.get('/api/v1/org/teams', params=params)
        return response.json()


class APIResponse:
    """API响应封装"""

    def __init__(self, response: requests.Response):
        self.response = response
        self.status_code = response.status_code
        self.headers = dict(response.headers)

        try:
            self.json = response.json()
            self.data = self.json.get('data')
            self.code = self.json.get('code', 0)
            self.message = self.json.get('message', '')
        except json.JSONDecodeError:
            self.json = {}
            self.data = None
            self.code = -1
            self.message = response.text

    @property
    def is_success(self) -> bool:
        """是否成功"""
        return self.status_code == 200 and self.code == 0

    @property
    def is_client_error(self) -> bool:
        """是否客户端错误"""
        return 400 <= self.status_code < 500

    @property
    def is_server_error(self) -> bool:
        """是否服务端错误"""
        return self.status_code >= 500


def create_api_client(token: str = None) -> ITSMAPIClient:
    """创建ITSM API客户端"""
    return ITSMAPIClient(token=token)
