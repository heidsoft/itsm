import React from 'react';
import { render, screen } from '@/lib/test-utils';
import { BusinessPageTemplate } from '../BusinessPageTemplate';

describe('BusinessPageTemplate statistics', () => {
  it('renders each statistic as a full grid item with an optional unit', () => {
    render(
      <BusinessPageTemplate
        title="事件管理"
        showViewSwitch={false}
        stats={[
          { label: '总事件', value: 1, icon: <span>📊</span> },
          { label: '待处理', value: 1, icon: <span>⏳</span> },
          { label: '紧急事件', value: 0, icon: <span>🚨</span> },
          { label: '平均解决时间', value: 0, suffix: '分钟', icon: <span>⏱️</span> },
        ]}
      >
        <div>事件列表</div>
      </BusinessPageTemplate>
    );

    const statsGrid = screen.getByRole('group', { name: '页面统计' });
    expect(statsGrid.children).toHaveLength(4);
    expect(statsGrid.querySelector('.ant-col')).not.toBeInTheDocument();
    expect(screen.getByLabelText('平均解决时间统计')).toHaveTextContent('0分钟');
  });
});
