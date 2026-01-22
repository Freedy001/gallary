/**
 * CNB 浏览器自动化 - 登录检测器
 * 
 * 处理登录状态检测和登录流程管理。
 * 需求: 2.1-2.4, 3.1-3.5
 */

import type {Page} from 'playwright';
import {type ILoginDetector, type LoginSelectors, LoginStatus} from './types/index.js';

/** 默认登录相关选择器 */
const DEFAULT_LOGIN_SELECTORS: LoginSelectors = {
  wechatLoginButton: 'text=微信登录',
  userAvatar: '.user-avatar, [class*="avatar"]',
  workbench: 'text=工作台'
};

/** 默认检查间隔时间（毫秒）（2秒） */
const DEFAULT_CHECK_INTERVAL = 2000;

/**
 * LoginDetector 处理登录状态检测和管理登录流程。
 * 
 * 它可以：
 * - 通过查找特定页面元素来检查用户是否已登录
 * - 通过点击登录按钮触发微信登录流程
 * - 通过轮询等待登录完成
 */
export class LoginDetector implements ILoginDetector {
  private selectors: LoginSelectors;
  private checkInterval: number;

  /**
   * 创建新的 LoginDetector 实例。
   * @param selectors - 登录元素的自定义选择器（可选）
   * @param checkInterval - 登录状态检查间隔（毫秒）（默认: 2000）
   */
  constructor(selectors?: Partial<LoginSelectors>, checkInterval?: number) {
    this.selectors = {
      ...DEFAULT_LOGIN_SELECTORS,
      ...selectors
    };
    this.checkInterval = checkInterval ?? DEFAULT_CHECK_INTERVAL;
  }

  /**
   * 检查页面的当前登录状态。
   * 
   * 需求 2.1: 检查页面头部是否存在“微信登录”按钮
   * 需求 2.2: 如果存在“微信登录”按钮，用户未登录
   * 需求 2.3: 如果存在用户头像或“工作台”文字，用户已登录
   * 需求 2.4: 返回布尔型状态表示登录状态
   * 
   * @param page - Playwright Page 对象
   * @returns 表示当前登录状态的 LoginStatus
   */
  async checkLoginStatus(page: Page): Promise<LoginStatus> {
    try {
      // 首先检查已登录指示器（用户头像或工作台文字）
      // 需求 2.3: 检测用户头像或“工作台”文字以确认已登录状态
      const [hasAvatar, hasWorkbench] = await Promise.all([
        page.locator(this.selectors.userAvatar).first().isVisible({ timeout: 1000 }).catch(() => false),
        page.locator(this.selectors.workbench).first().isVisible({ timeout: 1000 }).catch(() => false)
      ]);

      if (hasAvatar || hasWorkbench) {
        return LoginStatus.LOGGED_IN;
      }

      // 检查未登录指示器（微信登录按钮）
      // 需求 2.1, 2.2: 检查“微信登录”按钮
      const hasLoginButton = await page
        .locator(this.selectors.wechatLoginButton)
        .first()
        .isVisible({ timeout: 1000 })
        .catch(() => false);

      if (hasLoginButton) {
        return LoginStatus.NOT_LOGGED_IN;
      }

      // 既未找到已登录指示器也未找到登录按钮 - 状态未知
      return LoginStatus.UNKNOWN;
    } catch (error) {
      // 检测过程中的任何错误都导致未知状态
      return LoginStatus.UNKNOWN;
    }
  }

  /**
   * 等待用户完成登录过程。
   * 
   * 需求 3.3: 每 2 秒轮询一次登录状态
   * 需求 3.5: 指定时间后超时（默认 5 分钟）
   * 
   * @param page - Playwright Page 对象
   * @param timeoutMs - 等待登录的最大时间（毫秒）
   * @returns 登录成功返回 true，超时返回 false
   */
  async waitForLogin(page: Page, timeoutMs: number): Promise<boolean> {
    const startTime = Date.now();
    
    // 需求 3.1: 控制台提示由调用者（主流程）处理
    // 此方法专注于轮询逻辑
    
    while (Date.now() - startTime < timeoutMs) {
      // 需求 3.3: 每隔 checkInterval（默认 2 秒）检查一次登录状态
      const status = await this.checkLoginStatus(page);
      
      if (status === LoginStatus.LOGGED_IN) {
        return true;
      }
      
      // 等待检查间隔后再次检查
      await this.sleep(this.checkInterval);
    }
    
    // 需求 3.5: 超时时间已到但未成功登录
    return false;
  }

  /**
   * 通过点击微信登录按钮触发登录流程。
   * 
   * 需求 3.2: 点击“微信登录”按钮触发登录弹窗
   * 
   * @param page - Playwright Page 对象
   * @throws 未找到登录按钮时抛出错误
   */
  async triggerLogin(page: Page): Promise<void> {
    const loginButton = page.locator(this.selectors.wechatLoginButton).first();
    
    // 等待按钮可见
    await loginButton.waitFor({ state: 'visible', timeout: 10000 });
    
    // 点击登录按钮触发微信登录弹窗
    await loginButton.click();
  }

  /**
   * 获取当前选择器配置。
   * 用于调试和测试。
   */
  getSelectors(): LoginSelectors {
    return { ...this.selectors };
  }

  /**
   * 获取当前检查间隔时间。
   * 用于调试和测试。
   */
  getCheckInterval(): number {
    return this.checkInterval;
  }

  /**
   * 用于睡眠指定时间的辅助方法。
   * @param ms - 睡眠时间（毫秒）
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
