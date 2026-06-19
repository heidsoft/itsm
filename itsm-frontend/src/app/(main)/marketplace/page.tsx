'use client';

import React, { useState, useEffect } from 'react';
import { 
  Card, 
  CardContent, 
  CardDescription, 
  CardFooter, 
  CardHeader, 
  CardTitle 
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { 
  Search, 
  Plug, 
  Brain, 
  Layers, 
  Star, 
  Download, 
  User, 
  CheckCircle2,
  PlusCircle
} from 'lucide-react';
import Link from 'next/link';

// 组件类型定义
interface MarketplaceItem {
  id: number;
  name: string;
  title: string;
  type: 'connector' | 'skill' | 'plugin';
  provider: string;
  description: string;
  icon_url: string;
  rating: number;
  install_count: number;
  category: string;
  tags: string[];
  is_official: boolean;
  latest_version: string;
}

const MarketplacePage = () => {
  const [items, setItems] = useState<MarketplaceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState('all');
  const [category, setCategory] = useState('all');
  const [isOfficial, setIsOfficial] = useState<string>('all');

  // 模拟获取数据
  useEffect(() => {
    const fetchItems = async () => {
      try {
        // 这里替换为真实的API调用
        // const res = await fetch('/api/v1/marketplace/items');
        // const data = await res.json();
        // setItems(data.data.items);
        
        // 模拟数据
        const mockItems: MarketplaceItem[] = [
          {
            id: 1,
            name: 'feishu-connector',
            title: '飞书连接器',
            type: 'connector',
            provider: '官方',
            description: '集成飞书消息、审批、组织架构同步',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/5968/5968757.png',
            rating: 4.8,
            install_count: 1245,
            category: '办公协同',
            tags: ['飞书', '消息通知', '组织架构'],
            is_official: true,
            latest_version: '1.2.0'
          },
          {
            id: 2,
            name: 'dingtalk-connector',
            title: '钉钉连接器',
            type: 'connector',
            provider: '官方',
            description: '集成钉钉消息、审批、组织架构同步',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/5968/5968852.png',
            rating: 4.7,
            install_count: 987,
            category: '办公协同',
            tags: ['钉钉', '消息通知', '组织架构'],
            is_official: true,
            latest_version: '1.1.0'
          },
          {
            id: 3,
            name: 'wecom-connector',
            title: '企业微信连接器',
            type: 'connector',
            provider: '官方',
            description: '集成企业微信消息、审批、组织架构同步',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/5968/5968830.png',
            rating: 4.6,
            install_count: 856,
            category: '办公协同',
            tags: ['企业微信', '消息通知', '组织架构'],
            is_official: true,
            latest_version: '1.0.0'
          },
          {
            id: 4,
            name: 'ticket-classification-skill',
            title: '工单智能分类',
            type: 'skill',
            provider: '官方',
            description: '自动识别工单类型、优先级，智能分派到对应团队',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/4712/4712109.png',
            rating: 4.9,
            install_count: 1568,
            category: 'AI能力',
            tags: ['AI', '工单', '分类'],
            is_official: true,
            latest_version: '2.0.0'
          },
          {
            id: 5,
            name: 'ticket-summary-skill',
            title: '工单摘要生成',
            type: 'skill',
            provider: '官方',
            description: '自动生成工单摘要，提取关键信息，提高处理效率',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/1005/1005141.png',
            rating: 4.7,
            install_count: 1123,
            category: 'AI能力',
            tags: ['AI', '工单', '摘要'],
            is_official: true,
            latest_version: '1.1.0'
          },
          {
            id: 6,
            name: 'slack-connector',
            title: 'Slack连接器',
            type: 'connector',
            provider: '社区',
            description: '集成Slack消息通知',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/2111/2111615.png',
            rating: 4.5,
            install_count: 542,
            category: '办公协同',
            tags: ['Slack', '消息通知'],
            is_official: false,
            latest_version: '0.9.0'
          },
          {
            id: 7,
            name: 'prometheus-connector',
            title: 'Prometheus连接器',
            type: 'connector',
            provider: '官方',
            description: '集成Prometheus告警，自动生成事件工单',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/9336/9336119.png',
            rating: 4.8,
            install_count: 1325,
            category: '监控告警',
            tags: ['Prometheus', '监控', '告警'],
            is_official: true,
            latest_version: '1.0.0'
          },
          {
            id: 8,
            name: 'bulk-operation-plugin',
            title: '批量操作插件',
            type: 'plugin',
            provider: '官方',
            description: '支持工单、CMDB等批量操作，提高管理效率',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/3209/3209675.png',
            rating: 4.6,
            install_count: 987,
            category: '效率工具',
            tags: ['批量操作', '工单', 'CMDB'],
            is_official: true,
            latest_version: '1.0.0'
          }
        ];
        
        setItems(mockItems);
      } catch (error) {
        console.error('Failed to fetch marketplace items:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchItems();
  }, []);

  // 过滤数据
  const filteredItems = items.filter(item => {
    // 搜索过滤
    const matchesSearch = !search || 
      item.title.toLowerCase().includes(search.toLowerCase()) ||
      item.description.toLowerCase().includes(search.toLowerCase()) ||
      item.tags.some(tag => tag.toLowerCase().includes(search.toLowerCase()));

    // 类型过滤
    const matchesType = activeTab === 'all' || item.type === activeTab;

    // 分类过滤
    const matchesCategory = category === 'all' || item.category === category;

    // 官方过滤
    const matchesOfficial = isOfficial === 'all' || 
      (isOfficial === 'true' && item.is_official) ||
      (isOfficial === 'false' && !item.is_official);

    return matchesSearch && matchesType && matchesCategory && matchesOfficial;
  });

  // 获取所有分类
  const categories = ['all', ...Array.from(new Set(items.map(item => item.category)))];

  // 类型图标映射
  const typeIcons = {
    connector: <Plug className="h-5 w-5 text-blue-500" />,
    skill: <Brain className="h-5 w-5 text-purple-500" />,
    plugin: <Layers className="h-5 w-5 text-green-500" />
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

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">应用市场</h1>
          <p className="text-gray-500 mt-1">发现和安装连接器、AI技能和扩展插件，提升IT服务管理效率</p>
        </div>
        <Link href="/installations">
          <Button variant="default">
            <PlusCircle className="h-4 w-4 mr-2" />
            我的应用
          </Button>
        </Link>
      </div>

      {/* 搜索和过滤区 */}
      <div className="bg-white rounded-lg shadow-sm p-4 mb-6 space-y-4">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
              <Input
                placeholder="搜索应用名称、描述或标签..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
          <div className="w-full md:w-48">
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue placeholder="分类" />
              </SelectTrigger>
              <SelectContent>
                {categories.map(cat => (
                  <SelectItem key={cat} value={cat}>
                    {cat === 'all' ? '全部分类' : cat}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="w-full md:w-48">
            <Select value={isOfficial} onValueChange={setIsOfficial}>
              <SelectTrigger>
                <SelectValue placeholder="提供商" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部提供商</SelectItem>
                <SelectItem value="true">官方</SelectItem>
                <SelectItem value="false">社区</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <Tabs defaultValue="all" value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="all">全部</TabsTrigger>
            <TabsTrigger value="connector">连接器</TabsTrigger>
            <TabsTrigger value="skill">AI技能</TabsTrigger>
            <TabsTrigger value="plugin">插件</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* 应用列表 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {filteredItems.map(item => (
          <Card key={item.id} className="hover:shadow-md transition-shadow">
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                    {item.icon_url ? (
                      <img src={item.icon_url} alt={item.title} className="w-full h-full object-cover" />
                    ) : (
                      typeIcons[item.type]
                    )}
                  </div>
                  <div>
                    <CardTitle className="text-lg">{item.title}</CardTitle>
                    <CardDescription className="flex items-center gap-2 mt-1">
                      <Badge variant="secondary" className="text-xs">
                        {typeNames[item.type]}
                      </Badge>
                      {item.is_official && (
                        <Badge variant="default" className="bg-blue-100 text-blue-800 hover:bg-blue-200 text-xs">
                          官方
                        </Badge>
                      )}
                    </CardDescription>
                  </div>
                </div>
              </div>
            </CardHeader>
            <CardContent className="pb-2">
              <p className="text-sm text-gray-600 line-clamp-2 h-10">
                {item.description}
              </p>
              <div className="flex flex-wrap gap-1 mt-2">
                {item.tags.slice(0, 3).map(tag => (
                  <Badge key={tag} variant="outline" className="text-xs">
                    {tag}
                  </Badge>
                ))}
                {item.tags.length > 3 && (
                  <span className="text-xs text-gray-500">+{item.tags.length - 3}</span>
                )}
              </div>
            </CardContent>
            <CardFooter className="flex items-center justify-between pt-2 border-t">
              <div className="flex items-center gap-3 text-sm text-gray-500">
                <div className="flex items-center gap-1">
                  <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                  <span>{item.rating.toFixed(1)}</span>
                </div>
                <div className="flex items-center gap-1">
                  <Download className="h-4 w-4" />
                  <span>{item.install_count}</span>
                </div>
                <div className="text-xs text-gray-400">
                  v{item.latest_version}
                </div>
              </div>
              <Link href={`/marketplace/${item.id}`}>
                <Button size="sm">详情</Button>
              </Link>
            </CardFooter>
          </Card>
        ))}
      </div>

      {filteredItems.length === 0 && (
        <div className="bg-white rounded-lg shadow-sm p-10 text-center">
          <div className="flex flex-col items-center justify-center">
            <Search className="h-12 w-12 text-gray-300 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">没有找到匹配的应用</h3>
            <p className="text-gray-500">尝试调整搜索条件或过滤选项</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default MarketplacePage;
