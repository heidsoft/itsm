# -*- coding: utf-8 -*-
"""
ITSM自动化测试框架
ITSM Automated Test Framework

提供:
- 配置管理
- API测试客户端
- 数据库测试工具
- UI测试封装
- 报告生成
"""

import os
import configparser
import logging
from pathlib import Path
from typing import Any, Dict, Optional

# 项目根目录
PROJECT_ROOT = Path(__file__).parent.parent
CONFIG_FILE = PROJECT_ROOT / "tests" / "config.ini"


class Config:
    """配置管理类"""

    _instance = None
    _config = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._load_config()
        return cls._instance

    def _load_config(self):
        """加载配置文件"""
        self._config = configparser.ConfigParser()
        if CONFIG_FILE.exists():
            self._config.read(CONFIG_FILE, encoding='utf-8')
        else:
            # 使用默认配置
            self._config['api'] = {
                'base_url': os.getenv('ITSM_API_URL', 'http://localhost:8080'),
                'timeout': '30'
            }
            self._config['database'] = {
                'host': os.getenv('ITSM_DB_HOST', 'localhost'),
                'port': os.getenv('ITSM_DB_PORT', '5432'),
                'database': os.getenv('ITSM_DB_NAME', 'itsm'),
                'user': os.getenv('ITSM_DB_USER', 'postgres'),
                'password': os.getenv('ITSM_DB_PASSWORD', 'password')
            }
            self._config['ui'] = {
                'base_url': os.getenv('ITSM_UI_URL', 'http://localhost:3000'),
                'headless': 'true'
            }

    def get(self, section: str, key: str, fallback: Any = None) -> Any:
        """获取配置项"""
        return self._config.get(section, key, fallback=fallback)

    def get_int(self, section: str, key: str, fallback: int = 0) -> int:
        """获取整数配置"""
        return self._config.getint(section, key, fallback=fallback)

    def get_bool(self, section: str, key: str, fallback: bool = False) -> bool:
        """获取布尔配置"""
        return self._config.getboolean(section, key, fallback=fallback)

    def get_dict(self, section: str) -> Dict[str, str]:
        """获取字典配置"""
        if section in self._config:
            return dict(self._config[section])
        return {}


class Logger:
    """日志管理类"""

    _loggers = {}

    @staticmethod
    def get_logger(name: str = 'itsm', level: str = None) -> logging.Logger:
        """获取日志记录器"""
        if name in Logger._loggers:
            return Logger._loggers[name]

        logger = logging.getLogger(name)

        # 默认日志级别
        if level is None:
            config = Config()
            level = config.get('logging', 'level', fallback='INFO')

        logger.setLevel(getattr(logging, level.upper()))

        # 控制台处理器
        if not logger.handlers:
            console_handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            console_handler.setFormatter(formatter)
            logger.addHandler(console_handler)

        Logger._loggers[name] = logger
        return logger


class TestContext:
    """测试上下文，管理测试共享数据"""

    def __init__(self):
        self.config = Config()
        self.logger = Logger.get_logger('context')
        self._data = {}
        self._token = None
        self._tenant_id = None

    def set(self, key: str, value: Any):
        """设置测试数据"""
        self._data[key] = value

    def get(self, key: str, default: Any = None) -> Any:
        """获取测试数据"""
        return self._data.get(key, default)

    def set_token(self, token: str):
        """设置认证token"""
        self._token = token
        self.logger.info("Authentication token set")

    def get_token(self) -> Optional[str]:
        """获取认证token"""
        return self._token

    def set_tenant_id(self, tenant_id: int):
        """设置租户ID"""
        self._tenant_id = tenant_id

    def get_tenant_id(self) -> Optional[int]:
        """获取租户ID"""
        return self._tenant_id

    def clear(self):
        """清理测试数据"""
        self._data.clear()
        self._token = None
        self._tenant_id = None


# 全局测试上下文
context = TestContext()


def get_config() -> Config:
    """获取配置实例"""
    return Config()


def get_logger(name: str = 'itsm') -> logging.Logger:
    """获取日志记录器"""
    return Logger.get_logger(name)
