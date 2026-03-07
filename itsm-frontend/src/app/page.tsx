import React from 'react';
import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="min-h-screen">
      {/* Hero Section */}
      <section className="bg-gradient-to-r from-purple-600 to-purple-800 text-white py-20">
        <div className="container mx-auto px-4">
          <div className="text-center max-w-4xl mx-auto">
            <h1 className="text-5xl font-bold mb-6">
              ITSM - 企业级 IT 服务管理平台
            </h1>
            <p className="text-xl mb-8 opacity-90">
              基于 ITIL 最佳实践，AI 智能赋能，让 IT 服务更简单、更高效
            </p>
            <div className="flex justify-center gap-4">
              <Link href="/deploy" className="bg-white text-purple-600 px-8 py-3 rounded-full font-semibold hover:bg-gray-100 transition">
                🚀 免费试用
              </Link>
              <Link href="#contact" className="border-2 border-white text-white px-8 py-3 rounded-full font-semibold hover:bg-white hover:text-purple-600 transition">
                📞 预约演示
              </Link>
            </div>
          </div>
          
          {/* 数据统计 */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mt-16">
            <div className="text-center">
              <div className="text-4xl font-bold mb-2">100+</div>
              <div className="opacity-80">服务企业</div>
            </div>
            <div className="text-center">
              <div className="text-4xl font-bold mb-2">100 万+</div>
              <div className="opacity-80">处理工单</div>
            </div>
            <div className="text-center">
              <div className="text-4xl font-bold mb-2">99%+</div>
              <div className="opacity-80">用户满意</div>
            </div>
            <div className="text-center">
              <div className="text-4xl font-bold mb-2">99.9%</div>
              <div className="opacity-80">系统可用</div>
            </div>
          </div>
        </div>
      </section>

      {/* 核心功能 */}
      <section className="py-20 bg-gray-50">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">核心功能</h2>
            <p className="text-xl text-gray-600">完整的 ITIL 流程支持，满足企业 IT 服务管理需求</p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
            {[
              { icon: '📋', title: '工单管理', desc: '完整的工单生命周期管理' },
              { icon: '🔥', title: '事件管理', desc: '快速响应和处理 IT 事件' },
              { icon: '🐛', title: '问题管理', desc: '根因分析，预防问题复发' },
              { icon: '🔄', title: '变更管理', desc: '规范的变更流程管理' },
              { icon: '⏰', title: 'SLA 管理', desc: '服务级别协议管理' },
              { icon: '💾', title: 'CMDB', desc: '配置管理数据库' },
              { icon: '📚', title: '知识库', desc: '知识积累和共享' },
              { icon: '📊', title: '报表分析', desc: '多维度数据分析' },
            ].map((feature, index) => (
              <div key={index} className="bg-white p-6 rounded-lg shadow-lg hover:shadow-xl transition">
                <div className="text-4xl mb-4">{feature.icon}</div>
                <h3 className="text-xl font-bold mb-2">{feature.title}</h3>
                <p className="text-gray-600">{feature.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 客户案例 */}
      <section className="py-20">
        <div className="container mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">客户案例</h2>
            <p className="text-xl text-gray-600">深受企业信赖的 ITSM 解决方案</p>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {[
              { name: '某大型企业', industry: '制造业', result: '效率提升 50%' },
              { name: '某政府部门', industry: '政府', result: '满意度 99%' },
              { name: '某教育机构', industry: '教育', result: '成本降低 30%' },
            ].map((case_study, index) => (
              <div key={index} className="bg-gradient-to-br from-purple-50 to-purple-100 p-6 rounded-lg">
                <h3 className="text-xl font-bold mb-2">{case_study.name}</h3>
                <p className="text-gray-600 mb-4">{case_study.industry}</p>
                <div className="text-2xl font-bold text-purple-600">{case_study.result}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section id="contact" className="py-20 bg-gradient-to-r from-purple-600 to-purple-800 text-white">
        <div className="container mx-auto px-4 text-center">
          <h2 className="text-4xl font-bold mb-4">立即开始使用</h2>
          <p className="text-xl mb-8 opacity-90">免费试用 30 天，无需信用卡</p>
          <div className="flex justify-center gap-4">
            <Link href="/deploy" className="bg-white text-purple-600 px-8 py-3 rounded-full font-semibold hover:bg-gray-100 transition">
              🚀 免费试用
            </Link>
            <Link href="#contact" className="border-2 border-white text-white px-8 py-3 rounded-full font-semibold hover:bg-white hover:text-purple-600 transition">
              📞 联系我们
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-12">
        <div className="container mx-auto px-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            <div>
              <h3 className="text-xl font-bold mb-4">ITSM</h3>
              <p className="text-gray-400">企业级 IT 服务管理平台</p>
            </div>
            <div>
              <h4 className="font-bold mb-4">产品</h4>
              <ul className="space-y-2 text-gray-400">
                <li><Link href="/product" className="hover:text-white">功能</Link></li>
                <li><Link href="/pricing" className="hover:text-white">定价</Link></li>
                <li><Link href="/deploy" className="hover:text-white">部署</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">资源</h4>
              <ul className="space-y-2 text-gray-400">
                <li><Link href="/docs" className="hover:text-white">文档</Link></li>
                <li><Link href="/cases" className="hover:text-white">案例</Link></li>
                <li><Link href="/blog" className="hover:text-white">博客</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">关于</h4>
              <ul className="space-y-2 text-gray-400">
                <li><Link href="/about" className="hover:text-white">公司</Link></li>
                <li><Link href="/contact" className="hover:text-white">联系</Link></li>
                <li><Link href="/privacy" className="hover:text-white">隐私</Link></li>
              </ul>
            </div>
          </div>
          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-gray-400">
            <p>&copy; 2026 ITSM. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
