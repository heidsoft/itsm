// Security utilities for HTTP client
export const security = {
  csrf: {
    getTokenFromMeta(): string | null {
      if (typeof window === 'undefined') return null;
      const meta = document.querySelector('meta[name="csrf-token"]');
      return meta ? meta.getAttribute('content') : null;
    }
  },
  
  network: {
    getSecureHeaders(csrfToken?: string): Record<string, string> {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      };
      
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }
      
      return headers;
    }
  }
};
