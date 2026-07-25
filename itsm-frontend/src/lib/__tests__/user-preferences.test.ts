/**
 * Tests for user-preferences.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

describe('UserPreferencesManager', () => {
  let userPreferences: any;

  beforeEach(() => {
    localStorage.clear();
    jest.resetModules();
  });

  const loadModule = async () => {
    const mod = await import('../user-preferences');
    return mod.userPreferences;
  };

  it('should load default preferences', async () => {
    userPreferences = await loadModule();
    const prefs = userPreferences.get();
    expect(prefs.theme).toBe('light');
    expect(prefs.language).toBe('zh-CN');
    expect(prefs.pageSize).toBe(20);
    expect(prefs.sidebarCollapsed).toBe(false);
  });

  it('should load saved preferences from localStorage', async () => {
    localStorage.setItem('user_preferences', JSON.stringify({ theme: 'dark', pageSize: 50 }));
    userPreferences = await loadModule();
    const prefs = userPreferences.get();
    expect(prefs.theme).toBe('dark');
    expect(prefs.pageSize).toBe(50);
  });

  it('should handle invalid JSON in localStorage', async () => {
    localStorage.setItem('user_preferences', 'invalid json');
    userPreferences = await loadModule();
    const prefs = userPreferences.get();
    expect(prefs.theme).toBe('light'); // defaults
  });

  it('should update preferences', async () => {
    userPreferences = await loadModule();
    userPreferences.update({ theme: 'dark', pageSize: 50 });
    const prefs = userPreferences.get();
    expect(prefs.theme).toBe('dark');
    expect(prefs.pageSize).toBe(50);
  });

  it('should persist to localStorage on update', async () => {
    userPreferences = await loadModule();
    userPreferences.update({ theme: 'dark' });
    const stored = JSON.parse(localStorage.getItem('user_preferences')!);
    expect(stored.theme).toBe('dark');
  });

  it('should update table settings', async () => {
    userPreferences = await loadModule();
    userPreferences.updateTableSettings('tickets', { columns: ['id', 'title'], sortBy: 'id', sortOrder: 'asc' });
    const prefs = userPreferences.get();
    expect(prefs.tableSettings.tickets).toEqual({ columns: ['id', 'title'], sortBy: 'id', sortOrder: 'asc' });
  });

  it('should subscribe and notify listeners', async () => {
    userPreferences = await loadModule();
    const listener = jest.fn();
    const unsubscribe = userPreferences.subscribe(listener);
    
    userPreferences.update({ theme: 'dark' });
    expect(listener).toHaveBeenCalled();
    
    unsubscribe();
    userPreferences.update({ theme: 'light' });
    expect(listener).toHaveBeenCalledTimes(1);
  });

  it('should reset to defaults', async () => {
    userPreferences = await loadModule();
    userPreferences.update({ theme: 'dark', pageSize: 100 });
    userPreferences.reset();
    const prefs = userPreferences.get();
    expect(prefs.theme).toBe('light');
    expect(prefs.pageSize).toBe(20);
  });

  it('should return a copy from get()', async () => {
    userPreferences = await loadModule();
    const prefs1 = userPreferences.get();
    const prefs2 = userPreferences.get();
    expect(prefs1).not.toBe(prefs2);
    expect(prefs1).toEqual(prefs2);
  });
});
