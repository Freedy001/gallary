/**
 * CNB Browser Automation - Logger Tests
 * 
 * Tests for the Logger class that provides colored console output.
 * Requirements: 1.4, 3.1, 4.3, 5.3, 5.4
 */

import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {Logger, logger} from '../logger.js';

describe('Logger', () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    consoleErrorSpy.mockRestore();
  });

  describe('Logger class', () => {
    it('should create a Logger instance', () => {
      const loggerInstance = new Logger();
      expect(loggerInstance).toBeInstanceOf(Logger);
    });

    it('should have info, success, warning, and error methods', () => {
      const loggerInstance = new Logger();
      expect(typeof loggerInstance.info).toBe('function');
      expect(typeof loggerInstance.success).toBe('function');
      expect(typeof loggerInstance.warning).toBe('function');
      expect(typeof loggerInstance.error).toBe('function');
    });
  });

  describe('info()', () => {
    it('should log info message to console.log', () => {
      const loggerInstance = new Logger();
      loggerInstance.info('Test info message');
      
      expect(consoleSpy).toHaveBeenCalledTimes(1);
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('Test info message');
    });

    it('should include timestamp in info message', () => {
      const loggerInstance = new Logger();
      loggerInstance.info('Test message');
      
      const output = consoleSpy.mock.calls[0][0];
      // Timestamp format: [HH:MM:SS]
      expect(output).toMatch(/\[\d{2}:\d{2}:\d{2}\]/);
    });

    it('should include info icon (ℹ) in message', () => {
      const loggerInstance = new Logger();
      loggerInstance.info('Test message');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('ℹ');
    });
  });

  describe('success()', () => {
    it('should log success message to console.log', () => {
      const loggerInstance = new Logger();
      loggerInstance.success('Test success message');
      
      expect(consoleSpy).toHaveBeenCalledTimes(1);
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('Test success message');
    });

    it('should include timestamp in success message', () => {
      const loggerInstance = new Logger();
      loggerInstance.success('Test message');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toMatch(/\[\d{2}:\d{2}:\d{2}\]/);
    });

    it('should include success icon (✔) in message', () => {
      const loggerInstance = new Logger();
      loggerInstance.success('Test message');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('✔');
    });

    it('should log "云原生开发环境启动中..." correctly (Requirement 5.3)', () => {
      const loggerInstance = new Logger();
      loggerInstance.success('云原生开发环境启动中...');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('云原生开发环境启动中...');
    });
  });

  describe('warning()', () => {
    it('should log warning message to console.log', () => {
      const loggerInstance = new Logger();
      loggerInstance.warning('Test warning message');
      
      expect(consoleSpy).toHaveBeenCalledTimes(1);
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('Test warning message');
    });

    it('should include timestamp in warning message', () => {
      const loggerInstance = new Logger();
      loggerInstance.warning('Test message');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toMatch(/\[\d{2}:\d{2}:\d{2}\]/);
    });

    it('should include warning icon (⚠) in message', () => {
      const loggerInstance = new Logger();
      loggerInstance.warning('Test message');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('⚠');
    });

    it('should log "请使用微信扫码登录" correctly (Requirement 3.1)', () => {
      const loggerInstance = new Logger();
      loggerInstance.warning('请使用微信扫码登录');
      
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('请使用微信扫码登录');
    });
  });

  describe('error()', () => {
    it('should log error message to console.error', () => {
      const loggerInstance = new Logger();
      loggerInstance.error('Test error message');
      
      expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
      const output = consoleErrorSpy.mock.calls[0][0];
      expect(output).toContain('Test error message');
    });

    it('should include timestamp in error message', () => {
      const loggerInstance = new Logger();
      loggerInstance.error('Test message');
      
      const output = consoleErrorSpy.mock.calls[0][0];
      expect(output).toMatch(/\[\d{2}:\d{2}:\d{2}\]/);
    });

    it('should include error icon (✖) in message', () => {
      const loggerInstance = new Logger();
      loggerInstance.error('Test message');
      
      const output = consoleErrorSpy.mock.calls[0][0];
      expect(output).toContain('✖');
    });

    it('should log browser launch error correctly (Requirement 1.4)', () => {
      const loggerInstance = new Logger();
      loggerInstance.error('浏览器启动失败');
      
      const output = consoleErrorSpy.mock.calls[0][0];
      expect(output).toContain('浏览器启动失败');
    });

    it('should log invalid repo URL error correctly (Requirement 4.3)', () => {
      const loggerInstance = new Logger();
      loggerInstance.error('仓库地址无效或页面返回 404');
      
      const output = consoleErrorSpy.mock.calls[0][0];
      expect(output).toContain('仓库地址无效或页面返回 404');
    });

    it('should log cloud dev button not found error correctly (Requirement 5.4)', () => {
      const loggerInstance = new Logger();
      loggerInstance.error('未找到"云原生开发"按钮');
      
      const output = consoleErrorSpy.mock.calls[0][0];
      expect(output).toContain('未找到"云原生开发"按钮');
    });
  });

  describe('default logger instance', () => {
    it('should export a default logger instance', () => {
      expect(logger).toBeInstanceOf(Logger);
    });

    it('should have all required methods', () => {
      expect(typeof logger.info).toBe('function');
      expect(typeof logger.success).toBe('function');
      expect(typeof logger.warning).toBe('function');
      expect(typeof logger.error).toBe('function');
    });

    it('should work correctly when using the default instance', () => {
      logger.info('Default instance test');
      
      expect(consoleSpy).toHaveBeenCalledTimes(1);
      const output = consoleSpy.mock.calls[0][0];
      expect(output).toContain('Default instance test');
    });
  });
});
