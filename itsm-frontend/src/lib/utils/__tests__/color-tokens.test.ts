import {
  statusColors,
  priorityColors,
  getStatusColor,
  getPriorityColor,
} from '../color-tokens';

describe('getStatusColor', () => {
  it('returns correct colors for known status "pending"', () => {
    const result = getStatusColor('pending');
    expect(result.color).toBe(statusColors.pending.color);
    expect(result.backgroundColor).toBe(statusColors.pending.backgroundColor);
    expect(result.borderColor).toBe(statusColors.pending.borderColor);
    expect(result.text).toBe('待审批');
  });

  it('returns correct colors for "completed"', () => {
    const result = getStatusColor('completed');
    expect(result.color).toBe(statusColors.completed.color);
    expect(result.text).toBe('已完成');
  });

  it('falls back to draft colors for unknown status', () => {
    const result = getStatusColor('unknown_status');
    expect(result.color).toBe(statusColors.draft.color);
    expect(result.backgroundColor).toBe(statusColors.draft.backgroundColor);
    expect(result.text).toBe('unknown_status');
  });

  it('returns label text for "cancelled"', () => {
    expect(getStatusColor('cancelled').text).toBe('已取消');
  });
});

describe('getPriorityColor', () => {
  it('returns correct colors for "urgent"', () => {
    const result = getPriorityColor('urgent');
    expect(result.color).toBe(priorityColors.urgent.color);
    expect(result.backgroundColor).toBe(priorityColors.urgent.backgroundColor);
    expect(result.text).toBe('紧急');
  });

  it('returns correct colors for "low"', () => {
    const result = getPriorityColor('low');
    expect(result.color).toBe(priorityColors.low.color);
    expect(result.text).toBe('低');
  });

  it('falls back to medium colors for unknown priority', () => {
    const result = getPriorityColor('unknown_priority');
    expect(result.color).toBe(priorityColors.medium.color);
    expect(result.backgroundColor).toBe(priorityColors.medium.backgroundColor);
    expect(result.text).toBe('unknown_priority');
  });
});

describe('statusColors map', () => {
  it('has expected keys', () => {
    expect(Object.keys(statusColors)).toEqual(
      expect.arrayContaining(['pending', 'approved', 'implementing', 'completed', 'cancelled', 'draft', 'rejected'])
    );
  });

  it('each entry has color, backgroundColor, borderColor', () => {
    for (const key of Object.keys(statusColors)) {
      expect(statusColors[key]).toHaveProperty('color');
      expect(statusColors[key]).toHaveProperty('backgroundColor');
      expect(statusColors[key]).toHaveProperty('borderColor');
    }
  });
});

describe('priorityColors map', () => {
  it('has expected keys', () => {
    expect(Object.keys(priorityColors)).toEqual(
      expect.arrayContaining(['urgent', 'high', 'medium', 'low'])
    );
  });
});
