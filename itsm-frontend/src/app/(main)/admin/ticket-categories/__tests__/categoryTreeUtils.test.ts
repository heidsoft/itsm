import {
  buildCategoryTree,
  collectDescendantIds,
  type CategoryTreeInput,
} from '../categoryTreeUtils';

const flat: CategoryTreeInput[] = [
  { id: 1, name: '硬件', parentId: null, sortOrder: 1 },
  { id: 2, name: '软件', parentId: null, sortOrder: 2 },
  { id: 3, name: '服务器', parentId: 1, sortOrder: 1 },
  { id: 4, name: '网络设备', parentId: 1, sortOrder: 2 },
  { id: 5, name: '机架服务器', parentId: 3, sortOrder: 1 },
];

describe('buildCategoryTree', () => {
  it('builds an arbitrary-depth tree from a flat list', () => {
    const tree = buildCategoryTree(flat);
    expect(tree).toHaveLength(2);
    const hardware = tree.find(n => n.id === 1)!;
    expect(hardware.children).toHaveLength(2);
    const server = hardware.children!.find(n => n.id === 3)!;
    expect(server.children).toHaveLength(1);
    expect(server.children![0].id).toBe(5);
  });

  it('sorts siblings by sortOrder', () => {
    const tree = buildCategoryTree(flat);
    expect(tree.map(n => n.id)).toEqual([1, 2]);
    const hardware = tree.find(n => n.id === 1)!;
    expect(hardware.children!.map(n => n.id)).toEqual([3, 4]);
  });

  it('treats orphan parentId as root', () => {
    const orphans: CategoryTreeInput[] = [{ id: 9, name: '孤儿', parentId: 999, sortOrder: 1 }];
    const tree = buildCategoryTree(orphans);
    expect(tree).toHaveLength(1);
    expect(tree[0].id).toBe(9);
  });

  it('omits children key for leaf nodes', () => {
    const tree = buildCategoryTree(flat);
    const software = tree.find(n => n.id === 2)!;
    expect(software.children).toBeUndefined();
  });
});

describe('collectDescendantIds', () => {
  it('collects self and all descendants', () => {
    const ids = collectDescendantIds(flat, 1);
    expect(ids).toEqual(new Set([1, 3, 4, 5]));
  });

  it('returns only self for leaf node', () => {
    const ids = collectDescendantIds(flat, 5);
    expect(ids).toEqual(new Set([5]));
  });
});
