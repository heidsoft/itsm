# -*- coding: utf-8 -*-
"""
测试数据fixtures
Test Data Fixtures
"""

import uuid
import random
from datetime import datetime, timedelta
from typing import Dict, List, Any


class TicketFixture:
    """工单测试数据生成器"""

    STATUSES = ['new', 'open', 'in_progress', 'pending', 'resolved', 'closed']
    PRIORITIES = ['low', 'medium', 'high', 'urgent']
    TYPES = ['incident', 'service_request', 'problem', 'change']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成工单数据"""
        data = {
            'title': f'Test Ticket {uuid.uuid4().hex[:8]}',
            'description': 'This is a test ticket created by automated tests.',
            'priority': random.choice(cls.PRIORITIES),
            'status': 'new',
            'type': random.choice(cls.TYPES),
            'category_id': random.randint(1, 5),
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data

    @classmethod
    def create_batch(cls, count: int) -> List[Dict]:
        """批量生成工单数据"""
        return [cls.create() for _ in range(count)]


class UserFixture:
    """用户测试数据生成器"""

    ROLES = ['end_user', 'agent', 'admin', 'super_admin']
    DEPARTMENTS = ['IT', 'HR', 'Finance', 'Operations']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成用户数据"""
        username = f'testuser_{uuid.uuid4().hex[:6]}'
        data = {
            'username': username,
            'email': f'{username}@test.com',
            'name': f'Test User {username}',
            'password': 'Test123456',
            'role': random.choice(cls.ROLES),
            'department': random.choice(cls.DEPARTMENTS),
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


class KnowledgeArticleFixture:
    """知识库文章测试数据生成器"""

    CATEGORIES = ['FAQ', 'Troubleshooting', 'How-To', 'Policy', 'Tutorial']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成知识库文章数据"""
        data = {
            'title': f'Test Article {uuid.uuid4().hex[:8]}',
            'content': '# Test Article\n\nThis is test content for the knowledge base article.',
            'category': random.choice(cls.CATEGORIES),
            'status': 'published',
            'tags': ['test', 'automated'],
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


class SLAFixture:
    """SLA测试数据生成器"""

    SERVICE_TYPES = ['IT Support', 'HR Services', 'Finance', 'Facilities']
    PRIORITIES = ['low', 'medium', 'high', 'critical']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成SLA定义数据"""
        data = {
            'name': f'Test SLA {uuid.uuid4().hex[:6]}',
            'description': 'Test SLA for automated testing',
            'service_type': random.choice(cls.SERVICE_TYPES),
            'priority': random.choice(cls.PRIORITIES),
            'response_time_minutes': random.choice([15, 30, 60, 240]),
            'resolution_time_minutes': random.choice([60, 240, 480, 1440]),
            'is_active': True,
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


class IncidentFixture:
    """事件测试数据生成器"""

    IMPACTS = ['low', 'medium', 'high', 'critical']
    URGENCIES = ['low', 'medium', 'high', 'critical']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成事件数据"""
        data = {
            'title': f'Test Incident {uuid.uuid4().hex[:8]}',
            'description': 'This is a test incident created by automated tests.',
            'impact': random.choice(cls.IMPACTS),
            'urgency': random.choice(cls.URGENCIES),
            'status': 'new',
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


class ProblemFixture:
    """问题测试数据生成器"""

    STATUSES = ['new', 'investigating', 'identified', 'resolved', 'closed']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成问题数据"""
        data = {
            'title': f'Test Problem {uuid.uuid4().hex[:8]}',
            'description': 'This is a test problem created by automated tests.',
            'status': random.choice(cls.STATUSES),
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


class ChangeFixture:
    """变更测试数据生成器"""

    TYPES = ['standard', 'normal', 'emergency']
    RISKS = ['low', 'medium', 'high']
    STATUSES = ['draft', 'pending_approval', 'approved', 'scheduled', 'in_progress', 'completed', 'rejected']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成变更数据"""
        data = {
            'title': f'Test Change {uuid.uuid4().hex[:8]}',
            'description': 'This is a test change created by automated tests.',
            'change_type': random.choice(cls.TYPES),
            'risk_level': random.choice(cls.RISKS),
            'status': random.choice(cls.STATUSES),
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


class ServiceCatalogFixture:
    """服务目录测试数据生成器"""

    CATEGORIES = ['Hardware', 'Software', 'Network', 'Security', 'Support']

    @classmethod
    def create(cls, overrides: Dict = None) -> Dict:
        """生成服务目录数据"""
        data = {
            'name': f'Test Service {uuid.uuid4().hex[:6]}',
            'description': 'This is a test service for automated testing.',
            'category': random.choice(cls.CATEGORIES),
            'is_active': True,
            'tenant_id': 1
        }

        if overrides:
            data.update(overrides)

        return data


# Fixtures集合
FIXTURES = {
    'ticket': TicketFixture,
    'user': UserFixture,
    'knowledge_article': KnowledgeArticleFixture,
    'sla': SLAFixture,
    'incident': IncidentFixture,
    'problem': ProblemFixture,
    'change': ChangeFixture,
    'service_catalog': ServiceCatalogFixture,
}


def get_fixture(name: str):
    """获取fixture类"""
    return FIXTURES.get(name)
