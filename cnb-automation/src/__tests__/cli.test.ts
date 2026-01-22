/**
 * CNB Browser Automation - CLI Tests
 * 
 * 测试 CLI 命令行参数解析功能
 */

import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {createProgram, getVersion, parseArgs} from '../cli.js';

describe('CLI Command Parser', () => {
  // 保存原始的 console.error
  const originalConsoleError = console.error;
  
  beforeEach(() => {
    // 静默 commander 的错误输出
    console.error = vi.fn();
  });
  
  afterEach(() => {
    console.error = originalConsoleError;
  });

  describe('parseArgs', () => {
    describe('with valid arguments', () => {
      it('should parse --repo with full URL', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo'];
        const options = parseArgs(args);
        
        expect(options.repo).toBe('https://cnb.cool/user/repo');
      });

      it('should parse --repo with relative path format', () => {
        const args = ['--repo', 'user/repo'];
        const options = parseArgs(args);
        
        expect(options.repo).toBe('user/repo');
      });

      it('should parse short form -r for repo', () => {
        const args = ['-r', 'https://cnb.cool/user/repo'];
        const options = parseArgs(args);
        
        expect(options.repo).toBe('https://cnb.cool/user/repo');
      });

      it('should use default headless value (false)', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo'];
        const options = parseArgs(args);
        
        expect(options.headless).toBe(false);
      });

      it('should parse --headless flag', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo', '--headless'];
        const options = parseArgs(args);
        
        expect(options.headless).toBe(true);
      });

      it('should parse short form -H for headless', () => {
        const args = ['-r', 'https://cnb.cool/user/repo', '-H'];
        const options = parseArgs(args);
        
        expect(options.headless).toBe(true);
      });

      it('should use default timeout value (300 seconds)', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo'];
        const options = parseArgs(args);
        
        expect(options.timeout).toBe(300);
      });

      it('should parse --timeout with custom value', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo', '--timeout', '600'];
        const options = parseArgs(args);
        
        expect(options.timeout).toBe(600);
      });

      it('should parse short form -t for timeout', () => {
        const args = ['-r', 'https://cnb.cool/user/repo', '-t', '120'];
        const options = parseArgs(args);
        
        expect(options.timeout).toBe(120);
      });

      it('should parse --cookie-path option', () => {
        const args = [
          '--repo', 'https://cnb.cool/user/repo',
          '--cookie-path', '/path/to/cookies.json'
        ];
        const options = parseArgs(args);
        
        expect(options.cookiePath).toBe('/path/to/cookies.json');
      });

      it('should parse short form -c for cookie-path', () => {
        const args = [
          '-r', 'https://cnb.cool/user/repo',
          '-c', '~/.cnb/cookies.json'
        ];
        const options = parseArgs(args);
        
        expect(options.cookiePath).toBe('~/.cnb/cookies.json');
      });

      it('should have undefined cookiePath when not provided', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo'];
        const options = parseArgs(args);
        
        expect(options.cookiePath).toBeUndefined();
      });

      it('should parse all options together', () => {
        const args = [
          '--repo', 'https://cnb.cool/myorg/myrepo',
          '--headless',
          '--timeout', '180',
          '--cookie-path', '/tmp/cookies.json'
        ];
        const options = parseArgs(args);
        
        expect(options).toEqual({
          repo: 'https://cnb.cool/myorg/myrepo',
          headless: true,
          timeout: 180,
          cookiePath: '/tmp/cookies.json'
        });
      });
    });

    describe('with invalid arguments', () => {
      it('should throw error when --repo is missing', () => {
        const args: string[] = [];
        
        expect(() => parseArgs(args)).toThrow('Missing required option: --repo');
      });

      it('should throw error for invalid repo URL format', () => {
        const args = ['--repo', 'invalid-url'];
        
        expect(() => parseArgs(args)).toThrow('Invalid repository URL');
      });

      it('should throw error for invalid timeout value (non-numeric)', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo', '--timeout', 'abc'];
        
        expect(() => parseArgs(args)).toThrow('Invalid timeout value');
      });

      it('should throw error for invalid timeout value (zero)', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo', '--timeout', '0'];
        
        expect(() => parseArgs(args)).toThrow('Invalid timeout value');
      });

      it('should throw error for invalid timeout value (negative)', () => {
        const args = ['--repo', 'https://cnb.cool/user/repo', '--timeout', '-10'];
        
        expect(() => parseArgs(args)).toThrow('Invalid timeout value');
      });
    });
  });

  describe('createProgram', () => {
    it('should create a Command instance with correct name', () => {
      const program = createProgram();
      expect(program.name()).toBe('cnb-automation');
    });

    it('should have version set', () => {
      const program = createProgram();
      expect(program.version()).toBe('1.0.0');
    });
  });

  describe('getVersion', () => {
    it('should return version string', () => {
      expect(getVersion()).toBe('1.0.0');
    });
  });

  describe('CLIOptions type compliance', () => {
    it('should return object matching CLIOptions interface', () => {
      const args = ['--repo', 'https://cnb.cool/user/repo'];
      const options = parseArgs(args);
      
      // Type check - these should all be defined with correct types
      expect(typeof options.repo).toBe('string');
      expect(typeof options.headless).toBe('boolean');
      expect(typeof options.timeout).toBe('number');
      // cookiePath is optional
      expect(options.cookiePath === undefined || typeof options.cookiePath === 'string').toBe(true);
    });
  });
});
