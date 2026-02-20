# -*- coding: utf-8 -*-
"""
测试运行器
Test Runner
"""

import os
import sys
import subprocess
import argparse
from pathlib import Path
from datetime import datetime


class TestRunner:
    """测试运行器"""

    def __init__(self):
        self.project_root = Path(__file__).parent.parent
        self.tests_dir = self.project_root / "tests"
        self.reports_dir = self.tests_dir / "reports"
        self.reports_dir.mkdir(exist_ok=True)

    def install_dependencies(self):
        """安装测试依赖"""
        print("Installing test dependencies...")
        requirements = self.tests_dir / "requirements.txt"

        cmd = [sys.executable, "-m", "pip", "install", "-r", str(requirements)]
        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode == 0:
            print("Dependencies installed successfully")
        else:
            print(f"Failed to install dependencies: {result.stderr}")
            sys.exit(1)

    def install_playwright(self):
        """安装Playwright浏览器"""
        print("Installing Playwright browsers...")
        cmd = [sys.executable, "-m", "playwright", "install", "chromium", "firefox", "webkit"]
        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode == 0:
            print("Playwright browsers installed")
        else:
            print(f"Note: {result.stderr}")

    def run_tests(
        self,
        test_type: str = "all",
        markers: str = None,
        keywords: str = None,
        parallel: bool = False,
        count: int = 1,
        verbose: bool = True
    ):
        """运行测试"""

        # 构建pytest命令
        cmd = [sys.executable, "-m", "pytest"]

        # 测试路径
        if test_type == "api":
            cmd.append(str(self.tests_dir / "api"))
        elif test_type == "ui":
            cmd.append(str(self.tests_dir / "ui"))
        elif test_type == "database":
            cmd.append(str(self.tests_dir / "database"))
        else:
            cmd.append(str(self.tests_dir))

        # 标记过滤
        if markers:
            cmd.extend(["-m", markers])

        # 关键词过滤
        if keywords:
            cmd.extend(["-k", keywords])

        # 并行执行
        if parallel:
            cmd.extend(["-n", "auto"])

        # 重复执行
        if count > 1:
            cmd.extend([f"--count={count}"])

        # 详细输出
        if verbose:
            cmd.append("-v")

        # 生成HTML报告
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = self.reports_dir / f"report_{test_type}_{timestamp}.html"
        cmd.extend([f"--html={report_file}", "--self-contained-html"])

        # JUnit XML报告
        xml_file = self.reports_dir / f"junit_{test_type}_{timestamp}.xml"
        cmd.extend([f"--junit-xml={xml_file}"])

        print(f"Running tests: {' '.join(cmd)}")
        print(f"Report will be saved to: {report_file}")

        # 执行测试
        result = subprocess.run(cmd, cwd=str(self.project_root))

        return result.returncode

    def run_smoke_tests(self):
        """运行冒烟测试"""
        return self.run_tests(markers="smoke", verbose=True)

    def run_regression_tests(self):
        """运行回归测试"""
        return self.run_tests(markers="regression", parallel=True)

    def run_api_tests(self):
        """运行API测试"""
        return self.run_tests(test_type="api", verbose=True)

    def run_ui_tests(self):
        """运行UI测试"""
        return self.run_tests(test_type="ui", verbose=True)

    def run_critical_tests(self):
        """运行关键路径测试"""
        return self.run_tests(markers="critical", verbose=True)


def main():
    """主函数"""
    parser = argparse.ArgumentParser(description="ITSM Test Runner")

    parser.add_argument(
        "--type",
        choices=["all", "api", "ui", "database"],
        default="all",
        help="Test type to run"
    )

    parser.add_argument(
        "--markers", "-m",
        help="pytest markers to filter tests"
    )

    parser.add_argument(
        "--keyword", "-k",
        help="Keyword to filter tests"
    )

    parser.add_argument(
        "--parallel", "-p",
        action="store_true",
        help="Run tests in parallel"
    )

    parser.add_argument(
        "--count", "-c",
        type=int,
        default=1,
        help="Number of times to repeat tests"
    )

    parser.add_argument(
        "--install",
        action="store_true",
        help="Install dependencies before running tests"
    )

    parser.add_argument(
        "--install-playwright",
        action="store_true",
        help="Install Playwright browsers"
    )

    parser.add_argument(
        "command",
        nargs="*",
        help="Additional pytest commands"
    )

    args = parser.parse_args()

    runner = TestRunner()

    # 安装依赖
    if args.install:
        runner.install_dependencies()

    # 安装Playwright
    if args.install_playwright:
        runner.install_playwright()

    # 运行测试
    if args.command:
        # 直接传递pytest命令
        cmd = [sys.executable, "-m", "pytest"] + args.command
        result = subprocess.run(cmd, cwd=str(runner.project_root))
        sys.exit(result.returncode)
    else:
        # 使用配置运行测试
        exit_code = runner.run_tests(
            test_type=args.type,
            markers=args.markers,
            keywords=args.keyword,
            parallel=args.parallel,
            count=args.count
        )
        sys.exit(exit_code)


if __name__ == "__main__":
    main()
