/**
 * 服务请求状态标签回归测试
 *
 * 关键回归点：列表/详情曾直接渲染后端原始值（如 "submitted"），
 * 中文界面出现英文状态。标签映射必须覆盖枚举全集，未知值原样返回不伪造。
 */

import {
  ServiceRequestStatus,
  ServiceRequestStatusLabels,
  serviceRequestStatusLabel,
} from '@/constants/service-request';

describe('serviceRequestStatusLabel', () => {
  it('maps every enum value to a Chinese label', () => {
    Object.values(ServiceRequestStatus).forEach((status) => {
      const label = serviceRequestStatusLabel(status);
      expect(label).not.toBe(status);
      expect(ServiceRequestStatusLabels[status]).toBe(label);
      expect(label).toMatch(/[一-龥A-Za-z]/);
    });
  });

  it('renders submitted as 已提交', () => {
    expect(serviceRequestStatusLabel('submitted')).toBe('已提交');
    expect(serviceRequestStatusLabel('manager_approved')).toBe('经理已审批');
  });

  it('returns unknown status verbatim instead of hiding it', () => {
    expect(serviceRequestStatusLabel('legacy_state')).toBe('legacy_state');
  });
});
