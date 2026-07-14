#!/usr/bin/env python3
"""
ITSM系统集成测试示例
测试新增的API功能
"""

import requests
import json
import time
from datetime import datetime, timedelta

BASE_URL = "http://localhost:8090/api/v1"

class TestLicenseUtilization:
    """许可证使用率测试"""

    def __init__(self, token):
        self.token = token
        self.headers = {"Authorization": f"Bearer {token}"}

    def test_get_utilization_report(self):
        """测试获取使用率报告"""
        print("\n=== 测试许可证使用率报告 ===")

        response = requests.get(
            f"{BASE_URL}/licenses/utilization",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()

        print(f"总许可证数: {data['data']['total_licenses']}")
        print(f"最优状态数: {data['data']['optimal_count']}")
        print(f"警告状态数: {data['data']['warning_count']}")
        print(f"严重状态数: {data['data']['critical_count']}")
        print(f"超额状态数: {data['data']['exceeded_count']}")

        return data['data']

    def test_assign_license(self, license_id, user_id):
        """测试分配许可证"""
        print(f"\n=== 测试分配许可证 {license_id} 给用户 {user_id} ===")

        response = requests.post(
            f"{BASE_URL}/licenses/{license_id}/assign",
            headers=self.headers,
            json={"user_id": user_id}
        )

        assert response.status_code == 200
        data = response.json()
        print(f"结果: {data['data']['message']}")

        return data

    def test_release_license(self, license_id):
        """测试释放许可证"""
        print(f"\n=== 测试释放许可证 {license_id} ===")

        response = requests.post(
            f"{BASE_URL}/licenses/{license_id}/release",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()
        print(f"结果: {data['data']['message']}")

        return data

    def test_get_expiring_licenses(self, days=30):
        """测试获取即将过期许可证"""
        print(f"\n=== 测试获取 {days} 天内过期许可证 ===")

        response = requests.get(
            f"{BASE_URL}/licenses/expiring?days={days}",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()
        print(f"找到 {len(data['data'])} 个即将过期许可证")

        for license_data in data['data']:
            print(f"  - {license_data['license_name']} (过期时间: {license_data['expiry_date']})")

        return data['data']


class TestEnhancedSearch:
    """增强搜索测试"""

    def __init__(self, token):
        self.token = token
        self.headers = {"Authorization": f"Bearer {token}"}

    def test_basic_search(self, keyword):
        """测试基础搜索"""
        print(f"\n=== 测试基础搜索: {keyword} ===")

        response = requests.get(
            f"{BASE_URL}/search/enhanced?keyword={keyword}",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()

        print(f"找到 {len(data['data'])} 条结果")
        for result in data['data'][:5]:  # 只显示前5条
            print(f"  [{result['type']}] {result['title']} - {result['status']}")

        return data['data']

    def test_filtered_search(self, keyword, types, status):
        """测试过滤搜索"""
        print(f"\n=== 测试过滤搜索: {keyword} (类型: {types}, 状态: {status}) ===")

        params = {
            "keyword": keyword,
            "types": ",".join(types),
            "status": status
        }

        response = requests.get(
            f"{BASE_URL}/search/enhanced",
            params=params,
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()

        print(f"找到 {len(data['data'])} 条结果")
        for result in data['data']:
            print(f"  [{result['type']}] {result['title']} - {result['status']}")

        return data['data']


class TestReleaseAutomation:
    """发布自动化测试"""

    def __init__(self, token):
        self.token = token
        self.headers = {"Authorization": f"Bearer {token}"}

    def test_trigger_deployment(self, release_id):
        """测试触发部署"""
        print(f"\n=== 测试触发发布 {release_id} 部署 ===")

        response = requests.post(
            f"{BASE_URL}/releases/{release_id}/deploy",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()

        pipeline = data['data']
        print(f"发布ID: {pipeline['release_id']}")
        print(f"流水线阶段: {pipeline['pipeline_stage']}")
        print(f"构建状态: {pipeline['build_status']}")

        return pipeline

    def test_get_deployment_status(self, release_id):
        """测试获取部署状态"""
        print(f"\n=== 测试获取发布 {release_id} 部署状态 ===")

        response = requests.get(
            f"{BASE_URL}/releases/{release_id}/deployment-status",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()

        status = data['data']
        print(f"发布ID: {status['release_id']}")
        print(f"当前阶段: {status['pipeline_stage']}")
        print(f"构建状态: {status['build_status']}")

        return status

    def test_schedule_deployment(self, release_id, scheduled_time):
        """测试定时部署"""
        print(f"\n=== 测试定时发布 {release_id} ===")

        response = requests.post(
            f"{BASE_URL}/releases/{release_id}/schedule",
            headers=self.headers,
            json={"scheduled_time": scheduled_time}
        )

        assert response.status_code == 200
        data = response.json()

        print(f"结果: {data['data']['message']}")
        print(f"计划时间: {data['data']['scheduled_time']}")

        return data

    def test_rollback_deployment(self, release_id):
        """测试回滚部署"""
        print(f"\n=== 测试回滚发布 {release_id} ===")

        response = requests.post(
            f"{BASE_URL}/releases/{release_id}/rollback",
            headers=self.headers
        )

        assert response.status_code == 200
        data = response.json()

        print(f"结果: {data['data']['message']}")

        return data


def get_auth_token(username, password):
    """获取认证Token"""
    response = requests.post(
        f"{BASE_URL}/auth/login",
        json={
            "username": username,
            "password": password
        }
    )

    if response.status_code == 200:
        return response.json()['data']['token']
    else:
        raise Exception("登录失败")


def main():
    """主测试函数"""
    print("ITSM系统集成测试")
    print("=" * 50)

    # 获取Token（请替换为实际的用户名和密码）
    try:
        token = get_auth_token("admin", "admin123")
        print("✓ 认证成功")
    except Exception as e:
        print(f"✗ 认证失败: {e}")
        return

    # 测试许可证使用率
    license_test = TestLicenseUtilization(token)
    try:
        license_test.test_get_utilization_report()
        license_test.test_get_expiring_licenses(30)
        # license_test.test_assign_license(1, 123)
        # license_test.test_release_license(1)
    except Exception as e:
        print(f"✗ 许可证测试失败: {e}")

    # 测试增强搜索
    search_test = TestEnhancedSearch(token)
    try:
        search_test.test_basic_search("网络")
        search_test.test_filtered_search(
            "故障",
            types=["ticket", "incident"],
            status="open"
        )
    except Exception as e:
        print(f"✗ 搜索测试失败: {e}")

    # 测试发布自动化
    release_test = TestReleaseAutomation(token)
    try:
        # 注意：需要有效的发布ID
        # release_test.test_trigger_deployment(1)
        # release_test.test_get_deployment_status(1)

        # 测试定时部署
        scheduled_time = (datetime.now() + timedelta(days=1)).isoformat()
        # release_test.test_schedule_deployment(1, scheduled_time)

        print("发布自动化测试跳过（需要有效的发布ID）")
    except Exception as e:
        print(f"✗ 发布自动化测试失败: {e}")

    print("\n" + "=" * 50)
    print("集成测试完成")


if __name__ == "__main__":
    main()
