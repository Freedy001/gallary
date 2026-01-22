/**
 * CNB Browser Automation - Login Detector Unit Tests
 * 
 * Tests for LoginDetector class functionality.
 * Requirements: 2.1-2.4, 3.1-3.5
 */

import {beforeEach, describe, expect, it, vi} from 'vitest';
import {LoginDetector} from '../login-detector.js';
import {LoginStatus} from '../types/index.js';
import type {Locator, Page} from 'playwright';

/**
 * Creates a mock Playwright Page object for testing
 */
function createMockPage(options: {
  hasLoginButton?: boolean;
  hasAvatar?: boolean;
  hasWorkbench?: boolean;
  throwError?: boolean;
}): Page {
  const { hasLoginButton = false, hasAvatar = false, hasWorkbench = false, throwError = false } = options;

  const createMockLocator = (isVisible: boolean, shouldThrow = false): Locator => {
    const locator = {
      first: vi.fn().mockReturnThis(),
      isVisible: vi.fn().mockImplementation(async () => {
        if (shouldThrow) {
          throw new Error('Element not found');
        }
        return isVisible;
      }),
      waitFor: vi.fn().mockResolvedValue(undefined),
      click: vi.fn().mockResolvedValue(undefined)
    } as unknown as Locator;
    return locator;
  };

  const page = {
    locator: vi.fn().mockImplementation((selector: string) => {
      if (throwError) {
        return createMockLocator(false, true);
      }
      
      if (selector.includes('微信登录')) {
        return createMockLocator(hasLoginButton);
      }
      if (selector.includes('avatar')) {
        return createMockLocator(hasAvatar);
      }
      if (selector.includes('工作台')) {
        return createMockLocator(hasWorkbench);
      }
      return createMockLocator(false);
    })
  } as unknown as Page;

  return page;
}

describe('LoginDetector', () => {
  describe('constructor', () => {
    it('should use default selectors when none provided', () => {
      const detector = new LoginDetector();
      const selectors = detector.getSelectors();
      
      expect(selectors.wechatLoginButton).toBe('text=微信登录');
      expect(selectors.userAvatar).toBe('.user-avatar, [class*="avatar"]');
      expect(selectors.workbench).toBe('text=工作台');
    });

    it('should use default check interval when none provided', () => {
      const detector = new LoginDetector();
      expect(detector.getCheckInterval()).toBe(2000);
    });

    it('should allow custom selectors', () => {
      const customSelectors = {
        wechatLoginButton: 'button.wechat-login',
        userAvatar: '.custom-avatar'
      };
      const detector = new LoginDetector(customSelectors);
      const selectors = detector.getSelectors();
      
      expect(selectors.wechatLoginButton).toBe('button.wechat-login');
      expect(selectors.userAvatar).toBe('.custom-avatar');
      expect(selectors.workbench).toBe('text=工作台'); // Default preserved
    });

    it('should allow custom check interval', () => {
      const detector = new LoginDetector(undefined, 5000);
      expect(detector.getCheckInterval()).toBe(5000);
    });
  });

  describe('checkLoginStatus', () => {
    /**
     * Requirement 2.3: When user avatar is detected, return LOGGED_IN status
     */
    it('should return LOGGED_IN when user avatar is visible', async () => {
      const page = createMockPage({ hasAvatar: true });
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.LOGGED_IN);
    });

    /**
     * Requirement 2.3: When "工作台" text is detected, return LOGGED_IN status
     */
    it('should return LOGGED_IN when workbench text is visible', async () => {
      const page = createMockPage({ hasWorkbench: true });
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.LOGGED_IN);
    });

    /**
     * Requirement 2.3: Either avatar OR workbench indicates logged in
     */
    it('should return LOGGED_IN when both avatar and workbench are visible', async () => {
      const page = createMockPage({ hasAvatar: true, hasWorkbench: true });
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.LOGGED_IN);
    });

    /**
     * Requirement 2.1, 2.2: When "微信登录" button is detected, return NOT_LOGGED_IN
     */
    it('should return NOT_LOGGED_IN when login button is visible', async () => {
      const page = createMockPage({ hasLoginButton: true });
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.NOT_LOGGED_IN);
    });

    /**
     * Requirement 2.3 takes precedence: logged-in indicators override login button
     */
    it('should return LOGGED_IN when both avatar and login button are visible', async () => {
      const page = createMockPage({ hasAvatar: true, hasLoginButton: true });
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.LOGGED_IN);
    });

    it('should return UNKNOWN when no indicators are found', async () => {
      const page = createMockPage({});
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.UNKNOWN);
    });

    it('should return UNKNOWN when an error occurs', async () => {
      const page = createMockPage({ throwError: true });
      const detector = new LoginDetector();
      
      const status = await detector.checkLoginStatus(page);
      
      expect(status).toBe(LoginStatus.UNKNOWN);
    });
  });

  describe('triggerLogin', () => {
    /**
     * Requirement 3.2: Click "微信登录" button to trigger login popup
     */
    it('should click the login button', async () => {
      const mockClick = vi.fn().mockResolvedValue(undefined);
      const mockWaitFor = vi.fn().mockResolvedValue(undefined);
      
      const page = {
        locator: vi.fn().mockReturnValue({
          first: vi.fn().mockReturnValue({
            waitFor: mockWaitFor,
            click: mockClick
          })
        })
      } as unknown as Page;
      
      const detector = new LoginDetector();
      await detector.triggerLogin(page);
      
      expect(page.locator).toHaveBeenCalledWith('text=微信登录');
      expect(mockWaitFor).toHaveBeenCalledWith({ state: 'visible', timeout: 10000 });
      expect(mockClick).toHaveBeenCalled();
    });
  });

  describe('waitForLogin', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    /**
     * Requirement 3.3: Poll login status every 2 seconds
     * Requirement 3.5: Return false if timeout is reached
     */
    it('should return true immediately if already logged in', async () => {
      const page = createMockPage({ hasAvatar: true });
      const detector = new LoginDetector();
      
      const resultPromise = detector.waitForLogin(page, 10000);
      
      // Advance timers to allow the async operation to complete
      await vi.runAllTimersAsync();
      
      const result = await resultPromise;
      expect(result).toBe(true);
    });

    /**
     * Requirement 3.5: Return false after timeout
     */
    it('should return false after timeout if not logged in', async () => {
      const page = createMockPage({ hasLoginButton: true });
      const detector = new LoginDetector(undefined, 1000); // 1 second interval
      
      const resultPromise = detector.waitForLogin(page, 3000); // 3 second timeout
      
      // Advance time past the timeout
      await vi.advanceTimersByTimeAsync(4000);
      
      const result = await resultPromise;
      expect(result).toBe(false);
    });

    /**
     * Requirement 3.3: Check interval should be approximately 2 seconds
     */
    it('should poll at the configured interval', async () => {
      let checkCount = 0;
      const mockLocator = {
        first: vi.fn().mockReturnThis(),
        isVisible: vi.fn().mockImplementation(async () => {
          checkCount++;
          // Return logged in after 3 checks
          return checkCount >= 3;
        })
      };
      
      const page = {
        locator: vi.fn().mockReturnValue(mockLocator)
      } as unknown as Page;
      
      const detector = new LoginDetector(undefined, 1000); // 1 second interval for faster test
      
      const resultPromise = detector.waitForLogin(page, 10000);
      
      // Advance time to allow multiple checks
      await vi.advanceTimersByTimeAsync(5000);
      
      const result = await resultPromise;
      expect(result).toBe(true);
      // Should have checked multiple times
      expect(checkCount).toBeGreaterThanOrEqual(3);
    });
  });
});
