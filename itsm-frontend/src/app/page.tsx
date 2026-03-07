import React from 'react';
import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="min-h-screen">
      {/* Navigation */}
      <nav className="fixed top-0 left-0 right-0 bg-white/95 backdrop-blur-sm shadow-sm z-50">
        <div className="container mx-auto px-4">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-2">
              <i className="fas fa-cloud text-2xl text-purple-600"></i>
              <span className="text-xl font-bold">CloudMesh</span>
            </div>
            <div className="hidden md:flex items-center gap-8">
              <a href="#products" className="text-gray-600 hover:text-purple-600 transition">产品</a>
              <a href="#features" className="text-gray-600 hover:text-purple-600 transition">功能</a>
              <a href="#pricing" className="text-gray-600 hover:text-purple-600 transition">定价</a>
              <a href="#contact" className="text-gray-600 hover:text-purple-600 transition">联系</a>
              <Link href="/login" className="text-gray-600 hover:text-purple-600 transition">登录</Link>
              <Link href="/deploy" className="bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition">
                免费试用
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-32 pb-20 bg-gradient-to-br from-purple-600 via-purple-700 to-indigo-800 text-white">
        <div className="container mx-auto px-4">
          <div className="text-center max-w-5xl mx-auto">
            <div className="inline-flex items-center gap-2 bg-white/10 backdrop-blur-sm px-4 py-2 rounded-full mb-6">
              <span className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></span>
              <span className="text-sm">智能 IT 管理平台</span>
            </div>
            <h1 className="text-5xl md:text-6xl font-bold mb-6 leading-tight">
              让 IT 管理更<span className="text-yellow-300">简单</span>
              <br />
              让业务运营更<span className="text-yellow-300">高效</span>
            </h1>
            <p className="text-xl md:text-2xl mb-8 opacity-90 max-w-3xl mx-auto">
              CloudMesh 提供完整的 IT 管理平台，从部署到运维，从工单到知识库，
              <br className="hidden md:block" />
              一站式解决企业 IT 管理需求
            </p>
            <div className="flex flex-col sm:flex-row justify-center gap-4">
              <Link href="/deploy" className="bg-white text-purple-600 px-8 py-4 rounded-full font-semibold text-lg hover:bg-gray-100 transition shadow-lg">
                🚀 开始免费试用
              </Link>
              <a href="#products" className="border-2 border-white text-white px-8 py-4 rounded-full font-semibold text-lg hover:bg-white hover:text-purple-600 transition">
                📖 了解产品
              </a>
            </div>
            
            {/* Stats */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6 mt-16 max-w-4xl mx-auto">
              {[
                { number: '100+', label: '服务企业' },
                { number: '100 万+', label: '处理工单' },
                { number: '99%', label: '用户满意' },
                { number: '99.9%', label: '系统可用' },
              ].map((stat, index) => (
                <div key={index} className="text-center">
                  <div className="text-3xl md:text-4xl font-bold mb-2">{stat.number}</div>
                  <div className="text-sm md:text-base opacity-80">{stat.label}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Products Section */}
      <section id="products" className="py-20 bg-gray-50">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">我们的产品</h2>
            <p className="text-xl text-gray-600">两大核心产品，覆盖 IT 管理全场景</p>
          </div>
          
          <div className="grid md:grid-cols-2 gap-8 max-w-6xl mx-auto">
            {/* OpenClaw */}
            <div className="bg-white rounded-2xl shadow-xl overflow-hidden hover:shadow-2xl transition">
              <div className="bg-gradient-to-r from-blue-500 to-cyan-500 p-8 text-white">
                <div className="text-5xl mb-4">🚀</div>
                <h3 className="text-3xl font-bold mb-2">OpenClaw</h3>
                <p className="text-blue-100 text-lg">AI 驱动的智能部署管理平台</p>
              </div>
              <div className="p-8">
                <ul className="space-y-3 mb-8">
                  {[
                    '一键部署，分钟级上线',
                    '24/7 实时监控与告警',
                    'AI 智能分析与优化建议',
                    '多环境管理与版本控制',
                    '自动化运维任务',
                  ].map((item, index) => (
                    <li key={index} className="flex items-center gap-3">
                      <i className="fas fa-check-circle text-green-500"></i>
                      <span className="text-gray-700">{item}</span>
                    </li>
                  ))}
                </ul>
                <div className="flex gap-4">
                  <a href="/deploy" className="flex-1 bg-blue-500 text-white text-center py-3 rounded-lg font-semibold hover:bg-blue-600 transition">
                    立即体验
                  </a>
                  <a href="#contact" className="flex-1 border-2 border-blue-500 text-blue-500 text-center py-3 rounded-lg font-semibold hover:bg-blue-50 transition">
                    预约演示
                  </a>
                </div>
              </div>
            </div>

            {/* ITSM */}
            <div className="bg-white rounded-2xl shadow-xl overflow-hidden hover:shadow-2xl transition">
              <div className="bg-gradient-to-r from-purple-500 to-pink-500 p-8 text-white">
                <div className="text-5xl mb-4">📋</div>
                <h3 className="text-3xl font-bold mb-2">ITSM</h3>
                <p className="text-purple-100 text-lg">企业级 IT 服务管理平台</p>
              </div>
              <div className="p-8">
                <ul className="space-y-3 mb-8">
                  {[
                    '完整的 ITIL 流程支持',
                    '智能工单管理与分配',
                    'SLA 服务级别管理',
                    'CMDB 配置管理数据库',
                    '知识库与自助服务',
                  ].map((item, index) => (
                    <li key={index} className="flex items-center gap-3">
                      <i className="fas fa-check-circle text-green-500"></i>
                      <span className="text-gray-700">{item}</span>
                    </li>
                  ))}
                </ul>
                <div className="flex gap-4">
                  <a href="/tickets" className="flex-1 bg-purple-500 text-white text-center py-3 rounded-lg font-semibold hover:bg-purple-600 transition">
                    立即体验
                  </a>
                  <a href="#contact" className="flex-1 border-2 border-purple-500 text-purple-500 text-center py-3 rounded-lg font-semibold hover:bg-purple-50 transition">
                    预约演示
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">核心优势</h2>
            <p className="text-xl text-gray-600">为什么选择 CloudMesh</p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 max-w-6xl mx-auto">
            {[
              { icon: '🤖', title: 'AI 智能赋能', desc: 'AI 驱动的自动化和智能分析，让 IT 管理更简单高效' },
              { icon: '⚡', title: '快速部署', desc: '分钟级部署上线，无需复杂配置，开箱即用' },
              { icon: '🔒', title: '安全可靠', desc: '企业级安全保护，数据加密传输，99.9% 可用性' },
              { icon: '📊', title: '数据驱动', desc: '多维度数据分析报表，助力科学决策' },
              { icon: '🔄', title: '灵活扩展', desc: '模块化设计，按需选择，随业务增长而扩展' },
              { icon: '💬', title: '专业服务', desc: '7x24 小时技术支持，专业团队全程陪伴' },
            ].map((feature, index) => (
              <div key={index} className="bg-white p-6 rounded-xl shadow-lg hover:shadow-xl transition">
                <div className="text-4xl mb-4">{feature.icon}</div>
                <h3 className="text-xl font-bold mb-3">{feature.title}</h3>
                <p className="text-gray-600">{feature.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section id="pricing" className="py-20 bg-gray-50">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">灵活定价</h2>
            <p className="text-xl text-gray-600">选择适合您的方案</p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
            {[
              {
                name: '基础版',
                price: '¥9,800',
                period: '/年',
                desc: '适合小型团队',
                features: ['最多 10 用户', '基础工单管理', '5GB 存储空间', '邮件支持'],
                cta: '免费试用',
                popular: false,
              },
              {
                name: '专业版',
                price: '¥98,000',
                period: '/年',
                desc: '适合中型企业',
                features: ['最多 100 用户', '完整 ITIL 流程', '100GB 存储空间', '优先支持 + SLA', 'API 访问'],
                cta: '免费试用',
                popular: true,
              },
              {
                name: '企业版',
                price: '面议',
                period: '',
                desc: '适合大型企业',
                features: ['无限用户', '定制化开发', '无限存储', '专属客户经理', '私有化部署', '培训服务'],
                cta: '联系我们',
                popular: false,
              },
            ].map((plan, index) => (
              <div 
                key={index} 
                className={`bg-white rounded-2xl shadow-xl overflow-hidden ${plan.popular ? 'ring-2 ring-purple-500 scale-105' : ''}`}
              >
                {plan.popular && (
                  <div className="bg-purple-500 text-white text-center py-2 text-sm font-semibold">
                    最受欢迎
                  </div>
                )}
                <div className="p-8">
                  <h3 className="text-2xl font-bold mb-2">{plan.name}</h3>
                  <p className="text-gray-600 mb-4">{plan.desc}</p>
                  <div className="mb-6">
                    <span className="text-4xl font-bold">{plan.price}</span>
                    <span className="text-gray-600">{plan.period}</span>
                  </div>
                  <ul className="space-y-3 mb-8">
                    {plan.features.map((feature, i) => (
                      <li key={i} className="flex items-center gap-3">
                        <i className="fas fa-check text-green-500"></i>
                        <span className="text-gray-700">{feature}</span>
                      </li>
                    ))}
                  </ul>
                  <a href="#contact" className={`block text-center py-3 rounded-lg font-semibold transition ${
                    plan.popular 
                      ? 'bg-purple-500 text-white hover:bg-purple-600' 
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}>
                    {plan.cta}
                  </a>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section id="contact" className="py-20 bg-gradient-to-r from-purple-600 to-indigo-600 text-white">
        <div className="container mx-auto px-4 text-center">
          <h2 className="text-4xl font-bold mb-4">准备好开始了吗？</h2>
          <p className="text-xl mb-8 opacity-90 max-w-2xl mx-auto">
            立即免费试用 30 天，无需信用卡。我们的专家团队随时为您服务。
          </p>
          <div className="flex flex-col sm:flex-row justify-center gap-4">
            <Link href="/deploy" className="bg-white text-purple-600 px-8 py-4 rounded-full font-semibold text-lg hover:bg-gray-100 transition">
              🚀 免费试用
            </Link>
            <a href="mailto:sales@cloudmesh.top" className="border-2 border-white text-white px-8 py-4 rounded-full font-semibold text-lg hover:bg-white hover:text-purple-600 transition">
              📧 联系我们
            </a>
          </div>
          <div className="mt-12 grid grid-cols-3 gap-8 max-w-2xl mx-auto">
            <div>
              <div className="text-2xl font-bold">📧</div>
              <div className="mt-2 opacity-80">sales@cloudmesh.top</div>
            </div>
            <div>
              <div className="text-2xl font-bold">💬</div>
              <div className="mt-2 opacity-80">企业微信支持</div>
            </div>
            <div>
              <div className="text-2xl font-bold">📞</div>
              <div className="mt-2 opacity-80">400-XXX-XXXX</div>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-12">
        <div className="container mx-auto px-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <i className="fas fa-cloud text-2xl text-purple-400"></i>
                <span className="text-xl font-bold">CloudMesh</span>
              </div>
              <p className="text-gray-400">智能 IT 管理平台，让 IT 管理更简单高效</p>
            </div>
            <div>
              <h4 className="font-bold mb-4">产品</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="/deploy" className="hover:text-white transition">OpenClaw</a></li>
                <li><a href="/tickets" className="hover:text-white transition">ITSM</a></li>
                <li><a href="#pricing" className="hover:text-white transition">定价</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">资源</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="/docs" className="hover:text-white transition">文档</a></li>
                <li><a href="/api" className="hover:text-white transition">API</a></li>
                <li><a href="/blog" className="hover:text-white transition">博客</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">关于</h4>
              <ul className="space-y-2 text-gray-400">
                <li><a href="/about" className="hover:text-white transition">关于我们</a></li>
                <li><a href="#contact" className="hover:text-white transition">联系</a></li>
                <li><a href="/privacy" className="hover:text-white transition">隐私政策</a></li>
              </ul>
            </div>
          </div>
          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-gray-400">
            <p>&copy; 2026 CloudMesh. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
