'use client';

import {
  UserPlus,
  UserMinus,
  Plus,
  Users,
  Shield,
  Search,
  Edit,
  Trash2,
  MoreHorizontal,
} from 'lucide-react';

import React, { useState } from 'react';
import { App, Modal } from 'antd';
// 用户组数据类型定义
interface Group {
  id: number;
  name: string;
  description: string;
  type: 'system' | 'custom';
  memberCount: number;
  permissions: string[];
  createdAt: string;
  updatedAt: string;
  status: 'active' | 'inactive';
}

const GroupManagement = () => {
  const { message, modal } = App.useApp();
  const [groups, setGroups] = useState<Group[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [showModal, setShowModal] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  // 加载用户组数据
  React.useEffect(() => {
    loadGroups();
  }, []);

  const loadGroups = async () => {
    try {
      const { RoleAPI } = await import('@/lib/api/role-api');
      const response = await RoleAPI.getRoles({ size: 100 }); // 获取所有角色
      
      const mappedGroups: Group[] = response.roles.map((role: any) => ({
        id: role.id,
        name: role.name,
        description: role.description || '',
        type: role.isSystem ? 'system' : 'custom',
        memberCount: role.userCount || 0,
        permissions: role.permissions || [],
        createdAt: role.createdAt,
        updatedAt: role.updatedAt,
        status: role.status || 'active'
      }));
      
      setGroups(mappedGroups);
    } catch (error) {
      console.error('加载用户组失败:', error);
      message.error('加载用户组失败');
    }
  };

  // 过滤用户组
  const filteredGroups = groups.filter(group => {
    const matchesSearch =
      group.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      group.description.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesType = typeFilter === 'all' || group.type === typeFilter;
    const matchesStatus = statusFilter === 'all' || group.status === statusFilter;

    return matchesSearch && matchesType && matchesStatus;
  });

  // 分页计算
  const totalPages = Math.ceil(filteredGroups.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const paginatedGroups = filteredGroups.slice(startIndex, startIndex + itemsPerPage);

  // 状态样式映射
  const getStatusStyle = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-800';
      case 'inactive':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  // 类型样式映射
  const getTypeStyle = (type: string) => {
    switch (type) {
      case 'system':
        return 'bg-blue-100 text-blue-800';
      case 'custom':
        return 'bg-purple-100 text-purple-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  // 状态文本映射
  const getStatusText = (status: string) => {
    switch (status) {
      case 'active':
        return '活跃';
      case 'inactive':
        return '停用';
      default:
        return '未知';
    }
  };

  // 类型文本映射
  const getTypeText = (type: string) => {
    switch (type) {
      case 'system':
        return '系统组';
      case 'custom':
        return '自定义组';
      default:
        return '未知';
    }
  };

  // 处理用户组操作
  const handleEditGroup = (group: Group) => {
    setSelectedGroup(group);
    setShowModal(true);
  };

  const handleDeleteGroup = (groupId: number) => {
    const group = groups.find(g => g.id === groupId);
    if (group?.type === 'system') {
      message.warning('系统组不能删除！');
      return;
    }
    modal.confirm({
      title: '确认删除',
      content: '确定要删除这个用户组吗？',
      okText: '确认',
      cancelText: '取消',
      onOk: () => {
        setGroups(groups.filter(g => g.id !== groupId));
        message.success('删除成功');
      },
    });
  };

  const handleToggleStatus = (groupId: number) => {
    setGroups(
      groups.map(group => {
        if (group.id === groupId) {
          return {
            ...group,
            status: group.status === 'active' ? 'inactive' : 'active',
          };
        }
        return group;
      })
    );
  };

  // 统计数据
  const activeGroups = groups.filter(g => g.status === 'active').length;
  const systemGroups = groups.filter(g => g.type === 'system').length;
  const totalMembers = groups.reduce((sum, g) => sum + g.memberCount, 0);

  return (
    <div className='space-y-6'>
      {/* 页面头部 */}
      <div className='flex justify-between items-center'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900'>用户组管理</h1>
          <p className='text-gray-600 mt-1'>管理用户组、权限分配和成员关系</p>
        </div>
        <button
          onClick={() => {
            setSelectedGroup(null);
            setShowModal(true);
          }}
          className='bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors'
        >
          <Plus className='w-4 h-4' />
          新建用户组
        </button>
      </div>

      {/* 统计卡片 */}
      <div className='grid grid-cols-1 md:grid-cols-4 gap-6'>
        <div className='bg-white p-6 rounded-lg shadow-sm border'>
          <div className='flex items-center justify-between'>
            <div>
              <p className='text-sm font-medium text-gray-600'>总用户组数</p>
              <p className='text-2xl font-bold text-gray-900'>{groups.length}</p>
            </div>
            <div className='p-3 bg-blue-100 rounded-full'>
              <Users className='w-6 h-6 text-blue-600' />
            </div>
          </div>
        </div>

        <div className='bg-white p-6 rounded-lg shadow-sm border'>
          <div className='flex items-center justify-between'>
            <div>
              <p className='text-sm font-medium text-gray-600'>活跃用户组</p>
              <p className='text-2xl font-bold text-green-600'>{activeGroups}</p>
            </div>
            <div className='p-3 bg-green-100 rounded-full'>
              <Shield className='w-6 h-6 text-green-600' />
            </div>
          </div>
        </div>

        <div className='bg-white p-6 rounded-lg shadow-sm border'>
          <div className='flex items-center justify-between'>
            <div>
              <p className='text-sm font-medium text-gray-600'>系统用户组</p>
              <p className='text-2xl font-bold text-purple-600'>{systemGroups}</p>
            </div>
            <div className='p-3 bg-purple-100 rounded-full'>
              <Shield className='w-6 h-6 text-purple-600' />
            </div>
          </div>
        </div>

        <div className='bg-white p-6 rounded-lg shadow-sm border'>
          <div className='flex items-center justify-between'>
            <div>
              <p className='text-sm font-medium text-gray-600'>总成员数</p>
              <p className='text-2xl font-bold text-orange-600'>{totalMembers}</p>
            </div>
            <div className='p-3 bg-orange-100 rounded-full'>
              <UserPlus className='w-6 h-6 text-orange-600' />
            </div>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className='bg-white p-6 rounded-lg shadow-sm border'>
        <div className='flex flex-col md:flex-row gap-4'>
          <div className='flex-1'>
            <div className='relative'>
              <Search className='absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4' />
              <input
                type='text'
                placeholder='搜索用户组名称或描述...'
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className='w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent'
              />
            </div>
          </div>
          <div className='flex gap-4'>
            <select
              value={typeFilter}
              onChange={e => setTypeFilter(e.target.value)}
              className='px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent'
            >
              <option value='all'>所有类型</option>
              <option value='system'>系统组</option>
              <option value='custom'>自定义组</option>
            </select>
            <select
              value={statusFilter}
              onChange={e => setStatusFilter(e.target.value)}
              className='px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent'
            >
              <option value='all'>所有状态</option>
              <option value='active'>活跃</option>
              <option value='inactive'>停用</option>
            </select>
          </div>
        </div>
      </div>

      {/* 用户组列表 */}
      <div className='bg-white rounded-lg shadow-sm border overflow-hidden'>
        <div className='overflow-x-auto'>
          <table className='w-full'>
            <thead className='bg-gray-50 border-b border-gray-200'>
              <tr>
                <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  用户组信息
                </th>
                <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  类型
                </th>
                <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  成员数量
                </th>
                <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  权限
                </th>
                <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  状态
                </th>
                <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  更新时间
                </th>
                <th className='px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider'>
                  操作
                </th>
              </tr>
            </thead>
            <tbody className='bg-white divide-y divide-gray-200'>
              {paginatedGroups.map(group => (
                <tr key={group.id} className='hover:bg-gray-50'>
                  <td className='px-6 py-4 whitespace-nowrap'>
                    <div>
                      <div className='text-sm font-medium text-gray-900'>{group.name}</div>
                      <div className='text-sm text-gray-500'>{group.description}</div>
                    </div>
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap'>
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getTypeStyle(
                        group.type
                      )}`}
                    >
                      {getTypeText(group.type)}
                    </span>
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap'>
                    <div className='flex items-center'>
                      <Users className='w-4 h-4 text-gray-400 mr-2' />
                      <span className='text-sm text-gray-900'>{group.memberCount}</span>
                    </div>
                  </td>
                  <td className='px-6 py-4'>
                    <div className='flex flex-wrap gap-1'>
                      {group.permissions.slice(0, 2).map((permission, index) => (
                        <span
                          key={index}
                          className='inline-flex px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded'
                        >
                          {permission}
                        </span>
                      ))}
                      {group.permissions.length > 2 && (
                        <span className='inline-flex px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded'>
                          +{group.permissions.length - 2}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap'>
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusStyle(
                        group.status
                      )}`}
                    >
                      {getStatusText(group.status)}
                    </span>
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap text-sm text-gray-500'>
                    {group.updatedAt}
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap text-right text-sm font-medium'>
                    <div className='flex items-center justify-end gap-2'>
                      <button
                        onClick={() => handleEditGroup(group)}
                        className='text-blue-600 hover:text-blue-900 p-1 rounded'
                        title='编辑'
                      >
                        <Edit className='w-4 h-4' />
                      </button>
                      <button
                        onClick={() => handleToggleStatus(group.id)}
                        className={`p-1 rounded ${
                          group.status === 'active'
                            ? 'text-red-600 hover:text-red-900'
                            : 'text-green-600 hover:text-green-900'
                        }`}
                        title={group.status === 'active' ? '停用' : '启用'}
                      >
                        {group.status === 'active' ? (
                          <UserMinus className='w-4 h-4' />
                        ) : (
                          <UserPlus className='w-4 h-4' />
                        )}
                      </button>
                      {group.type !== 'system' && (
                        <button
                          onClick={() => handleDeleteGroup(group.id)}
                          className='text-red-600 hover:text-red-900 p-1 rounded'
                          title='删除'
                        >
                          <Trash2 className='w-4 h-4' />
                        </button>
                      )}
                      <button
                        className='text-gray-600 hover:text-gray-900 p-1 rounded'
                        title='更多操作'
                      >
                        <MoreHorizontal className='w-4 h-4' />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* 分页 */}
        {totalPages > 1 && (
          <div className='bg-white px-4 py-3 border-t border-gray-200 sm:px-6'>
            <div className='flex items-center justify-between'>
              <div className='flex-1 flex justify-between sm:hidden'>
                <button
                  onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                  disabled={currentPage === 1}
                  className='relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed'
                >
                  上一页
                </button>
                <button
                  onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                  disabled={currentPage === totalPages}
                  className='ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed'
                >
                  下一页
                </button>
              </div>
              <div className='hidden sm:flex-1 sm:flex sm:items-center sm:justify-between'>
                <div>
                  <p className='text-sm text-gray-700'>
                    显示第 <span className='font-medium'>{startIndex + 1}</span> 到{' '}
                    <span className='font-medium'>
                      {Math.min(startIndex + itemsPerPage, filteredGroups.length)}
                    </span>{' '}
                    条，共 <span className='font-medium'>{filteredGroups.length}</span> 条记录
                  </p>
                </div>
                <div>
                  <nav className='relative z-0 inline-flex rounded-md shadow-sm -space-x-px'>
                    <button
                      onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                      disabled={currentPage === 1}
                      className='relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed'
                    >
                      上一页
                    </button>
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
                      <button
                        key={page}
                        onClick={() => setCurrentPage(page)}
                        className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${
                          currentPage === page
                            ? 'z-10 bg-blue-50 border-blue-500 text-blue-600'
                            : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'
                        }`}
                      >
                        {page}
                      </button>
                    ))}
                    <button
                      onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                      disabled={currentPage === totalPages}
                      className='relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed'
                    >
                      下一页
                    </button>
                  </nav>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 模态框占位符 */}
      {showModal && (
        <div className='fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50'>
          <div className='bg-white rounded-lg p-6 w-full max-w-md'>
            <h3 className='text-lg font-medium text-gray-900 mb-4'>
              {selectedGroup ? '编辑用户组' : '新建用户组'}
            </h3>
            <p className='text-gray-600 mb-4'>用户组编辑功能正在开发中...</p>
            <div className='flex justify-end gap-2'>
              <button
                onClick={() => setShowModal(false)}
                className='px-4 py-2 text-gray-700 bg-gray-200 rounded-lg hover:bg-gray-300'
              >
                取消
              </button>
              <button
                onClick={() => setShowModal(false)}
                className='px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700'
              >
                确定
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default GroupManagement;
