'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { loadSession } from '@/lib/api/session-api';
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
    let cancelled = false;
    // 只问后端会话端点：httpOnly 凭证 JS 读不到，历史上靠 document.cookie 里是否存在
    // auth-token 标记位判断，结果与真实登录态无关。这里瞬时故障按未登录处理，
    // 影响范围仅是介绍页的「进入系统」入口按钮。
    void loadSession().then(outcome => {
      if (!cancelled) setIsLoggedIn(outcome.state === 'authenticated');
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const features = [
    {
      icon: <Ticket className="w-6 h-6" />,
      title: 'Ticket Management',
      description: 'Complete work order life cycle management, supporting multi-level classification, priority, SLA association and automatic allocation.',
    },
    {
      icon: <Database className="w-6 h-6" />,
      title: 'CMDB',
      description: 'Configuration management database, supporting custom CI types, relationship topology diagrams and change tracing.',
    },
    {
      icon: <BookOpen className="w-6 h-6" />,
      title: 'Knowledge Base RAG',
      description: 'Intelligent knowledge base based on RAG, supporting document retrieval, vector similarity matching and AI question and answer.',
    },
    {
      icon: <Workflow className="w-6 h-6" />,
      title: 'BPMN workflow',
      description: 'Integrate BPMN 2.0 process engine to support visual process design and automated approval flow.',
    },
    {
      icon: <Gauge className="w-6 h-6" />,
      title: 'SLA Monitor',
      description: 'Real-time monitoring of service level agreement, automatic alarm and upgrade mechanism to ensure that service quality meets standards.',
    },
    {
      icon: <BrainCircuit className="w-6 h-6" />,
      title: 'AI-assisted triage',
      description: 'AI-driven work order triage and classification, automatically recommending handlers and solutions.',
    },
  ];

  const techStack = [
    'Next.js 15',
    'React 19',
    'Ant Design v6',
    'Tailwind CSS v4',
    'Go (backend)',
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
                href="https://github.com/heidsoft/itsm"
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

                  Enter the system
                  <ArrowRight className="w-4 h-4" />
                </Link>
              ) : (
                <Link
                  href="/login"
                  className="flex items-center gap-1.5 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
                >

                  Login
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
              <span className="text-sm text-blue-700 font-medium">Open source · AI driven · Ready out of the box</span>
            </div>
            <h1 className="text-4xl md:text-6xl font-bold text-gray-900 mb-6 leading-tight">
              AI-Native ITSM
            </h1>
            <p className="text-lg md:text-xl text-gray-600 mb-8 max-w-2xl mx-auto">

              An open source AI-driven IT service management system, covering work order management, CMDB, knowledge base RAG,
BPMN workflow, SLA monitoring and AI-assisted triage help enterprises efficiently manage the entire IT service process.
            </p>
            <div className="flex flex-col sm:flex-row justify-center gap-4 mb-12">
              <Link
                href="https://github.com/heidsoft/itsm"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-center gap-2 bg-gray-900 text-white px-6 py-3 rounded-lg font-medium hover:bg-gray-800 transition-colors"
              >
                <Github className="w-5 h-5" />

                View source code
              </Link>
              {isLoggedIn ? (
                <Link
                  href="/dashboard"
                  className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >

                  Enter the system
                  <ArrowRight className="w-5 h-5" />
                </Link>
              ) : (
                <Link
                  href="/login"
                  className="flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >

                  Login now
                  <ArrowRight className="w-5 h-5" />
                </Link>
              )}
            </div>

            {/* Quick Stats */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6 max-w-3xl mx-auto pt-8 border-t border-gray-100">
              {[
                { label: 'Open source protocol', value: 'Apache 2.0' },
                { label: 'Technology stack', value: 'Next.js + Go' },
                { label: 'AI Capabilities', value: 'RAG + LLM' },
                { label: 'Deployment method', value: 'Docker' },
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
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">Core Functions</h2>
            <p className="text-lg text-gray-600">Six core capabilities covering all scenarios of IT service management</p>
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
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">Quick start</h2>
            <p className="text-lg text-gray-600">Use Docker Compose to start the complete environment with one click</p>
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
                  <span>Terminal</span>
                </div>
              </div>
              <div className="p-6 font-mono text-sm">
                <div className="text-gray-500 mb-1"># Clone repository</div>
                <div className="text-green-400 mb-3">
                  $ git clone https://github.com/heidsoft/itsm.git
                </div>
                <div className="text-gray-500 mb-1"># Enter the project directory</div>
				<div className="text-green-400 mb-3">$ cd itsm</div>
				<div className="text-gray-500 mb-1"># Start local development environment</div>
				<div className="text-green-400 mb-3">$ cp .env.dev.example .env</div>
				<div className="text-green-400 mb-3">$ make dev-start-docker</div>
                <div className="text-gray-500 mb-1">#Access application</div>
                <div className="text-blue-400">$ open http://localhost:3000</div>
              </div>
            </div>

            <div className="mt-6 flex items-start gap-3 bg-blue-50 border border-blue-100 rounded-lg p-4">
              <CheckCircle2 className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-blue-800">
                {/* 公开落地页对未认证访客可见：演示凭据只在本地开发构建里出现。
                    与登录页 (auth)/login/page.tsx 使用同一套 NODE_ENV 门控；生产口径按
                    README 与 docker-compose.prod.yml：管理员由 itsm-init 依 .env.prod 的
                    ADMIN_PASSWORD 创建，bootstrap token 是可选流程（默认未启用），不是默认模型。 */}
                {process.env.NODE_ENV === 'development' ? (
                  <>
                    <p className="font-medium mb-1">Local development default account</p>
                    <p>

                      Only available in local development environment <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">admin</code> /{' '}
                      <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">admin123</code>. Do not keep this account in any deployment that can be accessed by others.
                    </p>
                  </>
                ) : (
                  <>
                    <p className="font-medium mb-1">Administrator credentials</p>
                    <p>

                      The admin password for the production deployment is given by <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">.env.prod</code>  in{' '}
                      <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">ADMIN_PASSWORD</code>  Decision (by{' '}
                      <code className="bg-blue-100 px-1.5 py-0.5 rounded text-blue-900">itsm-init</code>  Writing initiated for the first time).
Please set a strong password before starting for the first time, and replace JWT, database and Redis credentials simultaneously.
                    </p>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Tech Stack Section */}
      <section className="py-16 bg-gray-50">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold text-gray-900 mb-4">Technology stack</h2>
            <p className="text-gray-600">Modern full-stack technology solution</p>
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
                href="https://github.com/heidsoft/itsm"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1.5 text-gray-400 hover:text-white transition-colors"
              >
                <Github className="w-5 h-5" />
                <span className="text-sm">GitHub</span>
              </Link>
              <Link
                href="https://github.com/heidsoft/itsm/issues"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white transition-colors text-sm"
              >

                Problem feedback
              </Link>
              <Link
                href="https://github.com/heidsoft/itsm/blob/main/LICENSE"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white transition-colors text-sm"
              >
                Apache License 2.0
              </Link>
            </div>
          </div>
          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-gray-500 text-sm">
            <p>&copy; {new Date().getFullYear()} AI-Native ITSM. Open Source under Apache License 2.0.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
