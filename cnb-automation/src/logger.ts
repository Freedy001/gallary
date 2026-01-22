/**
 * CNB 浏览器自动化 - 日志器
 * 
 * 为不同的日志级别提供彩色控制台输出。
 * 需求: 1.4, 3.1, 4.3, 5.3, 5.4
 */

import chalk from 'chalk';
import type {ILogger} from './types/index.js';

/**
 * 格式化当前时间戳用于日志消息。
 * @returns 格式化的时间戳字符串 [HH:MM:SS]
 */
function getTimestamp(): string {
  const now = new Date();
  const hours = now.getHours().toString().padStart(2, '0');
  const minutes = now.getMinutes().toString().padStart(2, '0');
  const seconds = now.getSeconds().toString().padStart(2, '0');
  return `[${hours}:${minutes}:${seconds}]`;
}

/**
 * Logger 类提供带时间戳的彩色控制台输出，
 * 支持不同的日志级别（info、success、warning、error）。
 */
export class Logger implements ILogger {
  /**
   * 以蓝色输出信息日志。
   * @param message - 要输出的消息
   */
  info(message: string): void {
    const timestamp = chalk.gray(getTimestamp());
    const icon = chalk.blue('ℹ');
    const text = chalk.blue(message);
    console.log(`${timestamp} ${icon} ${text}`);
  }

  /**
   * 以绿色输出成功日志。
   * 需求 5.3: 点击成功时输出“云原生开发环境启动中...”
   * @param message - 要输出的消息
   */
  success(message: string): void {
    const timestamp = chalk.gray(getTimestamp());
    const icon = chalk.green('✔');
    const text = chalk.green(message);
    console.log(`${timestamp} ${icon} ${text}`);
  }

  /**
   * 以黄色输出警告日志。
   * 需求 3.1: 未登录时提示“请使用微信扫码登录”
   * @param message - 要输出的消息
   */
  warning(message: string): void {
    const timestamp = chalk.gray(getTimestamp());
    const icon = chalk.yellow('⚠');
    const text = chalk.yellow(message);
    console.log(`${timestamp} ${icon} ${text}`);
  }

  /**
   * 以红色输出错误日志。
   * 需求 1.4: 浏览器启动失败时输出错误信息
   * 需求 4.3: 仓库 URL 无效或返回 404 时输出错误信息
   * 需求 5.4: 未找到“云原生开发”按钮时输出错误信息
   * @param message - 要输出的消息
   */
  error(message: string): void {
    const timestamp = chalk.gray(getTimestamp());
    const icon = chalk.red('✖');
    const text = chalk.red(message);
    console.error(`${timestamp} ${icon} ${text}`);
  }
}

/**
 * 默认日志器实例，便于在整个应用程序中使用。
 */
export const logger = new Logger();
