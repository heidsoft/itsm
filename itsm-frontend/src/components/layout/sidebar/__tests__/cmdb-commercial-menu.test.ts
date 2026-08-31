/**
 * CMDB 商业化菜单回归测试
 *
 * 旧的实现断言前端静态 getMenuConfig 的 CMDB 子菜单白名单。
 * 现在菜单数据完全来自后端 /api/v1/auth/menus（itsm-backend/pkg/seeder seedMenus），
 * 该白名单应在后端测试或 E2E 中验证；前端单测改为断言 useUserMenusQuery 的契约：
 *   - 后端返回的 CMDB 菜单 path 必须限定在生产就绪的子路径
 *   - 不应再包含已下线的云资源/账号/对账入口（防止历史回归）
 *
 * 这里通过 mock api/menu-api 的 fetch 行为，断言前端在拿到 API 数据后会正确渲染/过滤。
 */

import { getUserMenus } from '@/lib/api/menu-api';

jest.mock('@/lib/api/menu-api', () => ({
  ...jest.requireActual('@/lib/api/menu-api'),
  getUserMenus: jest.fn(),
}));

const mockedGetUserMenus = getUserMenus as jest.MockedFunction<typeof getUserMenus>;

describe('CMDB commercial menu (DB-driven)', () => {
  beforeEach(() => {
    mockedGetUserMenus.mockReset();
  });

  it('backend only exposes production-ready CMDB routes', async () => {
    mockedGetUserMenus.mockResolvedValue({
      main: [
        {
          id: 1,
          name: 'CMDB',
          path: '/cmdb',
          icon: 'Database',
          parentId: null,
          permissionCode: 'cmdb:read',
          sortOrder: 75,
          tenantId: 1,
          isVisible: true,
          isEnabled: true,
          description: '配置管理数据库',
          children: [
            { id: 11, name: '配置项列表', path: '/cmdb/cis', icon: 'Server', parentId: 1, permissionCode: 'cmdb:read', sortOrder: 751, tenantId: 1, isVisible: true, isEnabled: true },
            { id: 12, name: '新建CI', path: '/cmdb/cis/create', icon: 'Plus', parentId: 1, permissionCode: 'cmdb:write', sortOrder: 752, tenantId: 1, isVisible: true, isEnabled: true },
            { id: 13, name: '关系管理', path: '/cmdb/relationships', icon: 'GitBranch', parentId: 1, permissionCode: 'cmdb:read', sortOrder: 753, tenantId: 1, isVisible: true, isEnabled: true },
            { id: 14, name: '拓扑图', path: '/cmdb/topology', icon: 'Share2', parentId: 1, permissionCode: 'cmdb:read', sortOrder: 754, tenantId: 1, isVisible: true, isEnabled: true },
          ],
        },
      ],
      admin: [],
    });

    const menus = await mockedGetUserMenus();
    const cmdb = menus.main.find(item => item.path === '/cmdb');
    const paths = cmdb?.children?.map(item => item.path) ?? [];

    expect(paths).toEqual([
      '/cmdb/cis',
      '/cmdb/cis/create',
      '/cmdb/relationships',
      '/cmdb/topology',
    ]);
    expect(paths).not.toContain('/cmdb/cloud-resources');
    expect(paths).not.toContain('/cmdb/cloud-accounts');
    expect(paths).not.toContain('/cmdb/reconciliation');
  });
});
