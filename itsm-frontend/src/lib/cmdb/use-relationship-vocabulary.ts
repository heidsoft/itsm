/**
 * CMDB 关系词表运行时注入 hook（P1-4）
 *
 * 单一源：后端 /api/v1/cmdb/ontology（CMDBOntologyResponse.relationshipTypes）。
 * 任何挂载了这个 hook 的页面，都会把后端权威词表写入
 * relationship-vocabulary.ts 的 runtimeVocabulary 模块级变量，
 * 之后所有 relationshipLabel()/relationshipMeta() 调用都会读取运行时版本。
 *
 * 后端不可达时 fail-soft（runtimeVocabulary 保持 null，函数回退到默认词表）。
 */

import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { CMDB_KEYS } from '@/lib/hooks/useCMDB';
import {
  loadRelationshipVocabulary,
  type RelationshipTypeMeta,
} from '@/lib/cmdb/relationship-vocabulary';

interface OntologyRelationshipType {
  type: string;
  name: string;
  description?: string;
  direction?: string;
  icon?: string;
  reverseType?: string;
}

interface OntologyResponse {
  relationshipTypes?: OntologyRelationshipType[];
}

/**
 * React hook：拉取 ontology 的 relationshipTypes 并写入 runtimeVocabulary。
 * 返回 vocab 数组（来自后端，远端失败时为 null）。
 */
export function useOntologyRelationshipVocabulary(): {
  vocabulary: RelationshipTypeMeta[] | null;
  isLoading: boolean;
  isError: boolean;
} {
  const query = useQuery({
    queryKey: [...CMDB_KEYS.all, 'ontology', 'vocabulary-loader'] as const,
    queryFn: () => CMDBApi.getOntology() as Promise<OntologyResponse>,
    staleTime: 600000,
  });

  useEffect(() => {
    if (!query.data?.relationshipTypes) return;
    const normalized: Array<{
      type: string;
      name: string;
      description?: string;
      direction?: string;
      reverse?: string;
      icon?: string;
    }> = query.data.relationshipTypes.map(item => ({
      type: item.type,
      name: item.name,
      description: item.description,
      direction: item.direction,
      reverse: item.reverseType,
      icon: item.icon,
    }));
    void loadRelationshipVocabulary(async () => normalized);
  }, [query.data]);

  return {
    vocabulary: query.data?.relationshipTypes
      ? query.data.relationshipTypes.map(item => ({
          type: item.type as RelationshipTypeMeta['type'],
          name: item.name,
          description: item.description ?? '',
          direction:
            (item.direction as RelationshipTypeMeta['direction']) ?? 'uni-directional',
          reverse: item.reverseType as RelationshipTypeMeta['reverse'],
          icon: item.icon ?? 'link',
        }))
      : null,
    isLoading: query.isLoading,
    isError: query.isError,
  };
}
