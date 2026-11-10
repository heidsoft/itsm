'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle
} from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/Select';
import { Badge } from '@/components/ui/Badge';
import {
  MARKETPLACE_TYPE_LABELS,
  marketplaceCategoryLabel,
  marketplaceTagLabel,
} from '@/lib/marketplaceLabels';
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
import { httpClient } from '@/lib/api/http-client';

// 组件类型定义
interface MarketplaceItem {
  id: number;
  name: string;
  title: string;
  type: 'connector' | 'skill' | 'plugin';
  provider: string;
  description: string;
  iconUrl: string;
  rating: number;
  installCount: number;
  category: string;
  tags: string[];
  isOfficial: boolean;
  latestVersion: string;
}

const MarketplacePage = () => {
  const [items, setItems] = useState<MarketplaceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState('all');
  const [category, setCategory] = useState('all');
  const [isOfficial, setIsOfficial] = useState<string>('all');
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const fetchItems = async () => {
      setLoading(true);
      setLoadError(null);
      try {
        const res = await httpClient.get<{
          items: MarketplaceItem[];
          total: number;
          page: number;
          pageSize: number;
        }>('/api/v1/marketplace/items', { page: 1, pageSize: 100 });
        if (cancelled) return;
        // 后端响应字段已统一为 camelCase。
        // 仅保留对前端有用的字段并对缺失字段做兜底。
        const normalized: MarketplaceItem[] = (res?.items || []).map((it: MarketplaceItem) => ({
          id: it.id,
          name: it.name,
          title: it.title || it.name,
          type: (it.type || 'plugin') as MarketplaceItem['type'],
          provider: it.provider || '',
          description: it.description || '',
          iconUrl: it.iconUrl || '',
          rating: typeof it.rating === 'number' ? it.rating : 0,
          installCount: it.installCount ?? 0,
          category: it.category || 'Uncategorized',
          tags: Array.isArray(it.tags) ? it.tags : [],
          isOfficial: Boolean(it.isOfficial),
          latestVersion: it.latestVersion || '1.0.0',
        }));
        setItems(normalized);
      } catch (error: unknown) {
        if (cancelled) return;
        console.error('Failed to fetch marketplace items:', error);
        setLoadError(error instanceof Error ? error.message : 'Failed to load the app store, please try again later.');
        setItems([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    fetchItems();
    return () => {
      cancelled = true;
    };
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
      (isOfficial === 'true' && item.isOfficial) ||
      (isOfficial === 'false' && !item.isOfficial);

    return matchesSearch && matchesType && matchesCategory && matchesOfficial;
  });

  // 获取所有分类（只保留真实分类 code，`all` 由「全部分类」选项单独提供，避免重复）
  const categories = Array.from(new Set(items.map(item => item.category).filter(Boolean)));

  // 类型图标映射
  const typeIcons = {
    connector: <Plug className="h-5 w-5 text-blue-500" />,
    skill: <Brain className="h-5 w-5 text-purple-500" />,
    plugin: <Layers className="h-5 w-5 text-green-500" />
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
          <h1 className="text-3xl font-bold tracking-tight">ApplyMarketplace</h1>
          <p className="text-gray-500 mt-1">Discover and install connectors, AI skills and extensions to improve IT service management efficiency</p>
        </div>
        <Link href="/installations">
          <Button variant="default">
            <PlusCircle className="h-4 w-4 mr-2" />

            My application
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
                placeholder="Search for app name, description or tags..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
          <div className="w-full md:w-48">
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue placeholder="Category" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All categories</SelectItem>
                {categories.map(cat => (
                  <SelectItem key={cat} value={cat}>
                    {marketplaceCategoryLabel(cat)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="w-full md:w-48">
            <Select value={isOfficial} onValueChange={setIsOfficial}>
              <SelectTrigger>
                <SelectValue placeholder="Provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All providers</SelectItem>
                <SelectItem value="true">Official</SelectItem>
                <SelectItem value="false">Community</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="w-full md:w-48">
            <Select value={activeTab} onValueChange={setActiveTab}>
              <SelectTrigger>
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="connector">Connectors</SelectItem>
                <SelectItem value="skill">AI skills</SelectItem>
                <SelectItem value="plugin">Plug-in</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {/* 应用列表 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {filteredItems.map(item => (
          <Card key={item.id} className="hover:shadow-md transition-shadow">
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                    {item.iconUrl ? (
                      <img src={item.iconUrl} alt={item.title} className="w-full h-full object-cover" />
                    ) : (
                      typeIcons[item.type]
                    )}
                  </div>
                  <div>
                    <CardTitle className="text-lg">{item.title}</CardTitle>
                    <CardDescription className="flex items-center gap-2 mt-1">
                      <Badge variant="secondary" className="text-xs">
                        {MARKETPLACE_TYPE_LABELS[item.type] ?? item.type}
                      </Badge>
                      {item.isOfficial && (
                        <Badge variant="default" className="bg-blue-100 text-blue-800 hover:bg-blue-200 text-xs">

                          Official
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
                    {marketplaceTagLabel(tag)}
                  </Badge>
                ))}
                {item.tags.length > 3 && (
                  <span className="text-xs text-gray-500">+{item.tags.length - 3}</span>
                )}
              </div>
            </CardContent>
            <CardFooter className="flex items-center justify-between pt-2 border-t">
              <div className="flex items-center gap-3 text-sm text-gray-500">
                {item.rating > 0 && (
                  <div className="flex items-center gap-1">
                    <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                    <span>{item.rating.toFixed(1)}</span>
                  </div>
                )}
                {item.installCount > 0 && (
                  <div className="flex items-center gap-1">
                    <Download className="h-4 w-4" />
                    <span>{item.installCount}</span>
                  </div>
                )}
                <div className="text-xs text-gray-400">
                  v{item.latestVersion}
                </div>
              </div>
              <Link href={`/marketplace/${item.id}`}>
                <Button size="sm">Details</Button>
              </Link>
            </CardFooter>
          </Card>
        ))}
      </div>

      {filteredItems.length === 0 && (
        <div className="bg-white rounded-lg shadow-sm p-10 text-center">
          <div className="flex flex-col items-center justify-center">
            <Search className="h-12 w-12 text-gray-300 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No matching application found</h3>
            <p className="text-gray-500">Try adjusting search criteria or filtering options</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default MarketplacePage;
