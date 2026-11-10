import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);

// 接受 Date 实例是为了让「本地已持有 Date 对象」的页面不必先 toISOString()
// 再解析回来（Invalid Date 上调用 toISOString 会抛 RangeError）。
export const formatDateTime = (value?: string | Date | null) => {
  if (!value) return '';
  const d = dayjs(value);
  if (!d.isValid() || d.year() <= 1) return '';
  // 统一使用 YYYY-MM-DD HH:mm，避免 toLocaleString 输出 "2026/8/15 14:30:00" 这类不一致格式
  return d.format('YYYY-MM-DD HH:mm');
};

export const mapLabel = (map: Record<string, string>, value?: string) => {
  if (!value) return '';
  return map[value] || value;
};

// 姓名由服务端投影；缺投影时如实显示用户 ID，不能渲染成「未分配」
export const displayUserName = (
  name: string | undefined,
  userId: number | undefined,
  emptyLabel: string
): string => {
  const trimmed = name?.trim();
  if (trimmed) return trimmed;
  return userId ? `User #${userId}` : emptyLabel;
};
