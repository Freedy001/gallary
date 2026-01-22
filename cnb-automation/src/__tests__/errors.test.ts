/**
 * CNB Browser Automation - Error Classes Tests
 * 
 * 测试自定义错误类的正确性，包括：
 * - 错误消息格式
 * - 错误名称
 * - 自定义属性
 * - instanceof 检查
 */

import {describe, expect, it} from 'vitest';
import {BrowserLaunchError, CookieError, ElementNotFoundError, LoginTimeoutError, NavigationError,} from '../errors';

describe('BrowserLaunchError', () => {
  it('should create error with correct message format', () => {
    const error = new BrowserLaunchError('Chromium 未安装');
    
    expect(error.message).toBe('浏览器启动失败: Chromium 未安装');
    expect(error.name).toBe('BrowserLaunchError');
  });

  it('should store the cause error', () => {
    const cause = new Error('Original error');
    const error = new BrowserLaunchError('启动失败', cause);
    
    expect(error.cause).toBe(cause);
  });

  it('should work without cause', () => {
    const error = new BrowserLaunchError('未知错误');
    
    expect(error.cause).toBeUndefined();
  });

  it('should be instanceof Error and BrowserLaunchError', () => {
    const error = new BrowserLaunchError('test');
    
    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(BrowserLaunchError);
  });
});

describe('LoginTimeoutError', () => {
  it('should create error with correct message format for 5 minutes', () => {
    const error = new LoginTimeoutError(300000); // 5 minutes in ms
    
    expect(error.message).toBe('登录超时: 等待 300 秒后仍未检测到登录成功');
    expect(error.name).toBe('LoginTimeoutError');
  });

  it('should store the timeout value', () => {
    const error = new LoginTimeoutError(60000);
    
    expect(error.timeoutMs).toBe(60000);
  });

  it('should handle various timeout values', () => {
    const error1 = new LoginTimeoutError(1000);
    expect(error1.message).toBe('登录超时: 等待 1 秒后仍未检测到登录成功');

    const error2 = new LoginTimeoutError(120000);
    expect(error2.message).toBe('登录超时: 等待 120 秒后仍未检测到登录成功');
  });

  it('should be instanceof Error and LoginTimeoutError', () => {
    const error = new LoginTimeoutError(300000);
    
    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(LoginTimeoutError);
  });
});

describe('NavigationError', () => {
  it('should create error with URL only', () => {
    const error = new NavigationError('https://cnb.cool/user/repo');
    
    expect(error.message).toBe('导航失败: https://cnb.cool/user/repo');
    expect(error.name).toBe('NavigationError');
    expect(error.url).toBe('https://cnb.cool/user/repo');
    expect(error.statusCode).toBeUndefined();
  });

  it('should create error with URL and status code', () => {
    const error = new NavigationError('https://cnb.cool/user/repo', 404);
    
    expect(error.message).toBe('导航失败: https://cnb.cool/user/repo (HTTP 404)');
    expect(error.statusCode).toBe(404);
  });

  it('should handle various HTTP status codes', () => {
    const error500 = new NavigationError('https://example.com', 500);
    expect(error500.message).toBe('导航失败: https://example.com (HTTP 500)');

    const error403 = new NavigationError('https://example.com', 403);
    expect(error403.message).toBe('导航失败: https://example.com (HTTP 403)');
  });

  it('should be instanceof Error and NavigationError', () => {
    const error = new NavigationError('https://cnb.cool');
    
    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(NavigationError);
  });
});

describe('CookieError', () => {
  it('should create error for load operation', () => {
    const error = new CookieError('文件不存在', 'load');
    
    expect(error.message).toBe('Cookie 加载失败: 文件不存在');
    expect(error.name).toBe('CookieError');
    expect(error.operation).toBe('load');
  });

  it('should create error for save operation', () => {
    const error = new CookieError('磁盘空间不足', 'save');
    
    expect(error.message).toBe('Cookie 保存失败: 磁盘空间不足');
    expect(error.operation).toBe('save');
  });

  it('should handle various error messages', () => {
    const error1 = new CookieError('JSON 解析错误', 'load');
    expect(error1.message).toBe('Cookie 加载失败: JSON 解析错误');

    const error2 = new CookieError('权限被拒绝', 'save');
    expect(error2.message).toBe('Cookie 保存失败: 权限被拒绝');
  });

  it('should be instanceof Error and CookieError', () => {
    const error = new CookieError('test', 'load');
    
    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(CookieError);
  });
});

describe('ElementNotFoundError', () => {
  it('should create error with selector and description', () => {
    const error = new ElementNotFoundError(
      'button:has-text("云原生开发")',
      '云原生开发按钮'
    );
    
    expect(error.message).toBe(
      '未找到元素: 云原生开发按钮 (选择器: button:has-text("云原生开发"))'
    );
    expect(error.name).toBe('ElementNotFoundError');
    expect(error.selector).toBe('button:has-text("云原生开发")');
    expect(error.description).toBe('云原生开发按钮');
  });

  it('should handle various selectors', () => {
    const error1 = new ElementNotFoundError('text=微信登录', '微信登录按钮');
    expect(error1.message).toBe('未找到元素: 微信登录按钮 (选择器: text=微信登录)');

    const error2 = new ElementNotFoundError('.user-avatar', '用户头像');
    expect(error2.message).toBe('未找到元素: 用户头像 (选择器: .user-avatar)');
  });

  it('should be instanceof Error and ElementNotFoundError', () => {
    const error = new ElementNotFoundError('selector', 'description');
    
    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(ElementNotFoundError);
  });
});

describe('Error type discrimination', () => {
  it('should allow type discrimination in catch blocks', () => {
    const errors: Error[] = [
      new BrowserLaunchError('test'),
      new LoginTimeoutError(1000),
      new NavigationError('url'),
      new CookieError('msg', 'load'),
      new ElementNotFoundError('sel', 'desc'),
    ];

    const errorTypes = errors.map((error) => {
      if (error instanceof BrowserLaunchError) return 'browser';
      if (error instanceof LoginTimeoutError) return 'timeout';
      if (error instanceof NavigationError) return 'navigation';
      if (error instanceof CookieError) return 'cookie';
      if (error instanceof ElementNotFoundError) return 'element';
      return 'unknown';
    });

    expect(errorTypes).toEqual([
      'browser',
      'timeout',
      'navigation',
      'cookie',
      'element',
    ]);
  });
});
