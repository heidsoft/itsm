import { buildMenuTree, collectMenuDescendantIds } from '../menuTreeUtils';

const menus = [
  { id: 1, parentId: null, sortOrder: 2 },
  { id: 2, parentId: null, sortOrder: 1 },
  { id: 3, parentId: 1, sortOrder: 2 },
  { id: 4, parentId: 1, sortOrder: 1 },
  { id: 5, parentId: 3, sortOrder: 1 },
  { id: 6, parentId: 999, sortOrder: 5 }, // 父级缺失 → 提升为根
];

describe('buildMenuTree', () => {
  it('按 sortOrder 组装树并将孤儿节点提升为根', () => {
    const tree = buildMenuTree(menus);
    expect(tree.map(n => n.id)).toEqual([2, 1, 6]);

    const node1 = tree.find(n => n.id === 1)!;
    expect(node1.children!.map(c => c.id)).toEqual([4, 3]);

    const node3 = node1.children!.find(c => c.id === 3)!;
    expect(node3.children!.map(c => c.id)).toEqual([5]);
  });

  it('叶子节点不带空 children 数组（避免表格显示展开按钮）', () => {
    const tree = buildMenuTree(menus);
    const node2 = tree.find(n => n.id === 2)!;
    expect(node2.children).toBeUndefined();
  });

  it('空列表返回空树', () => {
    expect(buildMenuTree([])).toEqual([]);
  });
});

describe('collectMenuDescendantIds', () => {
  it('收集自身与全部多级后代 id（用于父级下拉排除）', () => {
    const ids = collectMenuDescendantIds(menus, 1);
    expect(ids).toEqual(new Set([1, 3, 4, 5]));
  });

  it('叶子节点只包含自身', () => {
    expect(collectMenuDescendantIds(menus, 5)).toEqual(new Set([5]));
  });

  it('脏数据成环时不会死循环', () => {
    const cyclic = [
      { id: 1, parentId: 2, sortOrder: 1 },
      { id: 2, parentId: 1, sortOrder: 2 },
    ];
    expect(collectMenuDescendantIds(cyclic, 1)).toEqual(new Set([1, 2]));
  });
});
