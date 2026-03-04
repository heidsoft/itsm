/**
 * CI 拓扑图标签页组件
 * 包装现有的 TopologyGraph 组件
 */

import React from 'react';
import TopologyGraph from '../../TopologyGraph';

interface CITopologyGraphProps {
  ciId: number;
}

export const CITopologyGraph: React.FC<CITopologyGraphProps> = ({ ciId }) => {
  return <TopologyGraph ciId={ciId} initialDepth={2} height={500} />;
};

CITopologyGraph.displayName = 'CITopologyGraph';
