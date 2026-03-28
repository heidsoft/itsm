/**
 * Security 工具测试
 */

import {
  xssProtection,
  inputSanitization,
  passwordSecurity,
  sessionSecurity,
  contentSecurity,
  networkSecurity,
  encryption,
  securityConfig,
} from '../security';

describe('xssProtection', () => {
  describe('escapeHtml', () => {
    it('should escape HTML special characters', () => {
      expect(xssProtection.escapeHtml('<script>alert("xss")</script>')).toBe(
        '&lt;script&gt;alert(&quot;xss&quot;)&lt;&#x2F;script&gt;'
      );
    });

    it('should escape ampersands', () => {
      expect(xssProtection.escapeHtml('foo & bar')).toBe('foo &amp; bar');
    });

    it('should escape single quotes', () => {
      expect(xssProtection.escapeHtml("it's a test")).toBe('it&#39;s a test');
    });

    it('should escape forward slashes', () => {
      expect(xssProtection.escapeHtml('path/to/file')).toBe('path&#x2F;to&#x2F;file');
    });

    it('should return unchanged string without special characters', () => {
      expect(xssProtection.escapeHtml('plain text')).toBe('plain text');
    });
  });

  describe('unescapeHtml', () => {
    it('should unescape HTML entities', () => {
      expect(xssProtection.unescapeHtml('&lt;script&gt;')).toBe('<script>');
    });

    it('should unescape ampersands', () => {
      expect(xssProtection.unescapeHtml('foo &amp; bar')).toBe('foo & bar');
    });

    it('should unescape quotes', () => {
      expect(xssProtection.unescapeHtml('&quot;quoted&quot;')).toBe('"quoted"');
    });

    it('should unescape hex entities', () => {
      expect(xssProtection.unescapeHtml('&#x2F;')).toBe('/');
    });
  });

  describe('stripHtml', () => {
    it('should remove all HTML tags', () => {
      expect(xssProtection.stripHtml('<p>Hello <strong>World</strong></p>')).toBe('Hello World');
    });

    it('should return original string if no tags', () => {
      expect(xssProtection.stripHtml('plain text')).toBe('plain text');
    });

    it('should handle nested tags', () => {
      expect(xssProtection.stripHtml('<div><span><em>text</em></span></div>')).toBe('text');
    });
  });
});

describe('inputSanitization', () => {
  describe('sanitizeInput', () => {
    it('should trim whitespace', () => {
      expect(inputSanitization.sanitizeInput('  hello  ')).toBe('hello');
    });

    it('should remove potentially dangerous characters', () => {
      expect(inputSanitization.sanitizeInput('foo"bar')).toBe('foobar');
    });

    it('should limit length to 1000 characters', () => {
      const longString = 'a'.repeat(1500);
      expect(inputSanitization.sanitizeInput(longString).length).toBe(1000);
    });
  });

  describe('isValidEmail', () => {
    it('should validate correct email formats', () => {
      expect(inputSanitization.isValidEmail('user@example.com')).toBe(true);
      expect(inputSanitization.isValidEmail('user.name@example.co.uk')).toBe(true);
    });

    it('should reject invalid email formats', () => {
      expect(inputSanitization.isValidEmail('invalid')).toBe(false);
      expect(inputSanitization.isValidEmail('invalid@')).toBe(false);
      expect(inputSanitization.isValidEmail('@example.com')).toBe(false);
    });

    it('should reject emails over 254 characters', () => {
      const longEmail = 'a'.repeat(250) + '@example.com';
      expect(inputSanitization.isValidEmail(longEmail)).toBe(false);
    });
  });

  describe('isValidUrl', () => {
    it('should validate correct HTTP/HTTPS URLs', () => {
      expect(inputSanitization.isValidUrl('https://example.com')).toBe(true);
      expect(inputSanitization.isValidUrl('http://example.com/path')).toBe(true);
    });

    it('should reject invalid URLs', () => {
      expect(inputSanitization.isValidUrl('not-a-url')).toBe(false);
      expect(inputSanitization.isValidUrl('ftp://example.com')).toBe(false);
    });
  });

  describe('isValidFileType', () => {
    it('should validate allowed file extensions', () => {
      expect(inputSanitization.isValidFileType('document.pdf', ['pdf', 'doc'])).toBe(true);
      expect(inputSanitization.isValidFileType('image.PNG', ['png', 'jpg'])).toBe(true);
    });

    it('should reject disallowed file extensions', () => {
      expect(inputSanitization.isValidFileType('script.exe', ['pdf', 'doc'])).toBe(false);
    });

    it('should handle files without extension', () => {
      expect(inputSanitization.isValidFileType('noextension', ['pdf'])).toBe(false);
    });
  });

  describe('isValidFileSize', () => {
    it('should validate file sizes under limit', () => {
      expect(inputSanitization.isValidFileSize(5 * 1024 * 1024, 10)).toBe(true); // 5MB, 10MB limit
    });

    it('should reject file sizes over limit', () => {
      expect(inputSanitization.isValidFileSize(15 * 1024 * 1024, 10)).toBe(false); // 15MB, 10MB limit
    });
  });
});

describe('passwordSecurity', () => {
  describe('checkPasswordStrength', () => {
    it('should score weak passwords low', () => {
      const result = passwordSecurity.checkPasswordStrength('abc');
      expect(result.score).toBeLessThan(3);
      expect(result.isStrong).toBe(false);
    });

    it('should score strong passwords high', () => {
      const result = passwordSecurity.checkPasswordStrength('Str0ng!Pass');
      expect(result.score).toBeGreaterThanOrEqual(4);
      expect(result.isStrong).toBe(true);
    });

    it('should reject common passwords', () => {
      const result = passwordSecurity.checkPasswordStrength('password');
      expect(result.score).toBe(0);
      expect(result.feedback).toContain('请不要使用常见的弱密码');
    });

    it('should require minimum 8 characters', () => {
      const result = passwordSecurity.checkPasswordStrength('Ab1!xyz');
      expect(result.feedback).toContain('密码长度至少需要8个字符');
    });
  });

  describe('generateRandomPassword', () => {
    it('should generate password of specified length', () => {
      expect(passwordSecurity.generateRandomPassword(16).length).toBe(16);
    });

    it('should generate different passwords each time', () => {
      const pass1 = passwordSecurity.generateRandomPassword(12);
      const pass2 = passwordSecurity.generateRandomPassword(12);
      expect(pass1).not.toBe(pass2);
    });
  });
});

describe('sessionSecurity', () => {
  describe('isSessionExpired', () => {
    it('should return true for expired sessions', () => {
      const expiredTime = Date.now() - 1000; // 1 second ago
      expect(sessionSecurity.isSessionExpired(expiredTime)).toBe(true);
    });

    it('should return false for valid sessions', () => {
      const validTime = Date.now() + 60000; // 1 minute from now
      expect(sessionSecurity.isSessionExpired(validTime)).toBe(false);
    });
  });

  describe('generateSessionId', () => {
    it('should generate 32-character hex string', () => {
      const sessionId = sessionSecurity.generateSessionId();
      expect(sessionId).toMatch(/^[a-f0-9]{32}$/);
    });

    it('should generate unique session IDs', () => {
      const id1 = sessionSecurity.generateSessionId();
      const id2 = sessionSecurity.generateSessionId();
      expect(id1).not.toBe(id2);
    });
  });
});

describe('contentSecurity', () => {
  describe('validateContentType', () => {
    it('should validate allowed content types', () => {
      expect(contentSecurity.validateContentType('application/json', ['json', 'xml'])).toBe(true);
    });

    it('should reject disallowed content types', () => {
      expect(contentSecurity.validateContentType('text/html', ['json', 'xml'])).toBe(false);
    });
  });

  describe('containsMaliciousScript', () => {
    it('should detect script tags', () => {
      expect(contentSecurity.containsMaliciousScript('<script>alert(1)</script>')).toBe(true);
    });

    it('should detect javascript: URLs', () => {
      expect(contentSecurity.containsMaliciousScript('javascript:alert(1)')).toBe(true);
    });

    it('should detect event handlers', () => {
      expect(contentSecurity.containsMaliciousScript('<img onerror="alert(1)">')).toBe(true);
    });

    it('should return false for safe content', () => {
      expect(contentSecurity.containsMaliciousScript('plain text content')).toBe(false);
    });
  });

  describe('sanitizeContent', () => {
    it('should remove script tags', () => {
      const result = contentSecurity.sanitizeContent('<p>Hello</p><script>alert(1)</script>');
      expect(result).not.toContain('<script>');
    });

    it('should remove javascript: URLs', () => {
      const result = contentSecurity.sanitizeContent('Click <a href="javascript:alert(1)">here</a>');
      expect(result).not.toContain('javascript:');
    });
  });
});

describe('networkSecurity', () => {
  describe('isHttps', () => {
    it('should detect HTTPS protocol', () => {
      // Mock window.location.protocol
      Object.defineProperty(window, 'location', {
        value: { protocol: 'https:' },
        writable: true,
      });
      expect(networkSecurity.isHttps()).toBe(true);
    });

    it('should detect HTTP protocol', () => {
      Object.defineProperty(window, 'location', {
        value: { protocol: 'http:' },
        writable: true,
      });
      expect(networkSecurity.isHttps()).toBe(false);
    });
  });

  describe('validateOrigin', () => {
    it('should validate allowed origins', () => {
      expect(networkSecurity.validateOrigin('https://example.com', ['https://example.com'])).toBe(true);
    });

    it('should reject disallowed origins', () => {
      expect(networkSecurity.validateOrigin('https://evil.com', ['https://example.com'])).toBe(false);
    });
  });

  describe('getSecureHeaders', () => {
    it('should return secure headers', () => {
      const headers = networkSecurity.getSecureHeaders('csrf-token');
      expect(headers['Content-Type']).toBe('application/json');
      expect(headers['X-Requested-With']).toBe('XMLHttpRequest');
      expect(headers['X-CSRF-Token']).toBe('csrf-token');
    });

    it('should omit CSRF token if not provided', () => {
      const headers = networkSecurity.getSecureHeaders();
      expect(headers['X-CSRF-Token']).toBeUndefined();
    });
  });
});

describe('encryption', () => {
  describe('base64Encode', () => {
    it('should encode text to base64', () => {
      expect(encryption.base64Encode('Hello')).toBe('SGVsbG8=');
    });

    it('should handle unicode characters', () => {
      expect(encryption.base64Encode('你好')).toBe('5L2g5aW9');
    });
  });

  describe('base64Decode', () => {
    it('should decode base64 to text', () => {
      expect(encryption.base64Decode('SGVsbG8=')).toBe('Hello');
    });

    it('should handle unicode characters', () => {
      expect(encryption.base64Decode('5L2g5aW9')).toBe('你好');
    });
  });

  describe('obfuscate and deobfuscate', () => {
    it('should obfuscate and deobfuscate text', () => {
      const original = 'secret message';
      const key = 'encryption-key';
      const obfuscated = encryption.obfuscate(original, key);
      expect(obfuscated).not.toBe(original);
      expect(encryption.deobfuscate(obfuscated, key)).toBe(original);
    });

    it('should produce different obfuscations with different keys', () => {
      const original = 'secret message';
      const obfuscated1 = encryption.obfuscate(original, 'key1');
      const obfuscated2 = encryption.obfuscate(original, 'key2');
      expect(obfuscated1).not.toBe(obfuscated2);
    });
  });
});

describe('securityConfig', () => {
  it('should have default values', () => {
    expect(securityConfig.defaults.maxFileSize).toBe(10);
    expect(securityConfig.defaults.passwordMinLength).toBe(8);
    expect(securityConfig.defaults.sessionTimeout).toBe(30);
  });

  it('should get and set configuration values', () => {
    const originalValue = securityConfig.get('maxFileSize');
    securityConfig.set('maxFileSize', 20);
    expect(securityConfig.get('maxFileSize')).toBe(20);
    // Restore original
    securityConfig.set('maxFileSize', originalValue as number);
  });
});
