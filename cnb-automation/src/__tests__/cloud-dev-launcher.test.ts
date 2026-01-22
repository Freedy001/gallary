/**
 * CNB Browser Automation - Cloud Dev Launcher Unit Tests
 * 
 * Tests for CloudDevLauncher class functionality.
 * Requirements: 5.1-5.4
 */

import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {CloudDevLauncher} from '../cloud-dev-launcher.js';
import type {BrowserContext, Locator, Page, Response} from 'playwright';

/**
 * Creates a mock Playwright Locator object for testing
 */
function createMockLocator(options: {
  isVisible?: boolean;
  shouldThrow?: boolean;
}): Locator {
  const { isVisible = false, shouldThrow = false } = options;

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
}

/**
 * Creates a mock Playwright Page object for testing
 */
function createMockPage(options: {
  hasCloudDevButton?: boolean;
  hasForkButton?: boolean;
  throwError?: boolean;
  currentUrl?: string;
  responseStatus?: number;
  pageCount?: number;
}): Page {
  const { 
    hasCloudDevButton = false, 
    hasForkButton = false, 
    throwError = false,
    currentUrl = 'https://cnb.cool/test/repo',
    responseStatus = 200,
    pageCount = 1
  } = options;

  const mockResponse = {
    status: vi.fn().mockReturnValue(responseStatus)
  } as unknown as Response;

  const mockContext = {
    pages: vi.fn().mockReturnValue(Array(pageCount).fill({} as Page))
  } as unknown as BrowserContext;

  const page = {
    locator: vi.fn().mockImplementation((selector: string) => {
      if (throwError) {
        return createMockLocator({ shouldThrow: true });
      }
      
      if (selector.includes('云原生开发')) {
        return createMockLocator({ isVisible: hasCloudDevButton });
      }
      if (selector.includes('Fork')) {
        return createMockLocator({ isVisible: hasForkButton });
      }
      return createMockLocator({ isVisible: false });
    }),
    goto: vi.fn().mockResolvedValue(mockResponse),
    waitForLoadState: vi.fn().mockResolvedValue(undefined),
    url: vi.fn().mockReturnValue(currentUrl),
    context: vi.fn().mockReturnValue(mockContext)
  } as unknown as Page;

  return page;
}

describe('CloudDevLauncher', () => {
  let consoleSpy: {
    log: ReturnType<typeof vi.spyOn>;
    error: ReturnType<typeof vi.spyOn>;
  };

  beforeEach(() => {
    consoleSpy = {
      log: vi.spyOn(console, 'log').mockImplementation(() => {}),
      error: vi.spyOn(console, 'error').mockImplementation(() => {})
    };
  });

  afterEach(() => {
    consoleSpy.log.mockRestore();
    consoleSpy.error.mockRestore();
  });

  describe('constructor', () => {
    it('should use default selectors when none provided', () => {
      const launcher = new CloudDevLauncher();
      const selectors = launcher.getSelectors();
      
      expect(selectors.cloudDevButton).toBe('button:has-text("云原生开发")');
      expect(selectors.forkButton).toBe('button:has-text("Fork")');
    });

    it('should use default timeouts when none provided', () => {
      const launcher = new CloudDevLauncher();
      
      expect(launcher.getElementTimeout()).toBe(10000);
      expect(launcher.getLaunchTimeout()).toBe(60000);
    });

    it('should allow custom selectors', () => {
      const customSelectors = {
        cloudDevButton: 'button.cloud-dev',
        forkButton: 'button.fork'
      };
      const launcher = new CloudDevLauncher(customSelectors);
      const selectors = launcher.getSelectors();
      
      expect(selectors.cloudDevButton).toBe('button.cloud-dev');
      expect(selectors.forkButton).toBe('button.fork');
    });

    it('should allow custom timeouts', () => {
      const launcher = new CloudDevLauncher(undefined, 5000, 30000);
      
      expect(launcher.getElementTimeout()).toBe(5000);
      expect(launcher.getLaunchTimeout()).toBe(30000);
    });

    it('should allow partial custom selectors', () => {
      const customSelectors = {
        cloudDevButton: 'button.custom-cloud-dev'
      };
      const launcher = new CloudDevLauncher(customSelectors);
      const selectors = launcher.getSelectors();
      
      expect(selectors.cloudDevButton).toBe('button.custom-cloud-dev');
      expect(selectors.forkButton).toBe('button:has-text("Fork")'); // Default preserved
    });
  });

  describe('navigateToRepo', () => {
    it('should navigate to the repository URL', async () => {
      const page = createMockPage({});
      const launcher = new CloudDevLauncher();
      const repoUrl = 'https://cnb.cool/test/repo';
      
      await launcher.navigateToRepo(page, repoUrl);
      
      expect(page.goto).toHaveBeenCalledWith(repoUrl, {
        waitUntil: 'networkidle',
        timeout: 60000
      });
      expect(page.waitForLoadState).toHaveBeenCalledWith('domcontentloaded');
      expect(page.waitForLoadState).toHaveBeenCalledWith('networkidle');
    });

    it('should throw error for 404 response', async () => {
      const page = createMockPage({ responseStatus: 404 });
      const launcher = new CloudDevLauncher();
      const repoUrl = 'https://cnb.cool/nonexistent/repo';
      
      await expect(launcher.navigateToRepo(page, repoUrl))
        .rejects.toThrow('仓库页面不存在 (404)');
      
      expect(consoleSpy.error).toHaveBeenCalledWith(
        expect.stringContaining('仓库页面不存在 (404)')
      );
    });

    it('should throw error for other 4xx/5xx responses', async () => {
      const page = createMockPage({ responseStatus: 500 });
      const launcher = new CloudDevLauncher();
      const repoUrl = 'https://cnb.cool/test/repo';
      
      await expect(launcher.navigateToRepo(page, repoUrl))
        .rejects.toThrow('仓库页面加载失败 (HTTP 500)');
      
      expect(consoleSpy.error).toHaveBeenCalledWith(
        expect.stringContaining('仓库页面加载失败 (HTTP 500)')
      );
    });

    it('should handle navigation errors', async () => {
      const page = createMockPage({});
      (page.goto as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'));
      
      const launcher = new CloudDevLauncher();
      const repoUrl = 'https://cnb.cool/test/repo';
      
      await expect(launcher.navigateToRepo(page, repoUrl))
        .rejects.toThrow('Network error');
      
      expect(consoleSpy.error).toHaveBeenCalledWith(
        expect.stringContaining('导航到仓库失败')
      );
    });
  });

  describe('launchCloudDev', () => {
    /**
     * Requirement 5.1: Locate the "云原生开发" button when repository page loads
     * Requirement 5.2: Click the button when found
     * Requirement 5.3: Output "云原生开发环境启动中..." when click succeeds
     */
    it('should find and click the cloud dev button successfully', async () => {
      const page = createMockPage({ hasCloudDevButton: true });
      const launcher = new CloudDevLauncher();
      
      const result = await launcher.launchCloudDev(page);
      
      expect(result).toBe(true);
      expect(page.locator).toHaveBeenCalledWith('button:has-text("云原生开发")');
      expect(consoleSpy.log).toHaveBeenCalledWith('云原生开发环境启动中...');
    });

    /**
     * Requirement 5.4: Output error message if button is not found
     */
    it('should return false and log error when button is not found', async () => {
      const page = createMockPage({ hasCloudDevButton: false });
      const launcher = new CloudDevLauncher();
      
      const result = await launcher.launchCloudDev(page);
      
      expect(result).toBe(false);
      expect(consoleSpy.error).toHaveBeenCalledWith(
        '未找到"云原生开发"按钮，请确认当前页面是否为仓库页面'
      );
    });

    it('should return false and log error when an exception occurs during click', async () => {
      // Create a page where button is visible but click throws an error
      const mockLocator = {
        first: vi.fn().mockReturnThis(),
        isVisible: vi.fn().mockResolvedValue(true),
        waitFor: vi.fn().mockResolvedValue(undefined),
        click: vi.fn().mockRejectedValue(new Error('Click failed'))
      } as unknown as Locator;

      const page = {
        locator: vi.fn().mockReturnValue(mockLocator)
      } as unknown as Page;
      
      const launcher = new CloudDevLauncher();
      
      const result = await launcher.launchCloudDev(page);
      
      expect(result).toBe(false);
      expect(consoleSpy.error).toHaveBeenCalledWith(
        expect.stringContaining('启动云原生开发环境失败')
      );
    });

    it('should use custom selector when provided', async () => {
      const page = createMockPage({ hasCloudDevButton: true });
      const customSelectors = { cloudDevButton: 'button.custom-cloud-dev' };
      const launcher = new CloudDevLauncher(customSelectors);
      
      await launcher.launchCloudDev(page);
      
      expect(page.locator).toHaveBeenCalledWith('button.custom-cloud-dev');
    });
  });

  describe('waitForLaunch', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('should detect URL change and return', async () => {
      let urlCallCount = 0;
      const page = createMockPage({});
      (page.url as ReturnType<typeof vi.fn>).mockImplementation(() => {
        urlCallCount++;
        // Return different URL after first call
        return urlCallCount > 1 
          ? 'https://cnb.cool/cloud-dev/environment' 
          : 'https://cnb.cool/test/repo';
      });
      
      const launcher = new CloudDevLauncher();
      const waitPromise = launcher.waitForLaunch(page);
      
      // Advance timers to allow the check loop to run
      await vi.advanceTimersByTimeAsync(1000);
      
      await waitPromise;
      
      expect(consoleSpy.log).toHaveBeenCalledWith('云原生开发环境页面已打开');
    });

    it('should detect new tab opening and return', async () => {
      let pageCountCallCount = 0;
      const mockContext = {
        pages: vi.fn().mockImplementation(() => {
          pageCountCallCount++;
          // Return 2 pages after first call (simulating new tab)
          return pageCountCallCount > 1 
            ? [{}, {}] 
            : [{}];
        })
      } as unknown as BrowserContext;
      
      const page = createMockPage({});
      (page.context as ReturnType<typeof vi.fn>).mockReturnValue(mockContext);
      
      const launcher = new CloudDevLauncher();
      const waitPromise = launcher.waitForLaunch(page);
      
      // Advance timers to allow the check loop to run
      await vi.advanceTimersByTimeAsync(1000);
      
      await waitPromise;
      
      expect(consoleSpy.log).toHaveBeenCalledWith('云原生开发环境已在新标签页中打开');
    });

    it('should log timeout message when launch takes too long', async () => {
      const page = createMockPage({});
      const launcher = new CloudDevLauncher(undefined, undefined, 2000); // 2 second timeout
      
      const waitPromise = launcher.waitForLaunch(page);
      
      // Advance timers past the timeout
      await vi.advanceTimersByTimeAsync(3000);
      
      await waitPromise;
      
      expect(consoleSpy.log).toHaveBeenCalledWith(
        '等待云原生开发环境启动超时，请检查浏览器窗口'
      );
    });
  });

  describe('getSelectors', () => {
    it('should return a copy of selectors', () => {
      const launcher = new CloudDevLauncher();
      const selectors1 = launcher.getSelectors();
      const selectors2 = launcher.getSelectors();
      
      expect(selectors1).toEqual(selectors2);
      expect(selectors1).not.toBe(selectors2); // Different object references
    });
  });
});
