'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  Github,
  Ticket,
  Database,
  BookOpen,
  Workflow,
  Gauge,
  BrainCircuit,
  Terminal,
  ArrowRight,
  CheckCircle2,
  Server,
} from 'lucide-react';

/**
 * AI-Native ITSM 项目介绍页
 * 开源项目主页，展示项目信息、快速开始和核心功能亮点
 */
export default function HomePage() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  useEffect(() => {
    // Token 由后端 cookie 管理；不要从 localStorage/sessionStorage 读取 token。
    const checkAuth = () => {
      try {
        const cookieToken =
          document.cookie
            .split('; ')
            .find(row => row.startsWith('auth-token=') || row.startsWith('access_token='))
            ?.split('=')[1] ?? null;
        setIsLoggedIn(!!cookieToken);
      } catch {
        setIsLoggedIn(false);
      }
    };
    checkAuth();
  }, []);

  const features = [
    {
      icon: <Ticket className="w-6 h-6" />,
      title: '工单管理',
      description: '完整的工单生命周期管理，支持多级分类、优先级、SLA关联与自动分配。',
    },
    {
      icon: <Database className="w-6 h-6" />,
      title: 'CMDB',
      description: '配置管理数据库，支持自定义 CI 类型、关系拓扑图与变更追溯。',
    },
    {
      icon: <BookOpen className="w-6 h-6" />,
      title: '知识库 RAG',
      description: '基于 RAG 的智能知识库，支持文档检索、向量相似度匹配与 AI 问答。',
    },
    {
      icon: <Workflow className="w-6 h-6" />,
      title: 'BPMN 工作流',
      description: '集成 BPMN 2.0 流程引擎，支持可视化流程设计与自动化审批流转。',
    },
    {
      icon: <Gauge className="w-6 h-6" />,
      title: 'SLA 监控',
      description: '服务级别协议实时监控，自动告警与升级机制，保障服务质量达标。',
    },
    {
      icon: <BrainCircuit className="w-6 h-6" />,
      title: 'AI 智能分诊',
      description: 'AI 驱动的工单智能分诊与分类，自动推荐处理人与解决方案。',
    },
  ];

  const techStack = [
    'Next.js 15',
    'React 19',
    'Ant Design v6',
    'Tailwind CSS v4',
    'Go (后端)',
    'PostgreSQL',
    'Redis',
    'MinIO',
  ];

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
      {/* Navigation */}
      <nav className="sticky top-0 z-50 bg-white/90 backdrop-blur-md border-b border-gray-100">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 bg-gradient-to-br from-blue-600 to-indigo-600 rounded-lg flex items-center justify-center text-white font-bold text-sm">
                AI
              </div>
              <span className="text-lg font-bold text-gray-900">AI-Native ITSM</span>
            </div>
            <div className="flex items-center gap-4">
              <Link
                href="https://github.com/heidsoft/tism"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1.5 text-gray-600 hover:text-gray-900 transition-colors"
              >
                <Github className="w-5 h-5" />
                <span className="hidden sm:inline text-sm font-medium">GitHub</span>
              </Link>
              {isLoggedIn ? (
                <Link
                  href="/dashboard"
                  className="flex items-center gap-1.5 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
                >
                  进入系统
                  <ArrowRight className="w-4 h-4" />
                </Link>
              ) : (
                <Link
                  href="/login"
                  className="flex items-center gap-1.5 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
                >
                  登录
                  <ArrowRight className="w-4 h-4" />
                </Link>
              )}
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="relative overflow-hidden">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-20 lg:py-28">
          <div className="text-center max-w-4xl mx-auto">
            <div className="inline-flex items-center gap-2 bg-blue-50 border border-blue-100 px-4 py-1.5 rounded-full mb-6">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-sm text-blue-700 font-medium">开源 · AI 驱动 · 开箱即用</span>
            </div>
            <h1 className="text-4xl md:text-6xl font-bold text-gray-900 mb-6 leading-tight">
              AI-Native ITSM
            </h1>
            <p className="text-lg md:text-xl text-gray-600 mb-8 max-w-2xl mx-auto">
              一款开源的 AI 驱动 IT 服务管理系统，涵盖工单管理、CMDB、知识库 RAG、
              BPMN 工作流、SLA 监控与 AI 智能分诊，帮助企业高效管理 IT 服务全流程。
            </p>
            <div className="flex flex-col sm:flex-row justify-center gap-4 mb-12">
              <Link
                href="https://github.com/heidsoft/tism"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-center gap-2 bg-gray-900 text-white px-6 py-3 rounded-lg font-medium hover:bg-gray-800 transition-colors"
              >
                <Github className="w-5 h-5" />
                查看源码
              </Link>
              {isLoggedIn ? (
                <Link
                  href="/dashboard"
                  className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >
                  进入系统
                  <ArrowRight className="w-5 h-5" />
                </Link>
              ) : (
                <Link
                  href="/login"
                  className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >
                  立即登录
                  <ArrowRight className="w-5 h-5" />
                </Link>
              )}
            </div>

            {/* Quick Stats */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6 max-w-3xl mx-auto pt-8 border-t border-gray-100">
              {[
                { label: '开源协议', value: 'MIT' },
                { label: '技术栈', value: 'Next.js + Go' },
                { label: 'AI 能力', value: 'RAG + LLM' },
                { label: '部署方式', value: 'Docker' },
              ].map((stat, index) => (
                <div key={index} className="text-center">
                  <div className="text-xl md:text-2xl font-bold text-gray-900 mb-1">
                    {stat.value}
                  </div>
                  <div className="text-sm text-gray-500">{stat.label}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 bg-gray-50">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">核心功能</h2>
            <p className="text-lg text-gray-600">覆盖 IT 服务管理全场景的六大核心能力</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 max-w-6xl mx-auto">
            {features.map((feature, index) => (
              <div
                key={index}
                className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm hover:shadow-md hover:border-blue-200 transition-all"
              >
                <div className="w-12 h-12 bg-blue-50 rounded-lg flex items-center justify-center text-blue-600 mb-4">
                  {feature.icon}
                </div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">{feature.title}</h3>
                <p className="text-sm text-gray-600 leading-relaxed">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Quick Start Section */}
      <section className="py-20">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">快速开始</h2>
            <p className="text-lg text-gray-600">使用 Docker Compose 一键启动完整环境</p>
          </div>

          <div className="max-w-3xl mx-auto">
            <div className="bg-gray-900 rounded-xl overflow-hidden shadow-xl">
              <div className="flex items-center gap-2 px-4 py-3 bg-gray-800 border-b border-gray-700">
                <div className="flex gap-1.5">
                  <div className="w-3 h-3 bg-red-500 rounded-full" />
                  <div className="w-3 h-3 bg-yellow-500 rounded-full" />
                  <div className="w-3 h-3 bg-green-500 rounded-full" />
                </div>
                <div className="flex items-center gap-1.5 text-gray-400 text-xs ml-2">
                  <Terminal className="w-3.5 h-3.5" />
                  <span>终端</span>
                </div>
              </div>
              <div className="p-6 font-mono text-sm">
                <div className="text-gray-500 mb-1"># 克隆仓库</div>
                <div className="text-green-400 mb-3">
                  $ git clone https://github.com/heidsoft/tism.git
                </div>
                <div className="text-gray-500 mb-1"># 进入项目目录</div>
                <div className="text-green-400 mb-3">$ cd tism</div>
                <div className="text-gray-500 mb-1"># 使用 Docker Compose 启动</div>
                <div className="text-green-400 mb-3">$ docker compose up -d --build</div>
                <div className="text-gray-500 mb-1"># 访问应用</div>
                <div className="text-blue-400">$ open http://localhost:3000</div>
              </div>
            </div>

            <div className="mt-6 flex items-start gap-3 bg-blue-50 border border-blue-100 rounded-lg p-4">
              <CheckCircle2 className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-blue-800">
                <p className="font-medium mb-1">默认管理员账户</p>
                <p>
                  首次启动后，使用 <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">admin</code> /{' '}
                  <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">admin123</code> 登录系统，请及时修改密码。
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Tech Stack Section */}
      <section className="py-16 bg-gray-50">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold text-gray-900 mb-4">技术栈</h2>
            <p className="text-gray-600">现代化的全栈技术方案</p>
          </div>
          <div className="flex flex-wrap justify-center gap-3 max-w-3xl mx-auto">
            {techStack.map((tech, index) => (
              <div
                key={index}
                className="flex items-center gap-2 bg-white border border-gray-200 rounded-full px-4 py-2 text-sm font-medium text-gray-700"
              >
                <Server className="w-4 h-4 text-blue-500" />
                {tech}
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-12">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-indigo-500 rounded-lg flex items-center justify-center text-white font-bold text-sm">
                AI
              </div>
              <span className="text-lg font-bold">AI-Native ITSM</span>
            </div>
            <div className="flex items-center gap-6">
              <Link
                href="https://github.com/heidsoft/tism"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1.5 text-gray-400 hover:text-white transition-colors"
              >
                <Github className="w-5 h-5" />
                <span className="text-sm">GitHub</span>
              </Link>
              <Link
                href="https://github.com/heidsoft/tism/issues"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white transition-colors text-sm"
              >
                问题反馈
              </Link>
              <Link
                href="https://github.com/heidsoft/tism/blob/main/LICENSE"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white transition-colors text-sm"
              >
                MIT License
              </Link>
            </div>
          </div>
          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-gray-500 text-sm">
            <p>&copy; {new Date().getFullYear()} AI-Native ITSM. Open Source under MIT License.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
