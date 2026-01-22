/**
 * CNB Browser Automation - Configuration Tests
 * 
 * 测试配置模块的正确性
 */

import {describe, expect, it} from 'vitest';
import {
  CNB_BASE_URL,
  DEFAULT_CONFIG,
  DEFAULT_ELEMENT_TIMEOUT,
  DEFAULT_LOAD_TIMEOUT,
  DEFAULT_LOGIN_SELECTORS,
  DEFAULT_NAVIGATION_TIMEOUT,
  DEFAULT_REPO_SELECTORS,
  getConfig,
  getLoginSelectors,
  getRepoSelectors
} from '../config.js';

describe('DEFAULT_CONFIG', () => {
  it('should have correct cookie file path', () => {
    expect(DEFAULT_CONFIG.cookieFilePath).toBe('~/.cnb-automation/cookies.json');
  });

  it('should have 5 minute default timeout', () => {
    expect(DEFAULT_CONFIG.defaultTimeout).toBe(300000);
  });

  it('should have 2 second check interval', () => {
    expect(DEFAULT_CONFIG.checkInterval).toBe(2000);
  });

  it('should have login selectors', () => {
    expect(DEFAULT_CONFIG.selectors.login).toBeDefined();
    expect(DEFAULT_CONFIG.selectors.login.wechatLoginButton).toBe('text=微信登录');
    expect(DEFAULT_CONFIG.selectors.login.userAvatar).toBe('.user-avatar, [class*="avatar"]');
    expect(DEFAULT_CONFIG.selectors.login.workbench).toBe('text=工作台');
  });

  it('should have repo selectors', () => {
    expect(DEFAULT_CONFIG.selectors.repo).toBeDefined();
    expect(DEFAULT_CONFIG.selectors.repo.cloudDevButton).toBe('button:has-text("云原生开发")');
    expect(DEFAULT_CONFIG.selectors.repo.forkButton).toBe('button:has-text("Fork")');
  });
});

describe('DEFAULT_LOGIN_SELECTORS', () => {
  it('should have wechat login button selector', () => {
    expect(DEFAULT_LOGIN_SELECTORS.wechatLoginButton).toBe('text=微信登录');
  });

  it('should have user avatar selector', () => {
    expect(DEFAULT_LOGIN_SELECTORS.userAvatar).toBe('.user-avatar, [class*="avatar"]');
  });

  it('should have workbench selector', () => {
    expect(DEFAULT_LOGIN_SELECTORS.workbench).toBe('text=工作台');
  });
});

describe('DEFAULT_REPO_SELECTORS', () => {
  it('should have cloud dev button selector', () => {
    expect(DEFAULT_REPO_SELECTORS.cloudDevButton).toBe('button:has-text("云原生开发")');
  });

  it('should have fork button selector', () => {
    expect(DEFAULT_REPO_SELECTORS.forkButton).toBe('button:has-text("Fork")');
  });
});

describe('Constants', () => {
  it('should have correct CNB base URL', () => {
    expect(CNB_BASE_URL).toBe('https://cnb.cool/');
  });

  it('should have correct navigation timeout', () => {
    expect(DEFAULT_NAVIGATION_TIMEOUT).toBe(30000);
  });

  it('should have correct load timeout', () => {
    expect(DEFAULT_LOAD_TIMEOUT).toBe(60000);
  });

  it('should have correct element timeout', () => {
    expect(DEFAULT_ELEMENT_TIMEOUT).toBe(10000);
  });
});

describe('getConfig', () => {
  it('should return default config when no overrides', () => {
    const config = getConfig();
    expect(config).toEqual(DEFAULT_CONFIG);
  });

  it('should return a copy, not the original', () => {
    const config = getConfig();
    expect(config).not.toBe(DEFAULT_CONFIG);
  });

  it('should override top-level properties', () => {
    const config = getConfig({
      cookieFilePath: '/custom/path/cookies.json',
      defaultTimeout: 600000
    });
    
    expect(config.cookieFilePath).toBe('/custom/path/cookies.json');
    expect(config.defaultTimeout).toBe(600000);
    expect(config.checkInterval).toBe(2000); // unchanged
  });

  it('should merge login selectors', () => {
    const config = getConfig({
      selectors: {
        login: {
          wechatLoginButton: 'button.custom-login',
          userAvatar: '.user-avatar, [class*="avatar"]',
          workbench: 'text=工作台'
        },
        repo: DEFAULT_REPO_SELECTORS
      }
    });
    
    expect(config.selectors.login.wechatLoginButton).toBe('button.custom-login');
    expect(config.selectors.login.userAvatar).toBe('.user-avatar, [class*="avatar"]');
  });

  it('should merge repo selectors', () => {
    const config = getConfig({
      selectors: {
        login: DEFAULT_LOGIN_SELECTORS,
        repo: {
          cloudDevButton: 'button.custom-cloud-dev',
          forkButton: 'button:has-text("Fork")'
        }
      }
    });
    
    expect(config.selectors.repo.cloudDevButton).toBe('button.custom-cloud-dev');
    expect(config.selectors.repo.forkButton).toBe('button:has-text("Fork")');
  });
});

describe('getLoginSelectors', () => {
  it('should return default selectors when no overrides', () => {
    const selectors = getLoginSelectors();
    expect(selectors).toEqual(DEFAULT_LOGIN_SELECTORS);
  });

  it('should override specific selectors', () => {
    const selectors = getLoginSelectors({
      wechatLoginButton: 'button.custom'
    });
    
    expect(selectors.wechatLoginButton).toBe('button.custom');
    expect(selectors.userAvatar).toBe('.user-avatar, [class*="avatar"]');
    expect(selectors.workbench).toBe('text=工作台');
  });
});

describe('getRepoSelectors', () => {
  it('should return default selectors when no overrides', () => {
    const selectors = getRepoSelectors();
    expect(selectors).toEqual(DEFAULT_REPO_SELECTORS);
  });

  it('should override specific selectors', () => {
    const selectors = getRepoSelectors({
      cloudDevButton: 'button.custom-cloud'
    });
    
    expect(selectors.cloudDevButton).toBe('button.custom-cloud');
    expect(selectors.forkButton).toBe('button:has-text("Fork")');
  });
});
