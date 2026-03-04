/**
 * CI 关系管理标签页组件
 * 包装现有的 CIRelationshipManager 组件
 */

import React from 'react';
import CIRelationshipManager from '../../CIRelationshipManager';

interface CIRelationshipsTabProps {
  ciId: number;
  ciName: string;
  onRefresh?: () => void;
}

export const CIRelationshipsTab: React.FC<CIRelationshipsTabProps> = ({
  ciId,
  ciName,
  onRefresh,
}) => {
  return <CIRelationshipManager ciId={ciId} ciName={ciName} onRefresh={onRefresh} />;
};

CIRelationshipsTab.displayName = 'CIRelationshipsTab';
