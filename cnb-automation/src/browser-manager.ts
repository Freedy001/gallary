/**
 * CNB 浏览器自动化 - 浏览器管理器
 * 
 * 处理浏览器生命周期管理，包括初始化、导航和清理。
 * 需求: 1.1-1.4, 4.1-4.3
 */

import {type Browser, type BrowserContext, chromium, type Page} from 'playwright';
import type {Cookie, IBrowserManager, ICookieManager} from './types/index.js';

/** CNB 平台默认 URL */
const CNB_BASE_URL = 'https://cnb.cool/';

/** 默认导航超时时间（毫秒） */
const DEFAULT_NAVIGATION_TIMEOUT = 30000;

/** 默认页面加载超时时间 */
const DEFAULT_LOAD_TIMEOUT = 60000;

/**
 * BrowserManager 处理浏览器生命周期，包括：
 * - 浏览器初始化（支持可选的无头模式）
 * - Cookie 加载以保持登录状态
 * - URL 导航及错误处理
 * - 浏览器清理和资源管理
 */
export class BrowserManager implements IBrowserManager {
  private browser: Browser | null = null;
  private context: BrowserContext | null = null;
  private page: Page | null = null;
  private cookieManager: ICookieManager;

  /**
   * 创建新的 BrowserManager 实例。
   * @param cookieManager - 用于加载/保存 cookies 的管理器
   */
  constructor(cookieManager: ICookieManager) {
    this.cookieManager = cookieManager;
  }

  /**
   * 初始化浏览器并加载已保存的 cookies。
   * 
   * 需求 1.1: 执行 CLI 命令时启动可见的浏览器窗口
   * 需求 1.3: 导航前加载已保存的 cookies
   * 需求 1.4: 浏览器启动失败时输出错误信息并退出
   * 
   * @param headless - 是否以无头模式运行浏览器（默认: false，显示窗口）
   * @throws 浏览器启动失败时抛出错误
   */
  async initialize(headless: boolean = false): Promise<void> {
    try {
      // 需求 1.1: 启动浏览器（默认可见）
      this.browser = await chromium.launch({
        headless: headless,
        // 额外选项以提高稳定性
        args: [
          '--disable-blink-features=AutomationControlled',
          '--no-sandbox',
          '--disable-setuid-sandbox'
        ]
      });

      // 创建浏览器上下文
      this.context = await this.browser.newContext({
        viewport: { width: 1280, height: 800 },
        userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
      });

      // 需求 1.3: 导航前加载已保存的 cookies
      await this.loadSavedCookies();

      // 创建新页面
      this.page = await this.context.newPage();

      // 设置默认超时时间
      this.page.setDefaultTimeout(DEFAULT_NAVIGATION_TIMEOUT);
      this.page.setDefaultNavigationTimeout(DEFAULT_LOAD_TIMEOUT);

    } catch (error) {
      // 需求 1.4: 浏览器启动失败时输出错误信息
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.error(`浏览器启动失败: ${errorMessage}`);
      
      // 清理部分初始化的资源
      await this.close();
      
      throw new Error(`浏览器启动失败: ${errorMessage}`);
    }
  }

  /**
   * 导航到指定的 URL。
   *
   * 需求 1.2: 浏览器启动后导航到 https://cnb.cool/
   * 需求 4.1: 指定仓库地址时导航到仓库页面
   * 需求 4.2: 等待页面元素完全渲染
   * 需求 4.3: URL 无效或返回 404 时输出错误信息
   *
   * @param url - 要导航到的 URL
   * @throws 导航失败或页面返回错误状态时抛出错误
   */
  async navigateTo(url: string): Promise<void> {
    if (!this.page) {
      throw new Error('浏览器未初始化，请先调用 initialize()');
    }

    try {
      // 导航到 URL 并等待网络空闲
      const response = await this.page.goto(url, {
        waitUntil: 'networkidle',
        timeout: DEFAULT_LOAD_TIMEOUT
      });

      // 需求 4.3: 检查错误状态码
      if (response) {
        const status = response.status();

        if (status === 404) {
          console.error(`页面不存在 (404): ${url}`);
          throw new Error(`页面不存在 (404): ${url}`);
        }

        if (status >= 400) {
          console.error(`页面加载失败 (HTTP ${status}): ${url}`);
          throw new Error(`页面加载失败 (HTTP ${status}): ${url}`);
        }
      }

      // 需求 4.2: 等待页面完全渲染
      // 等待文档进入就绪状态
      await this.page.waitForLoadState('domcontentloaded');

      // 额外等待动态内容加载
      await this.page.waitForLoadState('networkidle');

    } catch (error) {
      // 需求 4.3: 导航失败时输出错误信息
      const errorMessage = error instanceof Error ? error.message : String(error);

      // 仅在不是我们自定义的错误时记录日志
      if (!errorMessage.includes('页面不存在') && !errorMessage.includes('页面加载失败')) {
        console.error(`导航失败: ${errorMessage}`);
      }

      throw error;
    }
  }

  /**
   * 获取当前页面实例。
   *
   * @returns 当前的 Playwright Page 对象
   * @throws 浏览器未初始化时抛出错误
   */
  getPage(): Page {
    if (!this.page) {
      throw new Error('浏览器未初始化，请先调用 initialize()');
    }
    return this.page;
  }

  /**
   * 获取当前浏览器上下文。
   * 用于 cookie 操作。
   *
   * @returns 当前的 BrowserContext，如果未初始化则返回 null
   */
  getContext(): BrowserContext | null {
    return this.context;
  }

  /**
   * 从当前浏览器上下文获取 cookies。
   * 用于保存登录状态。
   *
   * @returns 浏览器上下文中的 cookies 数组
   */
  async getCookies(): Promise<Cookie[]> {
    if (!this.context) {
      return [];
    }

    const playwrightCookies = await this.context.cookies();

    return playwrightCookies.map(cookie => ({
      name: cookie.name,
      value: cookie.value,
      domain: cookie.domain,
      path: cookie.path,
      expires: cookie.expires,
      httpOnly: cookie.httpOnly,
      secure: cookie.secure,
      sameSite: cookie.sameSite as 'Strict' | 'Lax' | 'None'
    }));
  }

  /**
   * 关闭浏览器并清理资源。
   * 可以安全地多次调用。
   */
  async close(): Promise<void> {
    try {
      if (this.page) {
        await this.page.close().catch(() => {});
        this.page = null;
      }

      if (this.context) {
        await this.context.close().catch(() => {});
        this.context = null;
      }

      if (this.browser) {
        await this.browser.close().catch(() => {});
        this.browser = null;
      }
    } catch (error) {
      // 忽略清理过程中的错误
      this.page = null;
      this.context = null;
      this.browser = null;
    }
  }

  /**
   * 检查浏览器是否已初始化并运行中。
   *
   * @returns 如果浏览器已初始化返回 true，否则返回 false
   */
  isInitialized(): boolean {
    return this.browser !== null && this.page !== null;
  }

  /**
   * 将已保存的 cookies 加载到浏览器上下文中。
   * 需求 1.3: 如果存在 cookies 则在导航前加载
   */
  private async loadSavedCookies(): Promise<void> {
    if (!this.context) {
      return;
    }

    try {
      const cookies = await this.cookieManager.loadCookies();

      if (cookies && cookies.length > 0) {
        // 将我们的 Cookie 类型转换为 Playwright 的 cookie 格式
        const playwrightCookies = cookies.map(cookie => ({
          name: cookie.name,
          value: cookie.value,
          domain: cookie.domain,
          path: cookie.path,
          expires: cookie.expires > 0 ? cookie.expires : undefined,
          httpOnly: cookie.httpOnly,
          secure: cookie.secure,
          sameSite: cookie.sameSite as 'Strict' | 'Lax' | 'None'
        }));

        await this.context.addCookies(playwrightCookies);
      }
    } catch (error) {
      // Cookie 加载失败不应阻止浏览器工作
      // 只记录警告并继续
      console.warn('加载 cookies 失败，将以未登录状态继续');
    }
  }
}
