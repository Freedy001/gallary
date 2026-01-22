/**
 * CNB 浏览器自动化 - Cookie 管理器
 * 
 * 处理 cookie 持久化以在会话之间保持登录状态。
 * 需求: 1.3, 3.4, 6.1-6.4
 */

import {existsSync, mkdirSync, readFileSync, unlinkSync, writeFileSync} from 'node:fs';
import {dirname} from 'node:path';
import type {Cookie, CookieStorage, ICookieManager} from './types/index.js';

/** 当前 cookie 存储格式版本 */
const COOKIE_STORAGE_VERSION = 1;

/**
 * CookieManager 处理浏览器 cookies 的加载、保存和管理，
 * 用于持久化登录状态。
 */
export class CookieManager implements ICookieManager {
  private cookiePath: string;

  /**
   * 创建新的 CookieManager 实例。
   * @param cookiePath - cookie 存储文件的路径
   */
  constructor(cookiePath: string) {
    this.cookiePath = this.expandPath(cookiePath);
  }

  /**
   * 检查是否存在已保存的 cookies 文件。
   * 需求 6.2: 启动时检查现有的 cookies 文件
   */
  hasSavedCookies(): boolean {
    return existsSync(this.cookiePath);
  }

  /**
   * 从存储文件加载 cookies。
   * 需求 1.3: 导航前加载 cookies
   * 需求 6.3: 将 cookies 加载到浏览器上下文
   * 需求 6.4: 处理无效/过期的 cookies
   *
   * @returns cookies 数组，如果不存在有效 cookies 则返回 null
   */
  async loadCookies(): Promise<Cookie[] | null> {
    if (!this.hasSavedCookies()) {
      return null;
    }

    try {
      const content = readFileSync(this.cookiePath, 'utf-8');
      const storage: CookieStorage = JSON.parse(content);

      // 验证存储格式
      if (!storage.version || !Array.isArray(storage.cookies)) {
        // 格式无效 - 清除并返回 null
        await this.clearCookies();
        return null;
      }

      // 过滤过期的 cookies
      const now = Date.now() / 1000; // 转换为秒以进行比较
      const validCookies = storage.cookies.filter(cookie => {
        // expires = -1 表示会话 cookie，保留它
        // expires = 0 或 undefined 表示未设置过期时间
        if (cookie.expires <= 0) {
          return true;
        }
        return cookie.expires > now;
      });

      // 如果所有 cookies 都已过期，清除文件并返回 null
      if (validCookies.length === 0 && storage.cookies.length > 0) {
        await this.clearCookies();
        return null;
      }

      return validCookies;
    } catch (error) {
      // JSON 解析错误或文件读取错误 - 清除损坏的文件
      await this.clearCookies();
      return null;
    }
  }

  /**
   * 将 cookies 保存到存储文件。
   * 需求 3.4: 登录成功后保存 cookies
   * 需求 6.1: 序列化并保存 cookies 到本地文件
   *
   * @param cookies - 要保存的 cookies 数组
   */
  async saveCookies(cookies: Cookie[]): Promise<void> {
    this.ensureDirectory();

    const storage: CookieStorage = {
      version: COOKIE_STORAGE_VERSION,
      savedAt: new Date().toISOString(),
      cookies: cookies
    };

    const content = JSON.stringify(storage, null, 2);
    writeFileSync(this.cookiePath, content, 'utf-8');
  }

  /**
   * 通过删除存储文件清除已保存的 cookies。
   * 需求 6.4: cookies 无效时删除旧文件
   */
  async clearCookies(): Promise<void> {
    if (this.hasSavedCookies()) {
      try {
        unlinkSync(this.cookiePath);
      } catch {
        // 忽略删除时的错误 - 文件可能已经不存在
      }
    }
  }

  /**
   * 获取解析后的 cookie 文件路径。
   * 用于调试和日志记录。
   */
  getCookiePath(): string {
    return this.cookiePath;
  }

  /**
   * 将路径中的 ~ 展开为主目录。
   */
  private expandPath(path: string): string {
    if (path.startsWith('~')) {
      const home = process.env.HOME || process.env.USERPROFILE || '';
      return path.replace('~', home);
    }
    return path;
  }

  /**
   * 确保 cookie 文件所在的目录存在。
   */
  private ensureDirectory(): void {
    const dir = dirname(this.cookiePath);
    if (!existsSync(dir)) {
      mkdirSync(dir, { recursive: true });
    }
  }
}
