/**
 * CNB Browser Automation - Main Entry Point Tests
 * 
 * Tests for the main flow logic in index.ts
 */

import {describe, expect, it} from 'vitest';
import {normalizeRepoUrl} from '../index.js';

describe('normalizeRepoUrl', () => {
  describe('完整 URL 处理', () => {
    it('应该保持 https URL 不变', () => {
      const url = 'https://cnb.cool/user/repo';
      expect(normalizeRepoUrl(url)).toBe(url);
    });

    it('应该保持 http URL 不变', () => {
      const url = 'http://cnb.cool/user/repo';
      expect(normalizeRepoUrl(url)).toBe(url);
    });

    it('应该保持带路径的完整 URL 不变', () => {
      const url = 'https://cnb.cool/org/project/tree/main';
      expect(normalizeRepoUrl(url)).toBe(url);
    });
  });

  describe('相对路径处理', () => {
    it('应该将 user/repo 格式转换为完整 URL', () => {
      const path = 'user/repo';
      expect(normalizeRepoUrl(path)).toBe('https://cnb.cool/user/repo');
    });

    it('应该将 org/project 格式转换为完整 URL', () => {
      const path = 'my-org/my-project';
      expect(normalizeRepoUrl(path)).toBe('https://cnb.cool/my-org/my-project');
    });

    it('应该处理带下划线的路径', () => {
      const path = 'user_name/repo_name';
      expect(normalizeRepoUrl(path)).toBe('https://cnb.cool/user_name/repo_name');
    });

    it('应该处理带数字的路径', () => {
      const path = 'user123/repo456';
      expect(normalizeRepoUrl(path)).toBe('https://cnb.cool/user123/repo456');
    });
  });
});

describe('main 函数', () => {
  // 注意：main 函数的完整测试需要模拟浏览器，这里只测试导出
  it('应该导出 main 函数', async () => {
    const { main } = await import('../index.js');
    expect(typeof main).toBe('function');
  });

  it('应该导出 run 函数', async () => {
    const { run } = await import('../index.js');
    expect(typeof run).toBe('function');
  });

  it('应该导出 normalizeRepoUrl 函数', async () => {
    const { normalizeRepoUrl } = await import('../index.js');
    expect(typeof normalizeRepoUrl).toBe('function');
  });
});
