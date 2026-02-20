interface UserPreferences {
  theme: 'light' | 'dark' | 'auto';
  language: 'zh-CN' | 'en-US';
  timezone: string;
  dateFormat: string;
  pageSize: number;
  sidebarCollapsed: boolean;
  dashboardLayout: string[];
  notifications: {
    email: boolean;
    browser: boolean;
    mobile: boolean;
    types: string[];
  };
  tableSettings: Record<string, {
    columns: string[];
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
  }>;
}

const defaultPreferences: UserPreferences = {
  theme: 'light',
  language: 'zh-CN',
  timezone: 'Asia/Shanghai',
  dateFormat: 'YYYY-MM-DD HH:mm:ss',
  pageSize: 20,
  sidebarCollapsed: false,
  dashboardLayout: ['kpi', 'charts', 'recent-tickets'],
  notifications: {
    email: true,
    browser: true,
    mobile: false,
    types: ['ticket', 'approval', 'alert']
  },
  tableSettings: {}
};

class UserPreferencesManager {
  private preferences: UserPreferences;
  private listeners: ((preferences: UserPreferences) => void)[] = [];

  constructor() {
    this.preferences = this.loadPreferences();
  }

  private loadPreferences(): UserPreferences {
    if (typeof window === 'undefined') return defaultPreferences;
    
    try {
      const saved = localStorage.getItem('user_preferences');
      return saved ? { ...defaultPreferences, ...JSON.parse(saved) } : defaultPreferences;
    } catch {
      return defaultPreferences;
    }
  }

  private savePreferences() {
    if (typeof window === 'undefined') return;
    localStorage.setItem('user_preferences', JSON.stringify(this.preferences));
    this.notifyListeners();
  }

  get(): UserPreferences {
    return { ...this.preferences };
  }

  update(updates: Partial<UserPreferences>) {
    this.preferences = { ...this.preferences, ...updates };
    this.savePreferences();
  }

  updateTableSettings(tableId: string, settings: UserPreferences['tableSettings'][string]) {
    this.preferences.tableSettings[tableId] = settings;
    this.savePreferences();
  }

  subscribe(listener: (preferences: UserPreferences) => void) {
    this.listeners.push(listener);
    return () => {
      this.listeners = this.listeners.filter(l => l !== listener);
    };
  }

  private notifyListeners() {
    this.listeners.forEach(listener => listener(this.preferences));
  }

  reset() {
    this.preferences = { ...defaultPreferences };
    this.savePreferences();
  }
}

export const userPreferences = new UserPreferencesManager();