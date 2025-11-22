'use client';

import { ArrowLeft, CalendarDays, User, Eye } from 'lucide-react';

import React from 'react';
import { useParams, useRouter } from 'next/navigation';
// 模拟知识文章详情数据
const mockArticleDetail = {
  'KB-001': {
    title: '如何重置您的域账户密码',
    category: '账号管理',
    author: 'IT服务台',
    publishedDate: '2024-01-15',
    views: 1250,
    content: `
        <p>如果您忘记了域账户密码，可以按照以下步骤进行自助重置：</p>
        <ol class="list-decimal list-inside space-y-2">
            <li>访问公司密码重置门户：<a href="#" class="text-blue-600 hover:underline">https://password.yourcompany.com</a></li>
            <li>输入您的域账户名和验证码。</li>
            <li>根据提示完成身份验证（例如：手机短信验证码、邮箱验证码）。</li>
            <li>设置新密码，并确保新密码符合公司密码策略要求（至少8位，包含大小写字母、数字和特殊字符）。</li>
            <li>点击“提交”完成密码重置。</li>
        </ol>
        <p class="mt-4">如果自助重置失败，请联系IT服务台寻求帮助。</p>
        `,
  },
  'KB-002': {
    title: 'Web服务器CPU高占用率排查指南',
    category: '故障排除',
    author: '运维团队',
    publishedDate: '2024-03-20',
    views: 890,
    content: `
        <p>当Web服务器出现CPU高占用率时，可以按照以下步骤进行排查：</p>
        <ol class="list-decimal list-inside space-y-2">
            <li><strong>登录服务器：</strong> 使用SSH工具登录到目标Web服务器。</li>
            <li><strong>检查进程：</strong> 运行 <code>top</code> 或 <code>htop</code> 命令，查看哪些进程占用了大量CPU资源。</li>
            <li><strong>分析日志：</strong> 检查Web服务器（如Apache, Nginx）的访问日志和错误日志，查找异常请求或错误信息。</li>
            <li><strong>应用层面排查：</strong> 如果是特定应用进程导致，检查应用自身的日志，分析代码逻辑或数据库查询。</li>
            <li><strong>系统资源检查：</strong> 检查内存、磁盘I/O等其他系统资源是否也存在瓶颈。</li>
            <li><strong>临时措施：</strong> 针对性地重启服务、优化配置或隔离问题应用。</li>
            <li><strong>记录与报告：</strong> 记录排查过程、发现的问题和解决方案，并更新事件或问题工单。</li>
        </ol>
        <p class="mt-4"><strong>常见原因：</strong> 恶意攻击（DDoS）、代码死循环、数据库慢查询、资源泄漏、配置错误等。</p>
        `,
  },
};

const KnowledgeArticlePage = () => {
  const params = useParams();
  const router = useRouter();
  const articleId = params.articleId as string;
  const article = mockArticleDetail[articleId];

  if (!article) {
    return <div className='p-10'>知识文章不存在或加载失败。</div>;
  }

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回知识库
        </button>
        <h2 className='text-4xl font-bold text-gray-800'>{article.title}</h2>
        <div className='flex items-center text-gray-500 text-sm mt-2 space-x-4'>
          <div className='flex items-center'>
            <User className='w-4 h-4 mr-1' /> {article.author}
          </div>
          <div className='flex items-center'>
            <CalendarDays className='w-4 h-4 mr-1' /> {article.publishedDate}
          </div>
          <div className='flex items-center'>
            <Eye className='w-4 h-4 mr-1' /> {article.views} 浏览
          </div>
        </div>
      </header>

      <div className='bg-white p-8 rounded-lg shadow-md'>
        <div
          className='prose max-w-none text-gray-700 leading-relaxed'
          dangerouslySetInnerHTML={{ __html: article.content }}
        />
        {/* 评论和评分区域 (模拟) */}
        <div className='mt-8 pt-8 border-t border-gray-200'>
          <h3 className='text-xl font-semibold text-gray-700 mb-4'>评论与反馈</h3>
          <p className='text-gray-500'>这里将是用户评论和文章评分区域。</p>
        </div>
      </div>
    </div>
  );
};

export default KnowledgeArticlePage;
