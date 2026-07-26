/**
 * 工单分类树工具函数
 */

export interface CategoryTreeInput {
  id: number;
  name: string;
  parentId: number | null;
  sortOrder: number;
  children?: CategoryTreeInput[];
}

/** 将扁平分类列表组装为任意层级的树 */
export function buildCategoryTree<T extends CategoryTreeInput>(list: T[]): T[] {
  const map = new Map<number, T>();
  list.forEach(item => map.set(item.id, { ...item, children: [] }));
  const roots: T[] = [];
  map.forEach(item => {
    if (item.parentId && map.has(item.parentId)) {
      (map.get(item.parentId)!.children as T[]).push(item);
    } else {
      roots.push(item);
    }
  });
  const sortRec = (nodes: T[]) => {
    nodes.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name));
    nodes.forEach(n => {
      if (n.children && n.children.length > 0) sortRec(n.children as T[]);
      else delete n.children;
    });
  };
  sortRec(roots);
  return roots;
}

/** 收集某分类的全部后代 id（含自身），用于父级选择时禁用，防止形成环 */
export function collectDescendantIds(
  list: { id: number; parentId: number | null }[],
  rootId: number
): Set<number> {
  const childrenMap = new Map<number, number[]>();
  list.forEach(item => {
    if (item.parentId) {
      const arr = childrenMap.get(item.parentId) || [];
      arr.push(item.id);
      childrenMap.set(item.parentId, arr);
    }
  });
  const result = new Set<number>([rootId]);
  const stack = [rootId];
  while (stack.length > 0) {
    const current = stack.pop()!;
    (childrenMap.get(current) || []).forEach(childId => {
      if (!result.has(childId)) {
        result.add(childId);
        stack.push(childId);
      }
    });
  }
  return result;
}
