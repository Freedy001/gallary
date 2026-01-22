/**
 * CNB Browser Automation - Browser Manager Tests
 * 
 * Unit tests for BrowserManager class.
 * Requirements tested: 1.1-1.4, 4.1-4.3
 */

import {afterEach, beforeEach, describe, expect, it} from 'vitest';
import {BrowserManager} from '../browser-manager.js';
import type {Cookie, ICookieManager} from '../types/index.js';

/**
 * Mock CookieManager for testing
 */
class MockCookieManager implements ICookieManager {
  private cookies: Cookie[] | null = null;
  private hasCookies = false;

  setCookies(cookies: Cookie[] | null): void {
    this.cookies = cookies;
    this.hasCookies = cookies !== null && cookies.length > 0;
  }

  async loadCookies(): Promise<Cookie[] | null> {
    return this.cookies;
  }

  async saveCookies(cookies: Cookie[]): Promise<void> {
    this.cookies = cookies;
    this.hasCookies = true;
  }

  hasSavedCookies(): boolean {
    return this.hasCookies;
  }

  async clearCookies(): Promise<void> {
    this.cookies = null;
    this.hasCookies = false;
  }
}

describe('BrowserManager', () => {
  let browserManager: BrowserManager;
  let mockCookieManager: MockCookieManager;

  beforeEach(() => {
    mockCookieManager = new MockCookieManager();
    browserManager = new BrowserManager(mockCookieManager);
  });

  afterEach(async () => {
    // Clean up browser resources after each test
    await browserManager.close();
  });

  describe('constructor', () => {
    it('should create a BrowserManager instance', () => {
      expect(browserManager).toBeInstanceOf(BrowserManager);
    });

    it('should not be initialized after construction', () => {
      expect(browserManager.isInitialized()).toBe(false);
    });
  });

  describe('getPage before initialization', () => {
    it('should throw error when getPage is called before initialize', () => {
      expect(() => browserManager.getPage()).toThrow('浏览器未初始化');
    });
  });

  describe('navigateTo before initialization', () => {
    it('should throw error when navigateTo is called before initialize', async () => {
      await expect(browserManager.navigateTo('https://example.com')).rejects.toThrow('浏览器未初始化');
    });
  });

  describe('getContext before initialization', () => {
    it('should return null when getContext is called before initialize', () => {
      expect(browserManager.getContext()).toBeNull();
    });
  });

  describe('getCookies before initialization', () => {
    it('should return empty array when getCookies is called before initialize', async () => {
      const cookies = await browserManager.getCookies();
      expect(cookies).toEqual([]);
    });
  });

  describe('close', () => {
    it('should be safe to call close multiple times', async () => {
      await browserManager.close();
      await browserManager.close();
      // Should not throw
      expect(browserManager.isInitialized()).toBe(false);
    });

    it('should be safe to call close before initialize', async () => {
      await browserManager.close();
      expect(browserManager.isInitialized()).toBe(false);
    });
  });

  describe('isInitialized', () => {
    it('should return false before initialization', () => {
      expect(browserManager.isInitialized()).toBe(false);
    });
  });
});

/**
 * Integration tests that require actual browser
 * These tests are slower and require Playwright to be properly installed
 * 
 * To run these tests, first install Playwright browsers:
 *   npx playwright install chromium
 * 
 * Then run with: npm test -- --run browser-manager.test.ts
 */
describe.skipIf(!process.env.RUN_BROWSER_TESTS)('BrowserManager Integration', () => {
  let browserManager: BrowserManager;
  let mockCookieManager: MockCookieManager;

  beforeEach(() => {
    mockCookieManager = new MockCookieManager();
    browserManager = new BrowserManager(mockCookieManager);
  });

  afterEach(async () => {
    await browserManager.close();
  });

  describe('initialize', () => {
    it('should initialize browser in headless mode', async () => {
      // Requirement 1.1: Launch browser
      await browserManager.initialize(true);
      
      expect(browserManager.isInitialized()).toBe(true);
      expect(browserManager.getPage()).toBeDefined();
      expect(browserManager.getContext()).not.toBeNull();
    }, 30000);

    it('should load cookies during initialization', async () => {
      // Requirement 1.3: Load saved cookies before navigation
      const testCookies: Cookie[] = [
        {
          name: 'test_cookie',
          value: 'test_value',
          domain: '.example.com',
          path: '/',
          expires: Date.now() / 1000 + 3600, // 1 hour from now
          httpOnly: false,
          secure: false,
          sameSite: 'Lax'
        }
      ];
      
      mockCookieManager.setCookies(testCookies);
      
      await browserManager.initialize(true);
      
      expect(browserManager.isInitialized()).toBe(true);
    }, 30000);

    it('should handle cookie loading failure gracefully', async () => {
      // Create a mock that throws on loadCookies
      const failingCookieManager: ICookieManager = {
        loadCookies: async () => { throw new Error('Cookie load failed'); },
        saveCookies: async () => {},
        hasSavedCookies: () => true,
        clearCookies: async () => {}
      };
      
      const manager = new BrowserManager(failingCookieManager);
      
      // Should not throw, just warn
      await manager.initialize(true);
      expect(manager.isInitialized()).toBe(true);
      
      await manager.close();
    }, 30000);
  });

  describe('navigateTo', () => {
    beforeEach(async () => {
      await browserManager.initialize(true);
    });

    it('should navigate to a valid URL', async () => {
      // Requirement 1.2, 4.1: Navigate to URL
      await browserManager.navigateTo('https://example.com');
      
      const page = browserManager.getPage();
      expect(page.url()).toContain('example.com');
    }, 30000);

    it('should wait for page to load', async () => {
      // Requirement 4.2: Wait for page elements to fully render
      await browserManager.navigateTo('https://example.com');
      
      const page = browserManager.getPage();
      // Page should be loaded and have content
      const title = await page.title();
      expect(title).toBeDefined();
    }, 30000);
  });

  describe('getCookies', () => {
    beforeEach(async () => {
      await browserManager.initialize(true);
    });

    it('should return cookies from browser context', async () => {
      // Navigate to a page first to establish context
      await browserManager.navigateTo('https://example.com');
      
      const cookies = await browserManager.getCookies();
      // Cookies array should be defined (may be empty for example.com)
      expect(Array.isArray(cookies)).toBe(true);
    }, 30000);
  });

  describe('close after initialization', () => {
    it('should properly close browser and clean up resources', async () => {
      await browserManager.initialize(true);
      expect(browserManager.isInitialized()).toBe(true);
      
      await browserManager.close();
      expect(browserManager.isInitialized()).toBe(false);
    }, 30000);
  });
});
