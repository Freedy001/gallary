/**
 * CNB Browser Automation - Type Definitions
 */

// ============ CLI Types ============

/**
 * CLI 命令行选项
 */
export interface CLIOptions {
  /** 目标仓库地址 */
  repo: string;
  /** 是否无头模式 */
  headless: boolean;
  /** 登录超时时间（秒） */
  timeout: number;
  /** Cookie 文件路径 */
  cookiePath?: string;
}

// ============ Cookie Types ============

/**
 * Cookie 数据结构
 */
export interface Cookie {
  name: string;
  value: string;
  domain: string;
  path: string;
  expires: number;
  httpOnly: boolean;
  secure: boolean;
  sameSite: 'Strict' | 'Lax' | 'None';
}

/**
 * Cookie 存储格式
 */
export interface CookieStorage {
  version: number;
  savedAt: string;
  cookies: Cookie[];
}

// ============ Login Types ============

/**
 * 登录状态枚举
 */
export enum LoginStatus {
  LOGGED_IN = 'logged_in',
  NOT_LOGGED_IN = 'not_logged_in',
  UNKNOWN = 'unknown'
}

/**
 * 登录相关选择器
 */
export interface LoginSelectors {
  /** 微信登录按钮选择器 */
  wechatLoginButton: string;
  /** 用户头像选择器 */
  userAvatar: string;
  /** 工作台文字选择器 */
  workbench: string;
}

// ============ Repo Page Types ============

/**
 * 仓库页面选择器
 */
export interface RepoPageSelectors {
  /** 云原生开发按钮选择器 */
  cloudDevButton: string;
  /** Fork 按钮选择器（用于定位参考） */
  forkButton: string;
}

// ============ Config Types ============

/**
 * 应用配置
 */
export interface Config {
  /** Cookie 文件存储路径 */
  cookieFilePath: string;
  /** 默认超时时间（毫秒） */
  defaultTimeout: number;
  /** 登录状态检查间隔（毫秒） */
  checkInterval: number;
  /** 选择器配置 */
  selectors: {
    login: LoginSelectors;
    repo: RepoPageSelectors;
  };
}

// ============ Interface Definitions ============

import type {Page} from 'playwright';

/**
 * Cookie 管理器接口
 */
export interface ICookieManager {
  /** 加载已保存的 cookies */
  loadCookies(): Promise<Cookie[] | null>;
  /** 保存 cookies 到文件 */
  saveCookies(cookies: Cookie[]): Promise<void>;
  /** 检查 cookies 是否存在 */
  hasSavedCookies(): boolean;
  /** 清除已保存的 cookies */
  clearCookies(): Promise<void>;
}

/**
 * 登录检测器接口
 */
export interface ILoginDetector {
  /** 检测当前登录状态 */
  checkLoginStatus(page: Page): Promise<LoginStatus>;
  /** 等待登录完成 */
  waitForLogin(page: Page, timeoutMs: number): Promise<boolean>;
  /** 触发登录流程 */
  triggerLogin(page: Page): Promise<void>;
}

/**
 * 浏览器管理器接口
 */
export interface IBrowserManager {
  /** 初始化浏览器 */
  initialize(headless: boolean): Promise<void>;
  /** 导航到指定 URL */
  navigateTo(url: string): Promise<void>;
  /** 获取当前页面 */
  getPage(): Page;
  /** 关闭浏览器 */
  close(): Promise<void>;
}

/**
 * 云原生开发启动器接口
 */
export interface ICloudDevLauncher {
  /** 导航到仓库页面 */
  navigateToRepo(page: Page, repoUrl: string): Promise<void>;
  /** 查找并点击云原生开发按钮 */
  launchCloudDev(page: Page): Promise<boolean>;
  /** 等待云原生开发环境启动 */
  waitForLaunch(page: Page): Promise<void>;
}

/**
 * 日志工具接口
 */
export interface ILogger {
  info(message: string): void;
  success(message: string): void;
  warning(message: string): void;
  error(message: string): void;
}
