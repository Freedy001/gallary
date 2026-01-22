/**
 * CNB Browser Automation - Custom Error Classes
 * 
 * 自定义错误类，用于提供更具体的错误信息和类型化的错误处理。
 * 
 * 需求覆盖:
 * - 1.4: 浏览器启动失败错误处理
 * - 3.5: 登录超时错误处理
 * - 4.3: 导航错误处理（404等）
 * - 5.4: 元素未找到错误处理
 * - 6.4: Cookie 操作错误处理
 */

/**
 * 浏览器启动错误
 * 
 * 当 Playwright 浏览器启动失败时抛出此错误。
 * 可能的原因包括：Playwright 未正确安装、系统资源不足等。
 * 
 * @example
 * throw new BrowserLaunchError('Chromium 未安装', originalError);
 */
export class BrowserLaunchError extends Error {
  public readonly cause?: Error;

  constructor(message: string, cause?: Error) {
    super(`浏览器启动失败: ${message}`);
    this.name = 'BrowserLaunchError';
    this.cause = cause;
    
    // 维护正确的原型链（TypeScript 编译到 ES5 时需要）
    Object.setPrototypeOf(this, BrowserLaunchError.prototype);
  }
}

/**
 * 登录超时错误
 * 
 * 当等待用户登录超过指定时间后抛出此错误。
 * 默认超时时间为 5 分钟（300000 毫秒）。
 * 
 * @example
 * throw new LoginTimeoutError(300000); // 5分钟超时
 */
export class LoginTimeoutError extends Error {
  public readonly timeoutMs: number;

  constructor(timeoutMs: number) {
    super(`登录超时: 等待 ${timeoutMs / 1000} 秒后仍未检测到登录成功`);
    this.name = 'LoginTimeoutError';
    this.timeoutMs = timeoutMs;
    
    Object.setPrototypeOf(this, LoginTimeoutError.prototype);
  }
}

/**
 * 导航错误
 * 
 * 当页面导航失败时抛出此错误。
 * 包括 404 页面不存在、网络错误等情况。
 * 
 * @example
 * throw new NavigationError('https://cnb.cool/user/repo', 404);
 */
export class NavigationError extends Error {
  public readonly url: string;
  public readonly statusCode?: number;

  constructor(url: string, statusCode?: number) {
    super(`导航失败: ${url}${statusCode ? ` (HTTP ${statusCode})` : ''}`);
    this.name = 'NavigationError';
    this.url = url;
    this.statusCode = statusCode;
    
    Object.setPrototypeOf(this, NavigationError.prototype);
  }
}

/**
 * Cookie 操作错误
 * 
 * 当 Cookie 加载或保存失败时抛出此错误。
 * 可能的原因包括：文件权限问题、JSON 解析错误、文件损坏等。
 * 
 * @example
 * throw new CookieError('文件不存在', 'load');
 * throw new CookieError('磁盘空间不足', 'save');
 */
export class CookieError extends Error {
  public readonly operation: 'load' | 'save';

  constructor(message: string, operation: 'load' | 'save') {
    super(`Cookie ${operation === 'load' ? '加载' : '保存'}失败: ${message}`);
    this.name = 'CookieError';
    this.operation = operation;
    
    Object.setPrototypeOf(this, CookieError.prototype);
  }
}

/**
 * 元素未找到错误
 * 
 * 当页面上找不到指定元素时抛出此错误。
 * 通常用于云原生开发按钮、登录按钮等关键元素的查找失败。
 * 
 * @example
 * throw new ElementNotFoundError('button:has-text("云原生开发")', '云原生开发按钮');
 */
export class ElementNotFoundError extends Error {
  public readonly selector: string;
  public readonly description: string;

  constructor(selector: string, description: string) {
    super(`未找到元素: ${description} (选择器: ${selector})`);
    this.name = 'ElementNotFoundError';
    this.selector = selector;
    this.description = description;
    
    Object.setPrototypeOf(this, ElementNotFoundError.prototype);
  }
}
