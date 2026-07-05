'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import {
  AlertCircle,
  ArrowLeft,
  Calendar,
  CheckCircle2,
  Code,
  Download,
  Globe,
  Shield,
  Star,
  Tag,
  Terminal,
  User,
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/Tabs';
import type {
  MarketplaceItem,
  TenantInstallation,
} from '@/lib/services/marketplace-service';
import marketplaceService from '@/lib/services/marketplace-service';

const typeNames = {
  connector: '连接器',
  skill: '技能',
  plugin: '插件',
};

const formatDate = (value?: string) => (value ? new Date(value).toLocaleDateString() : '暂无');

const normalizeMarkdown = (text?: string) =>
  (text || '暂无详细介绍').split('\n').filter(line => line.trim().length > 0);

const MarketplaceDetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const itemId = Number(params.id);

  const [item, setItem] = useState<MarketplaceItem | null>(null);
  const [installation, setInstallation] = useState<TenantInstallation | null>(null);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const versions = useMemo(() => item?.edges?.versions || [], [item]);
  const isInstalled = Boolean(installation);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      setLoadError(null);
      try {
        const [itemRes, installationRes] = await Promise.all([
          marketplaceService.getItem(itemId),
          marketplaceService.getInstallation(itemId),
        ]);
        if (cancelled) return;
        setItem(itemRes);
        setInstallation(installationRes);
      } catch (error: any) {
        if (cancelled) return;
        console.error('Failed to fetch marketplace item detail:', error);
        setLoadError(error?.message || '加载应用详情失败');
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    if (Number.isFinite(itemId)) {
      load();
    }
    return () => {
      cancelled = true;
    };
  }, [itemId]);

  const handleInstall = async () => {
    setInstalling(true);
    try {
      const installed = await marketplaceService.installItem(itemId);
      setInstallation(installed);
      router.push('/installations');
    } catch (error) {
      console.error('Failed to install item:', error);
      alert(error instanceof Error ? error.message : '安装失败');
    } finally {
      setInstalling(false);
    }
  };

  const handleUninstall = async () => {
    if (!item || !confirm(`确定要卸载「${item.title}」吗？卸载后相关功能将无法使用。`)) return;
    try {
      await marketplaceService.uninstallItem(itemId);
      setInstallation(null);
    } catch (error) {
      console.error('Failed to uninstall item:', error);
      alert(error instanceof Error ? error.message : '卸载失败');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (loadError || !item) {
    return (
      <div className="container mx-auto px-4 py-6">
        <div className="bg-white rounded-lg shadow-sm p-10 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4 mx-auto" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">应用不存在</h3>
          <p className="text-gray-500 mb-4">{loadError || '您访问的应用可能已被下架或删除'}</p>
          <Link href="/marketplace">
            <Button>返回应用市场</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center gap-2 mb-6 text-sm">
        <Link href="/marketplace" className="text-gray-500 hover:text-gray-700 flex items-center gap-1">
          <ArrowLeft className="h-4 w-4" />
          返回市场
        </Link>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <div className="flex flex-col md:flex-row md:items-start gap-4">
                <div className="w-16 h-16 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                  {item.iconUrl ? (
                    <img src={item.iconUrl} alt={item.title} className="w-full h-full object-cover" />
                  ) : (
                    <span className="text-sm">{typeNames[item.type]}</span>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-3 mb-1">
                    <h1 className="text-2xl font-bold">{item.title}</h1>
                    <Badge variant="secondary">{typeNames[item.type]}</Badge>
                    {item.isOfficial && <Badge className="bg-blue-100 text-blue-800 hover:bg-blue-200">官方</Badge>}
                    {item.isFree ? (
                      <Badge className="bg-green-100 text-green-800 hover:bg-green-200">免费</Badge>
                    ) : (
                      <Badge className="bg-orange-100 text-orange-800 hover:bg-orange-200">¥{item.price || 0}</Badge>
                    )}
                  </div>
                  <p className="text-gray-600">{item.description || '暂无描述'}</p>
                  <div className="flex flex-wrap items-center gap-4 mt-3 text-sm text-gray-500">
                    <span className="flex items-center gap-1">
                      <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                      {(item.rating || 0).toFixed(1)}
                    </span>
                    <span className="flex items-center gap-1">
                      <Download className="h-4 w-4" />
                      {item.installCount || 0} 次安装
                    </span>
                    <span className="flex items-center gap-1">
                      <User className="h-4 w-4" />
                      作者：{item.authorName || item.provider}
                    </span>
                    <span className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" />
                      更新于 {formatDate(item.updatedAt)}
                    </span>
                  </div>
                </div>
                <div className="md:text-right">
                  {isInstalled ? (
                    <div className="space-y-2">
                      <div className="flex items-center md:justify-end gap-1 text-green-600">
                        <CheckCircle2 className="h-5 w-5" />
                        <span>已安装</span>
                      </div>
                      <div className="flex gap-2">
                        <Button onClick={() => router.push('/installations')}>管理</Button>
                        <Button variant="destructive" onClick={handleUninstall}>卸载</Button>
                      </div>
                    </div>
                  ) : (
                    <Button size="lg" onClick={handleInstall} disabled={installing}>
                      {installing ? '安装中...' : '立即安装'}
                    </Button>
                  )}
                  <p className="text-xs text-gray-500 mt-2">版本 {item.latestVersion || '1.0.0'}</p>
                </div>
              </div>
            </CardHeader>
          </Card>

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
                <TabsContent value="description" className="mt-0 space-y-4">
                  <div className="prose prose-sm max-w-none">
                    {normalizeMarkdown(item.longDescription || item.description).map((line, index) => (
                      <p key={index}>{line}</p>
                    ))}
                  </div>
                  {(item.tags || []).length > 0 && (
                    <div>
                      <h3 className="text-lg font-medium mb-3">相关标签</h3>
                      <div className="flex flex-wrap gap-2">
                        {(item.tags || []).map(tag => (
                          <Badge key={tag} variant="outline">
                            <Tag className="h-3 w-3 mr-1" />
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <InfoLink icon={<Globe className="h-5 w-5 text-gray-400 mt-0.5" />} label="官方网站" value={item.homepage} />
                    <InfoLink icon={<Code className="h-5 w-5 text-gray-400 mt-0.5" />} label="代码仓库" value={item.repository} />
                    <InfoText icon={<Shield className="h-5 w-5 text-gray-400 mt-0.5" />} label="开源协议" value={item.license || '未声明'} />
                    <InfoText icon={<Terminal className="h-5 w-5 text-gray-400 mt-0.5" />} label="最低系统版本" value={item.minSystemVersion || '未声明'} />
                  </div>
                </TabsContent>

                <TabsContent value="capabilities" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">支持的功能特性</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {(item.capabilities || []).map(capability => (
                      <div key={capability} className="flex items-center gap-2 p-2 bg-gray-50 rounded-md">
                        <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
                        <span>{capability}</span>
                      </div>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="permissions" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">需要的系统权限</h3>
                  <div className="space-y-3">
                    {(item.requiredPermissions || []).map(permission => (
                      <div key={permission} className="flex items-start gap-2 p-2 bg-gray-50 rounded-md">
                        <Shield className="h-5 w-5 text-blue-500 flex-shrink-0 mt-0.5" />
                        <span>{permission}</span>
                      </div>
                    ))}
                    {(item.requiredPermissions || []).length === 0 && <p className="text-sm text-gray-500">暂无额外权限说明</p>}
                  </div>
                </TabsContent>

                <TabsContent value="versions" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">版本历史</h3>
                  <div className="space-y-4">
                    {versions.length > 0 ? versions.map(version => (
                      <div key={version.version} className="border-l-2 border-gray-200 pl-4 pb-4">
                        <div className="flex items-center justify-between mb-1">
                          <div className="font-medium">v{version.version}</div>
                          <div className="text-sm text-gray-500">{formatDate(version.releasedAt)}</div>
                        </div>
                        <p className="text-sm text-gray-600 whitespace-pre-line">{version.changelog || '暂无更新说明'}</p>
                      </div>
                    )) : <p className="text-sm text-gray-500">暂无版本历史</p>}
                  </div>
                </TabsContent>
              </CardContent>
            </Tabs>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>安装信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <InfoRow label="最新版本" value={item.latestVersion || '1.0.0'} />
              <InfoRow label="发布日期" value={formatDate(versions[0]?.releasedAt || item.updatedAt)} />
              <InfoRow label="最低系统要求" value={item.minSystemVersion || '未声明'} />
              <InfoRow label="安装量" value={String(item.installCount || 0)} />
              <div>
                <div className="text-sm font-medium mb-1">分类</div>
                <Badge variant="secondary">{item.category || '未分类'}</Badge>
              </div>
            </CardContent>
            <CardFooter className="border-t pt-4">
              {isInstalled ? (
                <Button className="w-full" onClick={() => router.push('/installations')}>管理已安装应用</Button>
              ) : (
                <Button className="w-full" size="lg" onClick={handleInstall} disabled={installing}>
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
                  {(item.provider || item.authorName || item.title).charAt(0)}
                </div>
                <div>
                  <div className="font-medium">{item.provider || item.authorName || '未知提供商'}</div>
                  {item.isOfficial && <Badge className="mt-1 bg-blue-100 text-blue-800 hover:bg-blue-200">官方认证</Badge>}
                </div>
              </div>
              <p className="text-sm text-gray-600">
                {item.isOfficial ? '由ITSM官方团队开发和维护，确保兼容性和安全性。' : '由社区开发者贡献，经过官方审核验证。'}
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

const InfoRow = ({ label, value }: { label: string; value: string }) => (
  <div>
    <div className="text-sm font-medium mb-1">{label}</div>
    <div className="text-gray-600">{value}</div>
  </div>
);

const InfoText = ({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) => (
  <div className="flex items-start gap-2">
    {icon}
    <div>
      <div className="text-sm font-medium">{label}</div>
      <div className="text-sm text-gray-600 break-all">{value}</div>
    </div>
  </div>
);

const InfoLink = ({ icon, label, value }: { icon: React.ReactNode; label: string; value?: string }) => (
  <div className="flex items-start gap-2">
    {icon}
    <div>
      <div className="text-sm font-medium">{label}</div>
      {value ? (
        <a href={value} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline break-all">
          {value}
        </a>
      ) : (
        <div className="text-sm text-gray-600">未提供</div>
      )}
    </div>
  </div>
);

export default MarketplaceDetailPage;
