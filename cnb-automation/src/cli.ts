/**
 * CNB Browser Automation - CLI Command Parser
 * 
 * 使用 Commander.js 实现命令行参数解析
 * 
 * 需求:
 * - 7.1: 支持 --repo 参数指定目标仓库地址
 * - 7.2: 支持 --headless 参数控制是否显示浏览器窗口
 * - 7.3: 支持 --timeout 参数设置登录等待超时时间
 * - 7.4: 未提供必要参数时显示帮助信息
 */

import {Command} from 'commander';
import type {CLIOptions} from './types/index.js';

/** 默认超时时间（秒） */
const DEFAULT_TIMEOUT = 300;

/** 默认无头模式 */
const DEFAULT_HEADLESS = false;

/**
 * 创建并配置 Commander 程序实例
 * @returns 配置好的 Command 实例
 */
export function createProgram(): Command {
  const program = new Command();

  program
    .name('cnb-automation')
    .description('CLI tool for automating cnb.cool cloud native development platform')
    .version('1.0.0')
    .option(
      '-r, --repo <url>',
      'Target repository URL on cnb.cool (e.g., https://cnb.cool/user/repo)'
    )
    .option(
      '-H, --headless',
      'Run browser in headless mode (no visible window)',
      DEFAULT_HEADLESS
    )
    .option(
      '-t, --timeout <seconds>',
      'Login timeout in seconds',
      String(DEFAULT_TIMEOUT)
    )
    .option(
      '-c, --cookie-path <path>',
      'Path to cookie file for session persistence'
    );

  return program;
}

/**
 * 验证超时参数是否为有效的正整数
 * @param value 超时值字符串
 * @returns 解析后的数字
 * @throws 如果值无效
 */
function parseTimeout(value: string): number {
  const parsed = parseInt(value, 10);
  if (isNaN(parsed) || parsed <= 0) {
    throw new Error(`Invalid timeout value: "${value}". Must be a positive integer.`);
  }
  return parsed;
}

/**
 * 验证仓库 URL 格式
 * @param url 仓库 URL
 * @returns 是否有效
 */
function isValidRepoUrl(url: string): boolean {
  // 允许 cnb.cool 域名的 URL 或相对路径格式
  const cnbUrlPattern = /^https?:\/\/cnb\.cool\/.+/;
  const relativePathPattern = /^[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+/;
  return cnbUrlPattern.test(url) || relativePathPattern.test(url);
}

/**
 * 解析命令行参数并返回 CLIOptions
 * @param args 命令行参数数组（默认使用 process.argv）
 * @returns 解析后的 CLI 选项
 */
export function parseArgs(args?: string[]): CLIOptions {
  const program = createProgram();
  
  // 解析参数
  program.parse(args, { from: args ? 'user' : 'node' });
  
  const opts = program.opts();
  
  // 验证必要参数 - repo
  if (!opts.repo) {
    program.outputHelp();
    throw new Error('Missing required option: --repo <url>');
  }
  
  // 验证 repo URL 格式
  if (!isValidRepoUrl(opts.repo)) {
    throw new Error(
      `Invalid repository URL: "${opts.repo}". ` +
      'Expected format: https://cnb.cool/user/repo or user/repo'
    );
  }
  
  // 解析并验证 timeout
  const timeout = parseTimeout(opts.timeout);
  
  // 构建 CLIOptions
  const options: CLIOptions = {
    repo: opts.repo,
    headless: opts.headless === true,
    timeout,
    cookiePath: opts.cookiePath
  };
  
  return options;
}

/**
 * 显示帮助信息
 */
export function showHelp(): void {
  const program = createProgram();
  program.outputHelp();
}

/**
 * 获取程序版本
 */
export function getVersion(): string {
  return '1.0.0';
}
