/**
 * ITSM前端架构 - 模块化工具函数
 * 
 * 工单管理模块工具函数
 * 提供可复用的业务逻辑和数据处理函数
 */

import { Ticket, TicketStatus, TicketPriority, TicketType, User, Comment, Activity } from './types/ticket-types';

// ==================== 状态管理工具 ====================

/**
 * 状态管理工具类
 */
export class StateUtils {
  /**
   * 深度合并对象
   */
  static deepMerge<T extends Record<string, any>>(target: T, source: Partial<T>): T {
    const result = { ...target };
    
    for (const key in source) {
      if (source.hasOwnProperty(key)) {
        const sourceValue = source[key];
        const targetValue = result[key];
        
        if (this.isObject(sourceValue) && this.isObject(targetValue)) {
          result[key] = this.deepMerge(targetValue, sourceValue);
        } else {
          result[key] = sourceValue;
        }
      }
    }
    
    return result;
  }

  /**
   * 检查是否为对象
   */
  private static isObject(value: any): value is Record<string, any> {
    return value !== null && typeof value === 'object' && !Array.isArray(value);
  }

  /**
   * 创建状态更新函数
   */
  static createStateUpdater<T>(setState: (updater: (state: T) => T) => void) {
    return (updates: Partial<T>) => {
      setState(state => this.deepMerge(state, updates));
    };
  }

  /**
   * 防抖函数
   */
  static debounce<T extends (...args: any[]) => any>(
    func: T,
    wait: number
  ): (...args: Parameters<T>) => void {
    let timeout: NodeJS.Timeout;
    
    return (...args: Parameters<T>) => {
      clearTimeout(timeout);
      timeout = setTimeout(() => func(...args), wait);
    };
  }

  /**
   * 节流函数
   */
  static throttle<T extends (...args: any[]) => any>(
    func: T,
    limit: number
  ): (...args: Parameters<T>) => void {
    let inThrottle: boolean;
    
    return (...args: Parameters<T>) => {
      if (!inThrottle) {
        func(...args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
  }
}

// ==================== 工单业务逻辑工具 ====================

/**
 * 工单业务逻辑工具类
 */
export class TicketUtils {
  /**
   * 获取工单状态显示文本
   */
  static getStatusText(status: TicketStatus): string {
    const statusMap: Record<TicketStatus, string> = {
      [TicketStatus.OPEN]: '开放',
      [TicketStatus.IN_PROGRESS]: '处理中',
      [TicketStatus.RESOLVED]: '已解决',
      [TicketStatus.CLOSED]: '已关闭',
      [TicketStatus.CANCELLED]: '已取消',
    };
    
    return statusMap[status] || status;
  }

  /**
   * 获取工单优先级显示文本
   */
  static getPriorityText(priority: TicketPriority): string {
    const priorityMap: Record<TicketPriority, string> = {
      [TicketPriority.LOW]: '低',
      [TicketPriority.MEDIUM]: '中',
      [TicketPriority.HIGH]: '高',
      [TicketPriority.URGENT]: '紧急',
      [TicketPriority.CRITICAL]: '严重',
    };
    
    return priorityMap[priority] || priority;
  }

  /**
   * 获取工单类型显示文本
   */
  static getTypeText(type: TicketType): string {
    const typeMap: Record<TicketType, string> = {
      [TicketType.INCIDENT]: '事件',
      [TicketType.SERVICE_REQUEST]: '服务请求',
      [TicketType.PROBLEM]: '问题',
      [TicketType.CHANGE]: '变更',
      [TicketType.QUESTION]: '咨询',
    };
    
    return typeMap[type] || type;
  }

  /**
   * 获取工单状态颜色
   */
  static getStatusColor(status: TicketStatus): string {
    const colorMap: Record<TicketStatus, string> = {
      [TicketStatus.OPEN]: 'blue',
      [TicketStatus.IN_PROGRESS]: 'orange',
      [TicketStatus.RESOLVED]: 'green',
      [TicketStatus.CLOSED]: 'gray',
      [TicketStatus.CANCELLED]: 'red',
    };
    
    return colorMap[status] || 'default';
  }

  /**
   * 获取工单优先级颜色
   */
  static getPriorityColor(priority: TicketPriority): string {
    const colorMap: Record<TicketPriority, string> = {
      [TicketPriority.LOW]: 'green',
      [TicketPriority.MEDIUM]: 'blue',
      [TicketPriority.HIGH]: 'orange',
      [TicketPriority.URGENT]: 'red',
      [TicketPriority.CRITICAL]: 'purple',
    };
    
    return colorMap[priority] || 'default';
  }

  /**
   * 检查工单是否过期
   */
  static isOverdue(ticket: Ticket): boolean {
    if (!ticket.due_date) return false;
    
    const dueDate = new Date(ticket.due_date);
    const now = new Date();
    
    return dueDate < now && ticket.status !== TicketStatus.CLOSED && ticket.status !== TicketStatus.RESOLVED;
  }

  /**
   * 计算工单处理时间
   */
  static getProcessingTime(ticket: Ticket): number | null {
    if (!ticket.resolved_at) return null;
    
    const createdDate = new Date(ticket.created_at);
    const resolvedDate = new Date(ticket.resolved_at);
    
    return resolvedDate.getTime() - createdDate.getTime();
  }

  /**
   * 格式化处理时间
   */
  static formatProcessingTime(milliseconds: number): string {
    const days = Math.floor(milliseconds / (1000 * 60 * 60 * 24));
    const hours = Math.floor((milliseconds % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((milliseconds % (1000 * 60 * 60)) / (1000 * 60));
    
    if (days > 0) {
      return `${days}天${hours}小时${minutes}分钟`;
    } else if (hours > 0) {
      return `${hours}小时${minutes}分钟`;
    } else {
      return `${minutes}分钟`;
    }
  }

  /**
   * 获取工单进度百分比
   */
  static getProgressPercentage(ticket: Ticket): number {
    const statusProgress: Record<TicketStatus, number> = {
      [TicketStatus.OPEN]: 0,
      [TicketStatus.IN_PROGRESS]: 50,
      [TicketStatus.RESOLVED]: 90,
      [TicketStatus.CLOSED]: 100,
      [TicketStatus.CANCELLED]: 100,
    };
    
    return statusProgress[ticket.status] || 0;
  }

  /**
   * 检查工单是否可以编辑
   */
  static canEdit(ticket: Ticket, user: User): boolean {
    // 已关闭或已取消的工单不能编辑
    if (ticket.status === TicketStatus.CLOSED || ticket.status === TicketStatus.CANCELLED) {
      return false;
    }
    
    // 管理员可以编辑所有工单
    if (user.role === 'admin') {
      return true;
    }
    
    // 工单创建者可以编辑
    if (ticket.reporter_id === user.id) {
      return true;
    }
    
    // 工单处理者可以编辑
    if (ticket.assignee_id === user.id) {
      return true;
    }
    
    return false;
  }

  /**
   * 检查工单是否可以删除
   */
  static canDelete(ticket: Ticket, user: User): boolean {
    // 只有管理员可以删除工单
    if (user.role === 'admin') {
      return true;
    }
    
    // 工单创建者可以删除未分配的工单
    if (ticket.reporter_id === user.id && !ticket.assignee_id) {
      return true;
    }
    
    return false;
  }

  /**
   * 获取工单下一步可执行的操作
   */
  static getAvailableActions(ticket: Ticket, user: User): string[] {
    const actions: string[] = [];
    
    switch (ticket.status) {
      case TicketStatus.OPEN:
        actions.push('assign', 'resolve', 'close');
        break;
      case TicketStatus.IN_PROGRESS:
        actions.push('resolve', 'close', 'reassign');
        break;
      case TicketStatus.RESOLVED:
        actions.push('close', 'reopen');
        break;
      case TicketStatus.CLOSED:
        actions.push('reopen');
        break;
      case TicketStatus.CANCELLED:
        // 已取消的工单不能执行任何操作
        break;
    }
    
    // 根据用户权限过滤操作
    if (!user.permissions.includes('ticket:assign') && actions.includes('assign')) {
      actions.splice(actions.indexOf('assign'), 1);
    }
    
    if (!user.permissions.includes('ticket:resolve') && actions.includes('resolve')) {
      actions.splice(actions.indexOf('resolve'), 1);
    }
    
    return actions;
  }
}

// ==================== 数据处理工具 ====================

/**
 * 数据处理工具类
 */
export class DataUtils {
  /**
   * 过滤工单列表
   */
  static filterTickets(tickets: Ticket[], filters: Record<string, any>): Ticket[] {
    return tickets.filter(ticket => {
      // 状态过滤
      if (filters.status && filters.status.length > 0) {
        if (!filters.status.includes(ticket.status)) {
          return false;
        }
      }
      
      // 优先级过滤
      if (filters.priority && filters.priority.length > 0) {
        if (!filters.priority.includes(ticket.priority)) {
          return false;
        }
      }
      
      // 类型过滤
      if (filters.type && filters.type.length > 0) {
        if (!filters.type.includes(ticket.type)) {
          return false;
        }
      }
      
      // 处理人过滤
      if (filters.assignee_id && filters.assignee_id.length > 0) {
        if (!ticket.assignee_id || !filters.assignee_id.includes(ticket.assignee_id)) {
          return false;
        }
      }
      
      // 创建人过滤
      if (filters.reporter_id && filters.reporter_id.length > 0) {
        if (!filters.reporter_id.includes(ticket.reporter_id)) {
          return false;
        }
      }
      
      // 标签过滤
      if (filters.tags && filters.tags.length > 0) {
        const hasMatchingTag = filters.tags.some((tag: string) => 
          ticket.tags.includes(tag)
        );
        if (!hasMatchingTag) {
          return false;
        }
      }
      
      // 搜索过滤
      if (filters.search) {
        const searchTerm = filters.search.toLowerCase();
        const matchesTitle = ticket.title.toLowerCase().includes(searchTerm);
        const matchesDescription = ticket.description.toLowerCase().includes(searchTerm);
        const matchesTags = ticket.tags.some(tag => 
          tag.toLowerCase().includes(searchTerm)
        );
        
        if (!matchesTitle && !matchesDescription && !matchesTags) {
          return false;
        }
      }
      
      // 日期范围过滤
      if (filters.date_range) {
        const { field, start, end } = filters.date_range;
        const ticketDate = new Date(ticket[field as keyof Ticket] as string);
        const startDate = new Date(start);
        const endDate = new Date(end);
        
        if (ticketDate < startDate || ticketDate > endDate) {
          return false;
        }
      }
      
      return true;
    });
  }

  /**
   * 排序工单列表
   */
  static sortTickets(tickets: Ticket[], sortBy: string, sortOrder: 'asc' | 'desc'): Ticket[] {
    return [...tickets].sort((a, b) => {
      let aValue: any = a[sortBy as keyof Ticket];
      let bValue: any = b[sortBy as keyof Ticket];
      
      // 处理日期字段
      if (sortBy.includes('_at') || sortBy === 'due_date') {
        aValue = new Date(aValue).getTime();
        bValue = new Date(bValue).getTime();
      }
      
      // 处理字符串字段
      if (typeof aValue === 'string') {
        aValue = aValue.toLowerCase();
        bValue = bValue.toLowerCase();
      }
      
      if (sortOrder === 'asc') {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      } else {
        return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
      }
    });
  }

  /**
   * 分页工单列表
   */
  static paginateTickets(tickets: Ticket[], page: number, pageSize: number): {
    data: Ticket[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  } {
    const total = tickets.length;
    const totalPages = Math.ceil(total / pageSize);
    const startIndex = (page - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    const data = tickets.slice(startIndex, endIndex);
    
    return {
      data,
      total,
      page,
      pageSize,
      totalPages,
    };
  }

  /**
   * 计算工单统计信息
   */
  static calculateTicketStats(tickets: Ticket[]): Record<string, number> {
    const stats: Record<string, number> = {
      total: tickets.length,
      open: 0,
      in_progress: 0,
      resolved: 0,
      closed: 0,
      cancelled: 0,
      overdue: 0,
    };
    
    tickets.forEach(ticket => {
      stats[ticket.status]++;
      
      if (TicketUtils.isOverdue(ticket)) {
        stats.overdue++;
      }
    });
    
    return stats;
  }

  /**
   * 按优先级分组工单
   */
  static groupTicketsByPriority(tickets: Ticket[]): Record<TicketPriority, Ticket[]> {
    const groups: Record<TicketPriority, Ticket[]> = {
      [TicketPriority.LOW]: [],
      [TicketPriority.MEDIUM]: [],
      [TicketPriority.HIGH]: [],
      [TicketPriority.URGENT]: [],
      [TicketPriority.CRITICAL]: [],
    };
    
    tickets.forEach(ticket => {
      groups[ticket.priority].push(ticket);
    });
    
    return groups;
  }

  /**
   * 按状态分组工单
   */
  static groupTicketsByStatus(tickets: Ticket[]): Record<TicketStatus, Ticket[]> {
    const groups: Record<TicketStatus, Ticket[]> = {
      [TicketStatus.OPEN]: [],
      [TicketStatus.IN_PROGRESS]: [],
      [TicketStatus.RESOLVED]: [],
      [TicketStatus.CLOSED]: [],
      [TicketStatus.CANCELLED]: [],
    };
    
    tickets.forEach(ticket => {
      groups[ticket.status].push(ticket);
    });
    
    return groups;
  }
}

// ==================== 格式化工具 ====================

/**
 * 格式化工具类
 */
export class FormatUtils {
  /**
   * 格式化日期
   */
  static formatDate(date: string | Date, format: 'short' | 'long' | 'relative' = 'short'): string {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    
    switch (format) {
      case 'short':
        return dateObj.toLocaleDateString('zh-CN');
      case 'long':
        return dateObj.toLocaleString('zh-CN');
      case 'relative':
        return this.formatRelativeTime(dateObj);
      default:
        return dateObj.toLocaleDateString('zh-CN');
    }
  }

  /**
   * 格式化相对时间
   */
  static formatRelativeTime(date: Date): string {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    const weeks = Math.floor(days / 7);
    const months = Math.floor(days / 30);
    const years = Math.floor(days / 365);
    
    if (years > 0) return `${years}年前`;
    if (months > 0) return `${months}个月前`;
    if (weeks > 0) return `${weeks}周前`;
    if (days > 0) return `${days}天前`;
    if (hours > 0) return `${hours}小时前`;
    if (minutes > 0) return `${minutes}分钟前`;
    return '刚刚';
  }

  /**
   * 格式化文件大小
   */
  static formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  }

  /**
   * 格式化数字
   */
  static formatNumber(num: number): string {
    return num.toLocaleString('zh-CN');
  }

  /**
   * 格式化百分比
   */
  static formatPercentage(value: number, total: number): string {
    if (total === 0) return '0%';
    return `${Math.round((value / total) * 100)}%`;
  }

  /**
   * 截断文本
   */
  static truncateText(text: string, maxLength: number): string {
    if (text.length <= maxLength) return text;
    return `${text.substring(0, maxLength)}...`;
  }

  /**
   * 高亮搜索关键词
   */
  static highlightSearchTerm(text: string, searchTerm: string): string {
    if (!searchTerm) return text;
    
    const regex = new RegExp(`(${searchTerm})`, 'gi');
    return text.replace(regex, '<mark>$1</mark>');
  }
}

// ==================== 验证工具 ====================

/**
 * 验证工具类
 */
export class ValidationUtils {
  /**
   * 验证邮箱
   */
  static isValidEmail(email: string): boolean {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  }

  /**
   * 验证手机号
   */
  static isValidPhone(phone: string): boolean {
    const phoneRegex = /^1[3-9]\d{9}$/;
    return phoneRegex.test(phone);
  }

  /**
   * 验证URL
   */
  static isValidUrl(url: string): boolean {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * 验证工单标题
   */
  static isValidTicketTitle(title: string): boolean {
    return title.trim().length >= 5 && title.trim().length <= 200;
  }

  /**
   * 验证工单描述
   */
  static isValidTicketDescription(description: string): boolean {
    return description.trim().length >= 10 && description.trim().length <= 5000;
  }

  /**
   * 验证评论内容
   */
  static isValidCommentContent(content: string): boolean {
    return content.trim().length >= 1 && content.trim().length <= 1000;
  }
}

// ==================== 导出 ====================

export default {
  StateUtils,
  TicketUtils,
  DataUtils,
  FormatUtils,
  ValidationUtils,
};
