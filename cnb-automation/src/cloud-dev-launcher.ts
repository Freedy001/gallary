/**
 * CNB 浏览器自动化 - 云原生开发启动器
 * 
 * 处理仓库页面导航和启动云原生开发环境。
 * 需求: 5.1-5.4
 */

import type {Page} from 'playwright';
import type {ICloudDevLauncher, RepoPageSelectors} from './types/index.js';

/** 默认仓库页面选择器 */
const DEFAULT_REPO_SELECTORS: RepoPageSelectors = {
  cloudDevButton: 'button:has-text("云原生开发")',
  forkButton: 'button:has-text("Fork")'
};

/** 默认元素等待超时时间（10秒） */
const DEFAULT_ELEMENT_TIMEOUT = 10000;

/** 默认环境启动等待超时时间（60秒） */
const DEFAULT_LAUNCH_TIMEOUT = 60000;

/**
 * CloudDevLauncher 处理仓库页面导航和启动云原生开发环境。
 * 
 * 它可以：
 * - 导航到 cnb.cool 上的特定仓库页面
 * - 查找并点击“云原生开发”按钮
 * - 等待开发环境开始启动
 */
export class CloudDevLauncher implements ICloudDevLauncher {
  private selectors: RepoPageSelectors;
  private elementTimeout: number;
  private launchTimeout: number;

  /**
   * 创建新的 CloudDevLauncher 实例。
   * @param selectors - 仓库页面元素的自定义选择器（可选）
   * @param elementTimeout - 等待元素的超时时间（毫秒）（默认: 10000）
   * @param launchTimeout - 等待启动的超时时间（毫秒）（默认: 60000）
   */
  constructor(
    selectors?: Partial<RepoPageSelectors>,
    elementTimeout?: number,
    launchTimeout?: number
  ) {
    this.selectors = {
      ...DEFAULT_REPO_SELECTORS,
      ...selectors
    };
    this.elementTimeout = elementTimeout ?? DEFAULT_ELEMENT_TIMEOUT;
    this.launchTimeout = launchTimeout ?? DEFAULT_LAUNCH_TIMEOUT;
  }

  /**
   * 导航到指定的仓库页面。
   * 
   * 需求 4.1: 指定仓库地址时导航到仓库页面
   * 
   * @param page - Playwright Page 对象
   * @param repoUrl - 要导航到的仓库 URL
   * @throws 导航失败或页面返回错误状态时抛出错误
   */
  async navigateToRepo(page: Page, repoUrl: string): Promise<void> {
    try {
      // 导航到仓库 URL 并等待网络空闲
      const response = await page.goto(repoUrl, {
        waitUntil: 'networkidle',
        timeout: this.launchTimeout
      });

      // 检查错误状态码
      if (response) {
        const status = response.status();
        
        if (status === 404) {
          console.error(`仓库页面不存在 (404): ${repoUrl}`);
          throw new Error(`仓库页面不存在 (404): ${repoUrl}`);
        }
        
        if (status >= 400) {
          console.error(`仓库页面加载失败 (HTTP ${status}): ${repoUrl}`);
          throw new Error(`仓库页面加载失败 (HTTP ${status}): ${repoUrl}`);
        }
      }

      // 等待页面完全渲染
      await page.waitForLoadState('domcontentloaded');
      await page.waitForLoadState('networkidle');

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      
      // 仅在不是我们自定义的错误时记录日志
      if (!errorMessage.includes('仓库页面不存在') && !errorMessage.includes('仓库页面加载失败')) {
        console.error(`导航到仓库失败: ${errorMessage}`);
      }
      
      throw error;
    }
  }

  /**
   * 查找并点击云原生开发按钮。
   * 
   * 需求 5.1: 仓库页面加载时定位橙色的“云原生开发”按钮
   * 需求 5.2: 找到后点击“云原生开发”按钮
   * 需求 5.3: 点击成功后输出“云原生开发环境启动中...”
   * 需求 5.4: 未找到“云原生开发”按钮时输出错误信息
   * 
   * @param page - Playwright Page 对象
   * @returns 如果成功找到并点击按钮返回 true，否则返回 false
   */
  async launchCloudDev(page: Page): Promise<boolean> {
    try {
      // 需求 5.1: 定位“云原生开发”按钮
      const cloudDevButton = page.locator(this.selectors.cloudDevButton).first();
      
      // 等待按钮可见
      const isVisible = await cloudDevButton
        .isVisible({ timeout: this.elementTimeout })
        .catch(() => false);

      if (!isVisible) {
        // 需求 5.4: 未找到按钮时输出错误信息
        console.error('未找到“云原生开发”按钮，请确认当前页面是否为仓库页面');
        return false;
      }
      
      // 等待按钮启用并可点击
      await cloudDevButton.waitFor({ 
        state: 'visible', 
        timeout: this.elementTimeout 
      });

      // 需求 5.2: 点击按钮
      await cloudDevButton.click();

      // 需求 5.3: 输出成功消息
      console.log('云原生开发环境启动中...');

      return true;
    } catch (error) {
      // 需求 5.4: 失败时输出错误信息
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.error(`启动云原生开发环境失败: ${errorMessage}`);
      return false;
    }
  }

  /**
   * 等待云原生开发环境开始启动。
   * 此方法等待页面导航或新标签页/窗口打开，
   * 表明环境正在准备中。
   * 
   * @param page - Playwright Page 对象
   * @throws 等待超时时抛出错误
   */
  async waitForLaunch(page: Page): Promise<void> {
    try {
      // 等待以下任一情况：
      // 1. URL 变化（导航到云开发环境）
      // 2. 新页面/标签页打开
      // 3. 出现加载指示器
      
      const startUrl = page.url();
      const startTime = Date.now();
      
      while (Date.now() - startTime < this.launchTimeout) {
        // 检查 URL 是否已变化（表明已导航到开发环境）
        const currentUrl = page.url();
        if (currentUrl !== startUrl) {
          console.log('云原生开发环境页面已打开');
          return;
        }

        // 检查是否有加载指示器或新窗口
        // 云开发环境可能在新标签页中打开
        const context = page.context();
        const pages = context.pages();
        
        if (pages.length > 1) {
          // 新标签页/窗口已打开
          console.log('云原生开发环境已在新标签页中打开');
          return;
        }

        // 等待一段时间后再次检查
        await this.sleep(500);
      }

      // 如果到达此处，启动可能仍在进行中
      // 但我们已等待足够长的时间
      console.log('等待云原生开发环境启动超时，请检查浏览器窗口');
      
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.error(`等待云原生开发环境启动时出错: ${errorMessage}`);
      throw error;
    }
  }

  /**
   * 获取当前选择器配置。
   * 用于调试和测试。
   */
  getSelectors(): RepoPageSelectors {
    return { ...this.selectors };
  }

  /**
   * 获取当前元素超时时间。
   * 用于调试和测试。
   */
  getElementTimeout(): number {
    return this.elementTimeout;
  }

  /**
   * 获取当前启动超时时间。
   * 用于调试和测试。
   */
  getLaunchTimeout(): number {
    return this.launchTimeout;
  }

  /**
   * 用于睡眠指定时间的辅助方法。
   * @param ms - 睡眠时间（毫秒）
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
