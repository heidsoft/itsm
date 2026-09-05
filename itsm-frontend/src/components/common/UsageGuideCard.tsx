'use client';

import React, { useState } from 'react';
import { Collapse, Typography, theme } from 'antd';
import { BookOpenCheck } from 'lucide-react';

const { Text } = Typography;

export interface UsageGuideCardProps {
  /** 引导标题，默认“如何使用本页面” */
  title?: string;
  /** 一句话说明本页面的定位/与其他模块的关系 */
  intro?: React.ReactNode;
  /** 按顺序排列的操作步骤，每条一个字符串或节点 */
  steps: React.ReactNode[];
  /** 默认是否展开 */
  defaultOpen?: boolean;
  /** 外层样式（如 margin） */
  style?: React.CSSProperties;
}

/**
 * 轻量使用引导卡片：用于管理/配置类页面顶部，回答“这个页面是干什么的、
 * 按什么顺序操作、和哪些模块联动”。默认折叠，不挤占操作空间。
 */
export const UsageGuideCard: React.FC<UsageGuideCardProps> = ({
  title = '如何使用本页面',
  intro,
  steps,
  defaultOpen = false,
  style,
}) => {
  const { token } = theme.useToken();
  const [open, setOpen] = useState(defaultOpen);

  return (
    <Collapse
      activeKey={open ? ['guide'] : []}
      onChange={keys => setOpen(keys.includes('guide'))}
      items={[
        {
          key: 'guide',
          label: (
            <span
              style={{ display: 'inline-flex', alignItems: 'center', gap: 8, fontWeight: 500 }}
            >
              <BookOpenCheck size={16} />
              {title}
            </span>
          ),
          children: (
            <div>
              {intro && (
                <Text style={{ display: 'block', marginBottom: token.marginSM }}>{intro}</Text>
              )}
              <ol
                style={{
                  margin: 0,
                  paddingLeft: 20,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: token.marginXS,
                }}
              >
                {steps.map((step, i) => (
                  <li key={i} style={{ lineHeight: 1.7 }}>
                    {step}
                  </li>
                ))}
              </ol>
            </div>
          ),
        },
      ]}
      style={{ background: token.colorInfoBg, borderColor: token.colorInfoBorder, ...style }}
    />
  );
};

export default UsageGuideCard;
