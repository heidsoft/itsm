// 菜单树工具：树形表格组装与父级候选排除
// 独立于页面组件，便于复用与单元测试

export interface MenuTreeInput {
  id: number;
  parentId?: number | null;
  sortOrder: number;
}

export type MenuTreeNode<T extends MenuTreeInput> = T & { children?: MenuTreeNode<T>[] };

/**
 * 将平铺菜单列表组装为树（按 sortOrder 升序），父级缺失的节点提升为根节点
 */
export function buildMenuTree<T extends MenuTreeInput>(list: T[]): MenuTreeNode<T>[] {
  const nodeMap = new Map<number, MenuTreeNode<T>>();
  list.forEach(item => {
    nodeMap.set(item.id, { ...item, children: [] });
  });

  const roots: MenuTreeNode<T>[] = [];
  nodeMap.forEach(node => {
    const parent = node.parentId ? nodeMap.get(node.parentId) : undefined;
    if (parent && parent.id !== node.id) {
      parent.children!.push(node);
    } else {
      roots.push(node);
    }
  });

  const sortNodes = (nodes: MenuTreeNode<T>[]) => {
    nodes.sort((a, b) => a.sortOrder - b.sortOrder);
    nodes.forEach(node => {
      if (node.children && node.children.length > 0) {
        sortNodes(node.children);
      } else {
        delete node.children;
      }
    });
  };
  sortNodes(roots);
  return roots;
}

/**
 * 收集指定菜单及其全部后代的 id 集合（含自身），带访问集防止脏数据成环
 */
export function collectMenuDescendantIds(list: MenuTreeInput[], rootId: number): Set<number> {
  const childrenMap = new Map<number, number[]>();
  list.forEach(item => {
    if (item.parentId) {
      const siblings = childrenMap.get(item.parentId) || [];
      siblings.push(item.id);
      childrenMap.set(item.parentId, siblings);
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
