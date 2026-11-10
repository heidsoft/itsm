/**
 * CMDB 关系类型受控词表（前端单一源）
 *
 * 与后端 ent/schema/ci_relationship.go 的 CIRelationshipType 常量一一对应。
 * 后端是权威来源：运行时优先用 /api/v1/cmdb/relationship-types 的返回值覆盖
 * 下面的静态默认值（loadRelationshipVocabulary），保证前后端词表不会漂移。
 *
 * 注意：@/types/cmdb.ts 里的 RelationType 枚举是历史遗留的 mock 词表，
 * 仅被工单关系域（ticket-relations）使用，CMDB 侧不要引用它。
 */

import type { CIRelationshipType } from '@/lib/api/cmdb-relationship';

export interface RelationshipTypeMeta {
  /** 关系类型标识 */
  type: CIRelationshipType;
  /** 中文名 */
  name: string;
  /** 语义描述 */
  description: string;
  /** 方向：单向 / 双向 */
  direction: 'uni-directional' | 'bi-directional';
  /** 反向关系类型（无反向则为空） */
  reverse?: CIRelationshipType;
  /** 展示图标名 */
  icon: string;
}

/** 静态默认词表（后端不可达时的降级值） */
export const DEFAULT_RELATIONSHIP_VOCABULARY: RelationshipTypeMeta[] = [
  { type: 'depends_on', name: 'dependency', description: 'Source CI depends on target CI', direction: 'uni-directional', reverse: 'impacted_by', icon: 'link' },
  { type: 'hosts', name: 'Hosting', description: 'Source CI hosts target CI', direction: 'uni-directional', reverse: 'hosted_on', icon: 'server' },
  { type: 'hosted_on', name: 'Hosted by', description: 'Source CI is running or deployed on target CI', direction: 'uni-directional', reverse: 'hosts', icon: 'hard-drive' },
  { type: 'connects_to', name: 'Connect to', description: 'There is a network connection between the source CI and the target CI', direction: 'bi-directional', reverse: 'connects_to', icon: 'network' },
  { type: 'runs_on', name: 'Running on', description: 'Source CI is running on target CI', direction: 'uni-directional', icon: 'play' },
  { type: 'contains', name: 'Contains', description: 'Source CI contains target CI', direction: 'uni-directional', reverse: 'part_of', icon: 'box' },
  { type: 'part_of', name: 'Components', description: 'Source CI is part of target CI', direction: 'uni-directional', reverse: 'contains', icon: 'component' },
  { type: 'impacts', name: 'Impact', description: 'Source CI failure affects target CI', direction: 'uni-directional', reverse: 'impacted_by', icon: 'activity' },
  { type: 'impacted_by', name: 'Affected by', description: 'Source CI affected by target CI failure', direction: 'uni-directional', reverse: 'depends_on', icon: 'activity' },
  { type: 'owns', name: 'Owned', description: 'Source CI owns target CI', direction: 'uni-directional', reverse: 'owned_by', icon: 'key' },
  { type: 'owned_by', name: 'owned', description: 'Source CI is owned by target CI', direction: 'uni-directional', reverse: 'owns', icon: 'key' },
  { type: 'uses', name: 'use', description: 'Source CI uses target CI capabilities', direction: 'uni-directional', reverse: 'used_by', icon: 'plug' },
  { type: 'used_by', name: 'used', description: 'Source CI capability is used by target CI', direction: 'uni-directional', reverse: 'uses', icon: 'plug' },
];

let runtimeVocabulary: RelationshipTypeMeta[] | null = null;

/**
 * 从后端拉取关系词表并覆盖静态默认值。
 * 后端不可达或返回异常时保留静态默认值，不抛错（fail-soft）。
 */
export async function loadRelationshipVocabulary(
  fetcher?: () => Promise<
    Array<{ type: string; name: string; description?: string; direction?: string; reverse?: string; icon?: string }>
  >,
): Promise<RelationshipTypeMeta[]> {
  if (!fetcher) return runtimeVocabulary ?? DEFAULT_RELATIONSHIP_VOCABULARY;

  try {
    const remote = await fetcher();
    if (!Array.isArray(remote) || remote.length === 0) {
      return runtimeVocabulary ?? DEFAULT_RELATIONSHIP_VOCABULARY;
    }
    const byType = new Map(DEFAULT_RELATIONSHIP_VOCABULARY.map(m => [m.type, m]));
    const merged: RelationshipTypeMeta[] = remote.map(item => {
      const fallback = byType.get(item.type as CIRelationshipType);
      return {
        type: item.type as CIRelationshipType,
        name: item.name || fallback?.name || item.type,
        description: item.description || fallback?.description || '',
        direction: (item.direction as RelationshipTypeMeta['direction']) || fallback?.direction || 'uni-directional',
        reverse: (item.reverse as CIRelationshipType) || fallback?.reverse,
        icon: item.icon || fallback?.icon || 'link',
      };
    });
    runtimeVocabulary = merged;
    return merged;
  } catch {
    return runtimeVocabulary ?? DEFAULT_RELATIONSHIP_VOCABULARY;
  }
}

/** 取关系类型中文名；未收录时回退为原始 type，保证 UI 不出现空白 */
export function relationshipLabel(type: string): string {
  const vocab = runtimeVocabulary ?? DEFAULT_RELATIONSHIP_VOCABULARY;
  return vocab.find(m => m.type === type)?.name ?? type;
}

/** 取关系类型元数据 */
export function relationshipMeta(type: string): RelationshipTypeMeta | undefined {
  const vocab = runtimeVocabulary ?? DEFAULT_RELATIONSHIP_VOCABULARY;
  return vocab.find(m => m.type === type);
}
