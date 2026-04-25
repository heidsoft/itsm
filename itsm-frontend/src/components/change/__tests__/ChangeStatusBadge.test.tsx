/**
 * ChangeStatusBadge Component Tests
 * 测试变更状态徽章组件的渲染
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { ChangeStatusBadge } from '../ChangeStatusBadge';
import { ChangeStatus } from '@/constants/taxonomy';

describe('ChangeStatusBadge', () => {
  describe('状态渲染测试', () => {
    it('renders draft status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.DRAFT} />);
      expect(screen.getByText('草稿')).toBeInTheDocument();
    });

    it('renders pending status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.PENDING} />);
      expect(screen.getByText('待审批')).toBeInTheDocument();
    });

    it('renders approved status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.APPROVED} />);
      expect(screen.getByText('已批准')).toBeInTheDocument();
    });

    it('renders rejected status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.REJECTED} />);
      expect(screen.getByText('已拒绝')).toBeInTheDocument();
    });

    it('renders in_progress status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.IN_PROGRESS} />);
      expect(screen.getByText('实施中')).toBeInTheDocument();
    });

    it('renders completed status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.COMPLETED} />);
      expect(screen.getByText('已完成')).toBeInTheDocument();
    });

    it('renders rolled_back status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.ROLLED_BACK} />);
      expect(screen.getByText('已回滚')).toBeInTheDocument();
    });

    it('renders cancelled status correctly', () => {
      render(<ChangeStatusBadge status={ChangeStatus.CANCELLED} />);
      expect(screen.getByText('已取消')).toBeInTheDocument();
    });
  });

  describe('showText prop tests', () => {
    it('renders text by default', () => {
      render(<ChangeStatusBadge status={ChangeStatus.DRAFT} />);
      expect(screen.getByText('草稿')).toBeInTheDocument();
    });

    it('hides text when showText is false', () => {
      render(<ChangeStatusBadge status={ChangeStatus.DRAFT} showText={false} />);
      expect(screen.queryByText('草稿')).not.toBeInTheDocument();
    });

    it('shows text when showText is true', () => {
      render(<ChangeStatusBadge status={ChangeStatus.PENDING} showText={true} />);
      expect(screen.getByText('待审批')).toBeInTheDocument();
    });
  });

  describe('unknown status handling', () => {
    it('renders unknown status with the status value as label', () => {
      render(<ChangeStatusBadge status="unknown_status" />);
      expect(screen.getByText('unknown_status')).toBeInTheDocument();
    });

    it('renders empty string status with default fallback', () => {
      const { container } = render(<ChangeStatusBadge status="" />);
      // Should render a Tag component even for empty status
      const tag = container.querySelector('.ant-tag');
      expect(tag).toBeInTheDocument();
    });
  });

  describe('color mapping tests', () => {
    it('applies correct color for draft status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.DRAFT} />);
      const tag = screen.getByText('草稿').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-default');
    });

    it('applies correct color for pending status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.PENDING} />);
      const tag = screen.getByText('待审批').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-gold');
    });

    it('applies correct color for approved status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.APPROVED} />);
      const tag = screen.getByText('已批准').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-blue');
    });

    it('applies correct color for rejected status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.REJECTED} />);
      const tag = screen.getByText('已拒绝').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-red');
    });

    it('applies correct color for in_progress status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.IN_PROGRESS} />);
      const tag = screen.getByText('实施中').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-cyan');
    });

    it('applies correct color for completed status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.COMPLETED} />);
      const tag = screen.getByText('已完成').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-green');
    });

    it('applies correct color for rolled_back status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.ROLLED_BACK} />);
      const tag = screen.getByText('已回滚').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-orange');
    });

    it('applies correct color for cancelled status', () => {
      render(<ChangeStatusBadge status={ChangeStatus.CANCELLED} />);
      const tag = screen.getByText('已取消').closest('.ant-tag');
      expect(tag).toHaveClass('ant-tag-default');
    });
  });
});
