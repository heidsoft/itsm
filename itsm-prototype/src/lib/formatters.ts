export const formatDateTime = (iso?: string) => {
  if (!iso) return '';
  try {
    return new Date(iso).toLocaleString('zh-CN');
  } catch {
    return iso || '';
  }
};

export const mapLabel = (map: Record<string, string>, value?: string) => {
  if (!value) return '';
  return map[value] || value;
};
