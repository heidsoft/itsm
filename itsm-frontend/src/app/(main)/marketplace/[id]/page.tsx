'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { 
  Card, 
  CardContent, 
  CardDescription, 
  CardFooter, 
  CardHeader, 
  CardTitle 
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { 
  ArrowLeft, 
  Star, 
  Download, 
  CheckCircle2, 
  AlertCircle,
  User, 
  Calendar, 
  Code, 
  Shield,
  Terminal,
  Tag,
  Globe
} from 'lucide-react';
import Link from 'next/link';

// 应用详情类型
interface MarketplaceItemDetail {
  id: number;
  name: string;
  title: string;
  type: 'connector' | 'skill' | 'plugin';
  provider: string;
  description: string;
  long_description: string;
  icon_url: string;
  screenshots: string[];
  rating: number;
  install_count: number;
  category: string;
  tags: string[];
  is_official: boolean;
  is_free: boolean;
  price: number;
  latest_version: string;
  min_system_version: string;
  author: string;
  homepage: string;
  repository: string;
  license: string;
  capabilities: string[];
  required_permissions: string[];
  config_schema: any;
  created_at: string;
  updated_at: string;
  versions: {
    version: string;
    changelog: string;
    released_at: string;
  }[];
}

const MarketplaceDetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const itemId = params.id as string;
  
  const [item, setItem] = useState<MarketplaceItemDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState(false);
  const [isInstalled, setIsInstalled] = useState(false);

  // 模拟获取详情
  useEffect(() => {
    const fetchItem = async () => {
      try {
        // 替换为真实API调用
        // const res = await fetch(`/api/v1/marketplace/items/${itemId}`);
        // const data = await res.json();
        // setItem(data.data);

        // 模拟数据
        const mockItem: MarketplaceItemDetail = {
          id: parseInt(itemId),
          name: 'feishu-connector',
          title: '飞书连接器',
          type: 'connector',
          provider: '官方',
          description: '集成飞书消息、审批、组织架构同步',
          long_description: `
## 功能特性
- ✅ **消息通知**：支持工单通知、审批通知、SLA告警等多种消息类型推送到飞书
- ✅ **消息卡片**：丰富的卡片模板，支持操作按钮，用户可以直接在飞书中处理工单
- ✅ **组织架构同步**：自动同步飞书组织架构、用户信息，无需手动维护
- ✅ **审批集成**：飞书审批通过后自动更新ITSM中的工单状态
- ✅ **事件回调**：支持飞书告警事件自动创建ITSM事件工单

## 使用场景
1. 工单状态变更时自动通知相关处理人
2. 审批流程通过飞书进行移动端审批
3. 自动同步用户信息，减少管理员维护工作量
4. 监控告警事件自动生成工单并通知相关人员
          `,
          icon_url: 'https://cdn-icons-png.flaticon.com/512/5968/5968757.png',
          screenshots: [
            'https://picsum.photos/800/450?random=1',
            'https://picsum.photos/800/450?random=2',
            'https://picsum.photos/800/450?random=3'
          ],
          rating: 4.8,
          install_count: 1245,
          category: '办公协同',
          tags: ['飞书', '消息通知', '组织架构', '审批'],
          is_official: true,
          is_free: true,
          price: 0,
          latest_version: '1.2.0',
          min_system_version: '1.5.0',
          author: 'ITSM团队',
          homepage: 'https://github.com/heidsoft/itsm',
          repository: 'https://github.com/heidsoft/itsm/tree/main/itsm-backend/connector/builtin/feishu',
          license: 'MIT',
          capabilities: [
            '发送消息通知',
            '接收消息回调',
            '发送卡片消息',
            '同步组织架构',
            '同步用户信息',
            '审批流程集成'
          ],
          required_permissions: [
            '读取用户基本信息',
            '发送消息',
            '读取组织架构',
            '接收事件回调'
          ],
          config_schema: {
            type: 'object',
            properties: {
              app_id: {
                type: 'string',
                title: '应用ID',
                description: '飞书开放平台应用的App ID'
              },
              app_secret: {
                type: 'string',
                title: '应用密钥',
                description: '飞书开放平台应用的App Secret'
              },
              verification_token: {
                type: 'string',
                title: '验证令牌',
                description: '事件回调的验证令牌'
              },
              encrypt_key: {
                type: 'string',
                title: '加密密钥',
                description: '事件回调的加密密钥',
                optional: true
              }
            },
            required: ['app_id', 'app_secret', 'verification_token']
          },
          created_at: '2024-01-15T00:00:00Z',
          updated_at: '2024-05-20T00:00:00Z',
          versions: [
            {
              version: '1.2.0',
              changelog: '支持审批回调功能，优化同步性能',
              released_at: '2024-05-20T00:00:00Z'
            },
            {
              version: '1.1.0',
              changelog: '新增组织架构同步功能，修复已知问题',
              released_at: '2024-03-15T00:00:00Z'
            },
            {
              version: '1.0.0',
              changelog: '初始版本，支持消息通知功能',
              released_at: '2024-01-15T00:00:00Z'
            }
          ]
        };

        setItem(mockItem);

        // 模拟检查是否已安装
        // const installRes = await fetch(`/api/v1/marketplace/installations/${itemId}`);
        // setIsInstalled(installRes.ok);
        setIsInstalled(false);
      } catch (error) {
        console.error('Failed to fetch item detail:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchItem();
  }, [itemId]);

  // 安装应用
  const handleInstall = async () => {
    setInstalling(true);
    try {
      // 替换为真实的安装API调用
      // await fetch(`/api/v1/marketplace/items/${itemId}/install`, {
      //   method: 'POST'
      // });
      
      // 模拟安装延迟
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      setIsInstalled(true);
      // 跳转到已安装页面或者配置页面
      // router.push('/installations');
    } catch (error) {
      console.error('Failed to install item:', error);
    } finally {
      setInstalling(false);
    }
  };

  // 卸载应用
  const handleUninstall = async () => {
    if (confirm('确定要卸载该应用吗？卸载后相关功能将无法使用。')) {
      try {
        // 替换为真实的卸载API调用
        // await fetch(`/api/v1/marketplace/items/${itemId}/uninstall`, {
        //   method: 'POST'
        // });
        
        setIsInstalled(false);
      } catch (error) {
        console.error('Failed to uninstall item:', error);
      }
    }
  };

  // 类型名称映射
  const typeNames = {
    connector: '连接器',
    skill: '技能',
    plugin: '插件'
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!item) {
    return (
      <div className="container mx-auto px-4 py-6">
        <div className="bg-white rounded-lg shadow-sm p-10 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4 mx-auto" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">应用不存在</h3>
          <p className="text-gray-500 mb-4">您访问的应用可能已被下架或删除</p>
          <Link href="/marketplace">
            <Button>返回应用市场</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      {/* 面包屑导航 */}
      <div className="flex items-center gap-2 mb-6 text-sm">
        <Link href="/marketplace" className="text-gray-500 hover:text-gray-700 flex items-center gap-1">
          <ArrowLeft className="h-4 w-4" />
          返回市场
        </Link>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 左侧主内容区 */}
        <div className="lg:col-span-2 space-y-6">
          {/* 应用基本信息卡片 */}
          <Card>
            <CardHeader>
              <div className="flex items-start gap-4">
                <div className="w-16 h-16 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                  {item.icon_url ? (
                    <img src={item.icon_url} alt={item.title} className="w-full h-full object-cover" />
                  ) : (
                    <div className="text-2xl">{typeNames[item.type]}</div>
                  )}
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-1">
                    <h1 className="text-2xl font-bold">{item.title}</h1>
                    <Badge variant="secondary">{typeNames[item.type]}</Badge>
                    {item.is_official && (
                      <Badge className="bg-blue-100 text-blue-800 hover:bg-blue-200">官方</Badge>
                    )}
                    {item.is_free ? (
                      <Badge className="bg-green-100 text-green-800 hover:bg-green-200">免费</Badge>
                    ) : (
                      <Badge className="bg-orange-100 text-orange-800 hover:bg-orange-200">¥{item.price}</Badge>
                    )}
                  </div>
                  <p className="text-gray-600">{item.description}</p>
                  <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                    <div className="flex items-center gap-1">
                      <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                      <span>{item.rating.toFixed(1)}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <Download className="h-4 w-4" />
                      <span>{item.install_count} 次安装</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <User className="h-4 w-4" />
                      <span>作者：{item.author}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" />
                      <span>更新于 {new Date(item.updated_at).toLocaleDateString()}</span>
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  {isInstalled ? (
                    <div className="space-y-2">
                      <div className="flex items-center justify-end gap-1 text-green-600 mb-2">
                        <CheckCircle2 className="h-5 w-5" />
                        <span>已安装</span>
                      </div>
                      <Button variant="default" onClick={() => router.push('/installations')}>
                        管理
                      </Button>
                      <Button variant="destructive" className="ml-2" onClick={handleUninstall}>
                        卸载
                      </Button>
                    </div>
                  ) : (
                    <Button 
                      size="lg" 
                      onClick={handleInstall}
                      disabled={installing}
                    >
                      {installing ? '安装中...' : '立即安装'}
                    </Button>
                  )}
                  <p className="text-xs text-gray-500 mt-2">
                    版本 {item.latest_version}
                  </p>
                </div>
              </div>
            </CardHeader>
          </Card>

          {/* 详情标签页 */}
          <Card>
            <Tabs defaultValue="description">
              <CardHeader>
                <TabsList>
                  <TabsTrigger value="description">详情介绍</TabsTrigger>
                  <TabsTrigger value="capabilities">功能特性</TabsTrigger>
                  <TabsTrigger value="permissions">权限说明</TabsTrigger>
                  <TabsTrigger value="versions">版本历史</TabsTrigger>
                </TabsList>
              </CardHeader>
              <CardContent>
                <TabsContent value="description" className="mt-0">
                  <div className="prose prose-sm max-w-none">
                    {item.long_description.split('\n').map((line, index) => (
                      <p key={index}>{line}</p>
                    ))}
                  </div>
                  
                  {item.screenshots.length > 0 && (
                    <div className="mt-6">
                      <h3 className="text-lg font-medium mb-3">截图预览</h3>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        {item.screenshots.map((screenshot, index) => (
                          <img 
                            key={index} 
                            src={screenshot} 
                            alt={`截图 ${index + 1}`} 
                            className="rounded-md border shadow-sm w-full h-auto"
                          />
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="mt-6">
                    <h3 className="text-lg font-medium mb-3">相关标签</h3>
                    <div className="flex flex-wrap gap-2">
                      {item.tags.map(tag => (
                        <Badge key={tag} variant="outline">
                          <Tag className="h-3 w-3 mr-1" />
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </div>

                  <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="flex items-start gap-2">
                      <Globe className="h-5 w-5 text-gray-400 mt-0.5" />
                      <div>
                        <div className="text-sm font-medium">官方网站</div>
                        <a 
                          href={item.homepage} 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-sm text-blue-600 hover:underline"
                        >
                          {item.homepage}
                        </a>
                      </div>
                    </div>
                    <div className="flex items-start gap-2">
                      <Code className="h-5 w-5 text-gray-400 mt-0.5" />
                      <div>
                        <div className="text-sm font-medium">代码仓库</div>
                        <a 
                          href={item.repository} 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-sm text-blue-600 hover:underline"
                        >
                          {item.repository}
                        </a>
                      </div>
                    </div>
                    <div className="flex items-start gap-2">
                      <Shield className="h-5 w-5 text-gray-400 mt-0.5" />
                      <div>
                        <div className="text-sm font-medium">开源协议</div>
                        <div className="text-sm text-gray-600">{item.license}</div>
                      </div>
                    </div>
                    <div className="flex items-start gap-2">
                      <Terminal className="h-5 w-5 text-gray-400 mt-0.5" />
                      <div>
                        <div className="text-sm font-medium">最低系统版本</div>
                        <div className="text-sm text-gray-600">{item.min_system_version}</div>
                      </div>
                    </div>
                  </div>
                </TabsContent>

                <TabsContent value="capabilities" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">支持的功能特性</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {item.capabilities.map((capability, index) => (
                      <div key={index} className="flex items-center gap-2 p-2 bg-gray-50 rounded-md">
                        <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
                        <span>{capability}</span>
                      </div>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="permissions" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">需要的系统权限</h3>
                  <div className="space-y-3">
                    {item.required_permissions.map((permission, index) => (
                      <div key={index} className="flex items-start gap-2 p-2 bg-gray-50 rounded-md">
                        <Shield className="h-5 w-5 text-blue-500 flex-shrink-0 mt-0.5" />
                        <span>{permission}</span>
                      </div>
                    ))}
                  </div>
                  <div className="mt-4 p-3 bg-blue-50 rounded-md text-sm text-blue-700">
                    <AlertCircle className="h-4 w-4 inline mr-1" />
                    我们会严格保护您的数据安全，所有权限仅用于应用正常功能，不会用于其他用途。
                  </div>
                </TabsContent>

                <TabsContent value="versions" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">版本历史</h3>
                  <div className="space-y-4">
                    {item.versions.map((version, index) => (
                      <div key={index} className="border-l-2 border-gray-200 pl-4 pb-4">
                        <div className="flex items-center justify-between mb-1">
                          <div className="font-medium">v{version.version}</div>
                          <div className="text-sm text-gray-500">
                            {new Date(version.released_at).toLocaleDateString()}
                          </div>
                        </div>
                        <p className="text-sm text-gray-600 whitespace-pre-line">{version.changelog}</p>
                      </div>
                    ))}
                  </div>
                </TabsContent>
              </CardContent>
            </Tabs>
          </Card>
        </div>

        {/* 右侧边栏 */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>安装信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <div className="text-sm font-medium mb-1">最新版本</div>
                <div className="text-gray-600">{item.latest_version}</div>
              </div>
              <div>
                <div className="text-sm font-medium mb-1">发布日期</div>
                <div className="text-gray-600">
                  {new Date(item.versions[0].released_at).toLocaleDateString()}
                </div>
              </div>
              <div>
                <div className="text-sm font-medium mb-1">最低系统要求</div>
                <div className="text-gray-600">{item.min_system_version}</div>
              </div>
              <div>
                <div className="text-sm font-medium mb-1">安装量</div>
                <div className="text-gray-600">{item.install_count}</div>
              </div>
              <div>
                <div className="text-sm font-medium mb-1">分类</div>
                <Badge variant="secondary">{item.category}</Badge>
              </div>
            </CardContent>
            <CardFooter className="border-t pt-4">
              {isInstalled ? (
                <Button 
                  className="w-full" 
                  variant="default"
                  onClick={() => router.push('/installations')}
                >
                  管理已安装应用
                </Button>
              ) : (
                <Button 
                  className="w-full" 
                  size="lg"
                  onClick={handleInstall}
                  disabled={installing}
                >
                  {installing ? '安装中...' : '立即安装'}
                </Button>
              )}
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>提供商信息</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-blue-600 font-medium">
                  {item.provider.charAt(0)}
                </div>
                <div>
                  <div className="font-medium">{item.provider}</div>
                  {item.is_official && (
                    <Badge className="mt-1 bg-blue-100 text-blue-800 hover:bg-blue-200">
                      官方认证
                    </Badge>
                  )}
                </div>
              </div>
              <p className="text-sm text-gray-600">
                {item.is_official 
                  ? '由ITSM官方团队开发和维护，确保兼容性和安全性。' 
                  : '由社区开发者贡献，经过官方审核验证。'
                }
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default MarketplaceDetailPage;
