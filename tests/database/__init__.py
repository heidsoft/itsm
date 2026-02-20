# -*- coding: utf-8 -*-
"""
数据库测试工具
Database Test Utilities
"""

import psycopg2
from psycopg2.extras import RealDictCursor
from typing import Any, Dict, List, Optional
from contextlib import contextmanager

from .. import get_config, get_logger


class DatabaseClient:
    """数据库测试客户端"""

    def __init__(self):
        self.config = get_config()
        self.logger = get_logger('database')
        self._connection = None
        self._connect()

    def _connect(self):
        """建立数据库连接"""
        try:
            self._connection = psycopg2.connect(
                host=self.config.get('database', 'host', 'localhost'),
                port=self.config.get_int('database', 'port', 5432),
                database=self.config.get('database', 'database', 'itsm'),
                user=self.config.get('database', 'user', 'postgres'),
                password=self.config.get('database', 'password', 'password'),
                cursor_factory=RealDictCursor
            )
            self.logger.info("Database connected successfully")
        except psycopg2.Error as e:
            self.logger.error(f"Database connection failed: {e}")
            raise

    @contextmanager
    def cursor(self):
        """获取数据库游标"""
        cursor = self._connection.cursor()
        try:
            yield cursor
            self._connection.commit()
        except Exception as e:
            self._connection.rollback()
            self.logger.error(f"Database operation failed: {e}")
            raise
        finally:
            cursor.close()

    def execute(self, query: str, params: tuple = None) -> List[Dict]:
        """执行查询并返回结果"""
        with self.cursor() as cursor:
            cursor.execute(query, params)
            if query.strip().upper().startswith(('SELECT', 'WITH', 'RETURNING')):
                return cursor.fetchall()
            return []

    def execute_one(self, query: str, params: tuple = None) -> Optional[Dict]:
        """执行查询并返回单条结果"""
        results = self.execute(query, params)
        return results[0] if results else None

    def execute_insert(self, query: str, params: tuple = None) -> Any:
        """执行插入并返回ID"""
        with self.cursor() as cursor:
            cursor.execute(query, params)
            # 获取最后插入的ID
            cursor.execute("SELECT lastval() as id")
            result = cursor.fetchone()
            return result['id'] if result else None

    def table_exists(self, table_name: str) -> bool:
        """检查表是否存在"""
        query = """
            SELECT EXISTS (
                SELECT FROM information_schema.tables
                WHERE table_schema = 'public'
                AND table_name = %s
            )
        """
        result = self.execute_one(query, (table_name,))
        return result['exists'] if result else False

    def get_table_count(self, table_name: str) -> int:
        """获取表记录数"""
        query = f"SELECT COUNT(*) as count FROM {table_name}"
        result = self.execute_one(query)
        return result['count'] if result else 0

    def get_table_columns(self, table_name: str) -> List[Dict]:
        """获取表列信息"""
        query = """
            SELECT
                column_name,
                data_type,
                is_nullable,
                column_default
            FROM information_schema.columns
            WHERE table_schema = 'public'
            AND table_name = %s
            ORDER BY ordinal_position
        """
        return self.execute(query, (table_name,))

    def clear_table(self, table_name: str):
        """清空表数据"""
        with self.cursor() as cursor:
            cursor.execute(f"TRUNCATE TABLE {table_name} CASCADE")

    def close(self):
        """关闭连接"""
        if self._connection:
            self._connection.close()
            self.logger.info("Database connection closed")


class ITSMDBClient(DatabaseClient):
    """ITSM数据库客户端 - 封装常用业务查询"""

    # ==================== 工单相关 ====================

    def get_all_tickets(self, limit: int = 100) -> List[Dict]:
        """获取所有工单"""
        query = "SELECT * FROM tickets ORDER BY created_at DESC LIMIT %s"
        return self.execute(query, (limit,))

    def get_ticket_by_id(self, ticket_id: int) -> Optional[Dict]:
        """根据ID获取工单"""
        query = "SELECT * FROM tickets WHERE id = %s"
        return self.execute_one(query, (ticket_id,))

    def get_ticket_by_number(self, ticket_number: str) -> Optional[Dict]:
        """根据编号获取工单"""
        query = "SELECT * FROM tickets WHERE ticket_number = %s"
        return self.execute_one(query, (ticket_number,))

    def count_tickets_by_status(self) -> List[Dict]:
        """按状态统计工单"""
        query = """
            SELECT status, COUNT(*) as count
            FROM tickets
            GROUP BY status
            ORDER BY count DESC
        """
        return self.execute(query)

    def count_tickets_by_priority(self) -> List[Dict]:
        """按优先级统计工单"""
        query = """
            SELECT priority, COUNT(*) as count
            FROM tickets
            GROUP BY priority
            ORDER BY count DESC
        """
        return self.execute(query)

    # ==================== 用户相关 ====================

    def get_all_users(self, limit: int = 100) -> List[Dict]:
        """获取所有用户"""
        query = "SELECT id, username, name, email, role, status FROM users ORDER BY id LIMIT %s"
        return self.execute(query, (limit,))

    def get_user_by_username(self, username: str) -> Optional[Dict]:
        """根据用户名获取用户"""
        query = "SELECT * FROM users WHERE username = %s"
        return self.execute_one(query, (username,))

    def get_users_by_role(self, role: str) -> List[Dict]:
        """根据角色获取用户"""
        query = "SELECT * FROM users WHERE role = %s"
        return self.execute(query, (role,))

    # ==================== SLA相关 ====================

    def get_all_sla_definitions(self) -> List[Dict]:
        """获取所有SLA定义"""
        query = "SELECT * FROM sla_definitions ORDER BY priority"
        return self.execute(query)

    def get_active_sla_definitions(self) -> List[Dict]:
        """获取激活的SLA定义"""
        query = "SELECT * FROM sla_definitions WHERE is_active = true ORDER BY priority"
        return self.execute(query)

    def get_sla_violations(self, limit: int = 100) -> List[Dict]:
        """获取SLA违规记录"""
        query = """
            SELECT sv.*, t.ticket_number, t.title as ticket_title
            FROM sla_violations sv
            LEFT JOIN tickets t ON sv.ticket_id = t.id
            ORDER BY sv.created_at DESC
            LIMIT %s
        """
        return self.execute(query, (limit,))

    # ==================== 部门团队相关 ====================

    def get_all_departments(self) -> List[Dict]:
        """获取所有部门"""
        query = "SELECT * FROM departments ORDER BY name"
        return self.execute(query)

    def get_all_teams(self) -> List[Dict]:
        """获取所有团队"""
        query = "SELECT * FROM teams ORDER BY name"
        return self.execute(query)

    # ==================== 工作流相关 ====================

    def get_approval_workflows(self) -> List[Dict]:
        """获取审批工作流"""
        query = "SELECT * FROM approval_workflows ORDER BY name"
        return self.execute(query)

    def get_process_bindings(self) -> List[Dict]:
        """获取流程绑定"""
        query = "SELECT * FROM process_bindings ORDER BY business_type, priority"
        return self.execute(query)

    # ==================== 知识库相关 ====================

    def get_knowledge_articles(self, limit: int = 100) -> List[Dict]:
        """获取知识库文章"""
        query = "SELECT * FROM knowledge_articles ORDER BY created_at DESC LIMIT %s"
        return self.execute(query, (limit,))

    def search_knowledge_articles(self, keyword: str) -> List[Dict]:
        """搜索知识库文章"""
        query = """
            SELECT * FROM knowledge_articles
            WHERE title ILIKE %s OR content ILIKE %s
            ORDER BY created_at DESC
        """
        pattern = f'%{keyword}%'
        return self.execute(query, (pattern, pattern))

    # ==================== 统计相关 ====================

    def get_ticket_growth_trend(self, days: int = 30) -> List[Dict]:
        """获取工单增长趋势"""
        query = """
            SELECT
                DATE(created_at) as date,
                COUNT(*) as count
            FROM tickets
            WHERE created_at >= CURRENT_DATE - INTERVAL '%s days'
            GROUP BY DATE(created_at)
            ORDER BY date
        """
        return self.execute(query, (days,))

    def get_resolution_time_stats(self) -> Dict:
        """获取解决时间统计"""
        query = """
            SELECT
                AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) as avg_hours,
                MIN(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) as min_hours,
                MAX(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) as max_hours
            FROM tickets
            WHERE resolved_at IS NOT NULL
        """
        return self.execute_one(query)

    # ==================== 系统相关 ====================

    def get_table_list(self) -> List[Dict]:
        """获取所有表"""
        query = """
            SELECT table_name
            FROM information_schema.tables
            WHERE table_schema = 'public'
            AND table_type = 'BASE TABLE'
            ORDER BY table_name
        """
        return self.execute(query)

    def get_database_stats(self) -> Dict:
        """获取数据库统计信息"""
        tables = self.get_table_list()
        stats = {
            'total_tables': len(tables),
            'table_counts': {}
        }

        for table in tables:
            table_name = table['table_name']
            try:
                count = self.get_table_count(table_name)
                stats['table_counts'][table_name] = count
            except Exception:
                stats['table_counts'][table_name] = 0

        return stats


def create_db_client() -> ITSMDBClient:
    """创建数据库客户端"""
    return ITSMDBClient()
