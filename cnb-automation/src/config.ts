/**
 * CNB Browser Automation - Configuration
 * 
 * 定义默认配置和选择器，提供配置管理功能。
 * 需求: 全部
 */

import type {Config, LoginSelectors, RepoPageSelectors} from './types/index.js';

/**
 * 默认登录选择器
 */
export const DEFAULT_LOGIN_SELECTORS: LoginSelectors = {
  wechatLoginButton: 'text=微信登录',
  userAvatar: '.user-avatar, [class*="avatar"]',
  workbench: 'text=工作台'
};

/**
 * 默认仓库页面选择器
 */
export const DEFAULT_REPO_SELECTORS: RepoPageSelectors = {
  cloudDevButton: 'button:has-text("云原生开发")',
  forkButton: 'button:has-text("Fork")'
};

/**
 * 默认配置
 */
export const DEFAULT_CONFIG: Config = {
  cookieFilePath: '~/.cnb-automation/cookies.json',
  defaultTimeout: 300000,     // 5 分钟
  checkInterval: 2000,        // 2 秒
  selectors: {
    login: DEFAULT_LOGIN_SELECTORS,
    repo: DEFAULT_REPO_SELECTORS
  }
};

/**
 * CNB 平台基础 URL
 */
export const CNB_BASE_URL = 'https://cnb.cool/';

/**
 * 默认导航超时时间（毫秒）
 */
export const DEFAULT_NAVIGATION_TIMEOUT = 30000;

/**
 * 默认页面加载超时时间（毫秒）
 */
export const DEFAULT_LOAD_TIMEOUT = 60000;

/**
 * 默认元素等待超时时间（毫秒）
 */
export const DEFAULT_ELEMENT_TIMEOUT = 10000;

/**
 * 获取配置，支持部分覆盖
 * @param overrides 要覆盖的配置项
 * @returns 合并后的配置
 */
export function getConfig(overrides?: Partial<Config>): Config {
  if (!overrides) {
    return { ...DEFAULT_CONFIG };
  }

  return {
    ...DEFAULT_CONFIG,
    ...overrides,
    selectors: {
      login: {
        ...DEFAULT_CONFIG.selectors.login,
        ...overrides.selectors?.login
      },
      repo: {
        ...DEFAULT_CONFIG.selectors.repo,
        ...overrides.selectors?.repo
      }
    }
  };
}

/**
 * 获取登录选择器，支持部分覆盖
 * @param overrides 要覆盖的选择器
 * @returns 合并后的登录选择器
 */
export function getLoginSelectors(overrides?: Partial<LoginSelectors>): LoginSelectors {
  return {
    ...DEFAULT_LOGIN_SELECTORS,
    ...overrides
  };
}

/**
 * 获取仓库页面选择器，支持部分覆盖
 * @param overrides 要覆盖的选择器
 * @returns 合并后的仓库页面选择器
 */
export function getRepoSelectors(overrides?: Partial<RepoPageSelectors>): RepoPageSelectors {
  return {
    ...DEFAULT_REPO_SELECTORS,
    ...overrides
  };
}
