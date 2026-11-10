describe('English language preference migration', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.resetModules();
  });

  it('uses English for new users', () => {
    const { userPreferences } = require('../user-preferences');
    expect(userPreferences.get().language).toBe('en-US');
  });

  it('normalizes a saved Chinese language while preserving other preferences', () => {
    localStorage.setItem('user_preferences', JSON.stringify({ language: 'zh-CN', pageSize: 50 }));
    const { userPreferences } = require('../user-preferences');
    expect(userPreferences.get()).toMatchObject({ language: 'en-US', pageSize: 50 });
    userPreferences.update({ theme: 'dark' });
    expect(JSON.parse(localStorage.getItem('user_preferences') || '{}').language).toBe('en-US');
    userPreferences.reset();
    expect(userPreferences.get().language).toBe('en-US');
  });
});
