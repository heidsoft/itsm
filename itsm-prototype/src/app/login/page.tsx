'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import {Off, ArrowRight, Zap, BarChart3, Globe, Server, Sparkles, AlertCircle } from 'lucide-react';
import { Icon, commonIcons } from '@/components/ui';

/**
 * 登录页面组件
 * 提供用户身份验证功能，包含现代化的UI设计和用户体验优化
 */
export default function LoginPage() {
  const router = useRouter();
  
  // 表单状态管理
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [isFormValid, setIsFormValid] = useState(false);

  // 实时表单验证
  useEffect(() => {
    setIsFormValid(username.trim().length > 0 && password.length >= 6);
  }, [username, password]);

  // 密码强度计算
  const getPasswordStrength = (pwd: string) => {
    if (pwd.length === 0) return { level: 0, text: '', color: '' };
    if (pwd.length < 6) return { level: 1, text: '弱', color: 'bg-red-500' };
    if (pwd.length < 10) return { level: 2, text: '中', color: 'bg-yellow-500' };
    return { level: 3, text: '强', color: 'bg-green-500' };
  };

  const passwordStrength = getPasswordStrength(password);

  // 模拟认证服务
  const mockAuthService = {
    async login(username: string, password: string) {
      // 模拟网络延迟
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // 模拟认证逻辑
      if (username === 'admin' && password === 'admin123') {
        return {
          success: true,
          user: { id: 1, username: 'admin', name: '系统管理员' },
          token: 'mock-jwt-token'
        };
      }
      
      throw new Error('用户名或密码错误');
    }
  };

  // 处理登录提交
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!username.trim() || !password) {
      setError('请输入用户名和密码');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const result = await mockAuthService.login(username, password);
      
      if (result.success) {
        // 存储认证信息
        localStorage.setItem('auth_token', result.token);
        localStorage.setItem('user_info', JSON.stringify(result.user));
        
        // 跳转到仪表板
        router.push('/dashboard');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  // 密码可见性通过按钮内联处理，无需单独函数

  return (
    <div className="min-h-screen bg-gradient-to-br from-indigo-50 via-sky-50 to-slate-100 flex">
      {/* 左侧品牌区域 */}
      <div className="hidden lg:flex lg:w-1/2 relative overflow-hidden">
        {/* 背景装饰 */}
        <div className="absolute inset-0 bg-gradient-to-br from-blue-600 via-indigo-700 to-purple-800"></div>
        <div className="absolute inset-0 bg-black/20"></div>
        
        {/* 装饰性几何图形 */}
        <div className="absolute top-20 left-20 w-32 h-32 bg-white/10 rounded-full blur-xl"></div>
        <div className="absolute bottom-40 right-20 w-48 h-48 bg-white/5 rounded-full blur-2xl"></div>
        <div className="absolute top-1/2 left-1/3 w-24 h-24 bg-yellow-400/20 rounded-full blur-lg"></div>
        
        {/* 主要内容 */}
        <div className="relative z-10 flex flex-col justify-center px-16 py-12 text-white">
          {/* Logo和标题 */}
          <div className="mb-12">
            <div className="flex items-center mb-6">
              <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center backdrop-blur-sm">
                <Server className="w-7 h-7 text-white" />
              </div>
              <div className="ml-4">
                <h1 className="heading-3 text-white">ITSM Pro</h1>
                  <p className="text-caption text-blue-100">智能IT服务管理平台</p>
              </div>
            </div>
            
            <h2 className="heading-2 text-white mb-4">
              现代化的
              <br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-yellow-400 to-orange-400">
                IT服务管理
              </span>
            </h2>
            
            <p className="body-large text-blue-100 readable-narrow">
              集成事件管理、问题管理、变更管理和资产管理于一体，
              <br />
              为您的企业提供全方位的IT服务支持解决方案。
            </p>
          </div>

          {/* 特性标签 */}
          <div className="flex flex-wrap gap-3 mb-12">
            {[
              { icon: Zap, text: '快速响应', desc: '秒级事件处理' },
              { icon: Shield, text: '安全可靠', desc: '企业级安全' },
              { icon: BarChart3, text: '数据洞察', desc: '智能分析报告' },
              { icon: Globe, text: '全球部署', desc: '多地域支持' }
            ].map((feature, index) => (
              <div 
                key={index}
                className="group bg-white/10 backdrop-blur-sm rounded-lg px-4 py-3 hover:bg-white/20 transition-all duration-300 cursor-pointer"
              >
                <div className="flex items-center space-x-3">
                  <feature.icon className="w-5 h-5 text-yellow-400 group-hover:scale-110 transition-transform" />
                  <div>
                    <span className="font-medium text-sm">{feature.text}</span>
                    <p className="text-xs text-blue-100 opacity-80">{feature.desc}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* 统计数据 */}
          <div className="grid grid-cols-3 gap-8 mb-12">
            <div className="text-center">
              <div className="text-3xl font-bold mb-1">99.9%</div>
              <div className="text-sm text-blue-200">系统可用性</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold mb-1">10k+</div>
              <div className="text-sm text-blue-200">企业用户</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold mb-1">24/7</div>
              <div className="text-sm text-blue-200">技术支持</div>
            </div>
          </div>

          {/* 底部信息 */}
          <div className="flex items-center justify-between text-sm text-blue-200">
            <div className="flex items-center space-x-4">
              <span>© 2024 ITSM Pro</span>
              <span>•</span>
              <span>企业级服务</span>
            </div>
            <div className="flex items-center space-x-2">
              <Sparkles className="w-4 h-4" />
              <span>持续创新</span>
            </div>
          </div>
        </div>
      </div>

      {/* 右侧登录表单区域 */}
      <div className="flex-1 flex items-center justify-center px-6 py-12 lg:px-8">
        <div className="w-full max-w-md bg-white/80 backdrop-blur-sm border border-gray-200 shadow-xl rounded-2xl p-8">
          {/* 移动端Logo */}
          <div className="lg:hidden text-center mb-8">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-2xl mb-4">
              <Server className="w-8 h-8 text-white" />
            </div>
            <h1 className="heading-4 text-gray-900">ITSM Pro</h1>
            <p className="text-subtitle text-gray-600">智能IT服务管理平台</p>
          </div>

          {/* 表单标题 */}
          <div className="text-center mb-8">
            <h2 className="heading-3 text-gray-900 mb-2">欢迎回来</h2>
            <p className="body-medium text-gray-600">请登录您的账户以继续使用服务</p>
          </div>

          {/* 错误提示 */}
          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-center space-x-3">
              <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
              <span className="text-red-700 text-sm">{error}</span>
            </div>
          )}

          {/* 登录表单 */}
          <form onSubmit={handleLogin} className="space-y-6">
            {/* 用户名输入 */}
            <div className="fade-in-up delay-100">
              <label htmlFor="username" className="block body-small font-medium text-gray-700 mb-2">
                用户名
              </label>
              <div className="relative">
                    <Icon 
                      icon={User} 
                      size="sm" 
                      className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" 
                    />
                    <input
                      type="text"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200 animate-fade-in"
                      placeholder="请输入用户名"
                      required
                    />
                  </div>
            </div>

            {/* 密码输入 */}
            <div className="fade-in-up delay-200">
              <label htmlFor="password" className="block body-small font-medium text-gray-700 mb-2">
                密码
              </label>
              <div className="relative">
                    <Icon 
                      icon={Lock} 
                      size="sm" 
                      className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" 
                    />
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full pl-10 pr-12 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200 animate-fade-in"
                      placeholder="请输入密码"
                      required
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors duration-200 animate-icon-hover"
                    >
                      <Icon 
                        icon={showPassword ? EyeOff : Eye} 
                        size="sm" 
                      />
                    </button>
                  </div>
              
              {/* 密码强度指示器 */}
              {password && (
                <div className="mt-2 fade-in-up">
                  <div className="flex items-center space-x-2">
                    <div className="flex-1 bg-gray-200 rounded-full h-1 progress-animated">
                      <div 
                        className={`h-1 rounded-full transition-all duration-500 ${passwordStrength.color}`}
                        style={{ width: `${(passwordStrength.level / 3) * 100}%` }}
                      ></div>
                    </div>
                    <span className="text-xs text-muted">
                      密码强度: {passwordStrength.text}
                    </span>
                  </div>
                </div>
              )}
            </div>

            {/* 登录按钮 */}
            <div className="fade-in-up delay-300">
              <button
                type="submit"
                disabled={loading || !isFormValid}
                className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 animate-button-hover animate-fade-in flex items-center justify-center gap-2"
                >
                  {loading ? (
                    <>
                      <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
                      登录中...
                    </>
                  ) : (
                    <>
                      登录
                      <Icon 
                        icon={ArrowRight} 
                        size="sm" 
                        className="animate-icon-hover" 
                      />
                    </>
                  )}
                </button>
            </div>
          </form>

          {/* 安全提示 */}
          <div className="mt-6">
            <div className="flex items-center gap-3 p-4 bg-blue-50 rounded-lg animate-fade-in">
              <Icon 
                icon={commonIcons.security} 
                size="sm" 
                color="primary" 
              />
              <div>
                <p className="body-small font-medium text-blue-900">安全提示</p>
                <p className="body-xs text-blue-700">请确保在安全的网络环境下登录，保护您的账户安全。</p>
              </div>
            </div>
          </div>

          {/* 底部链接 */}
          <div className="mt-8 text-center">
            <div className="text-sm text-gray-500">
              遇到问题？
              <a href="#" className="font-medium text-blue-600 hover:text-blue-500 ml-1">
                联系技术支持
              </a>
            </div>
            <div className="mt-4 text-xs text-gray-400">
              © 2024 ITSM Pro. 保留所有权利。
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
