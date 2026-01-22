#!/usr/bin/env node
/**
 * CNB Browser Automation - Main Entry Point
 * 
 * 主入口文件，串联所有组件实现完整的自动化流程：
 * 初始化浏览器 → 检测登录 → 等待登录 → 导航仓库 → 启动云原生开发
 * 
 * 需求: 全部 (1.1-1.4, 2.1-2.4, 3.1-3.5, 4.1-4.3, 5.1-5.4, 6.1-6.4, 7.1-7.4)
 */

import {parseArgs} from './cli.js';
import {CookieManager} from './cookie-manager.js';
import {BrowserManager} from './browser-manager.js';
import {LoginDetector} from './login-detector.js';
import {CloudDevLauncher} from './cloud-dev-launcher.js';
import {type CLIOptions, LoginStatus} from './types';

/** CNB 平台基础 URL */
const CNB_BASE_URL = 'https://cnb.cool/';

/** 默认 Cookie 文件路径 */
const DEFAULT_COOKIE_PATH = '~/.cnb-automation/cookies.json';

/** 默认登录超时时间（毫秒）- 5分钟 */
const DEFAULT_LOGIN_TIMEOUT_MS = 300000;

/**
 * 日志输出工具函数
 */
const logger = {
  info: (message: string) => console.log(`[INFO] ${message}`),
  success: (message: string) => console.log(`[SUCCESS] ✓ ${message}`),
  warning: (message: string) => console.warn(`[WARNING] ⚠ ${message}`),
  error: (message: string) => console.error(`[ERROR] ✗ ${message}`)
};

/**
 * 规范化仓库 URL
 * 如果是相对路径格式 (user/repo)，转换为完整 URL
 * @param repoUrl 仓库 URL 或相对路径
 * @returns 完整的仓库 URL
 */
function normalizeRepoUrl(repoUrl: string): string {
  if (repoUrl.startsWith('http://') || repoUrl.startsWith('https://')) {
    return repoUrl;
  }
  // 相对路径格式，添加 CNB 基础 URL
  return `${CNB_BASE_URL}${repoUrl}`;
}

/**
 * 主流程函数
 * 
 * 实现完整的自动化流程：
 * 1. 解析 CLI 参数
 * 2. 初始化 CookieManager
 * 3. 初始化 BrowserManager
 * 4. 导航到 cnb.cool
 * 5. 检测登录状态
 * 6. 如果未登录：提示用户扫码 → 触发登录 → 等待登录 → 保存 cookies
 * 7. 导航到仓库页面
 * 8. 启动云原生开发
 * 9. 等待启动完成
 * 
 * @param options CLI 选项
 */
export async function main(options: CLIOptions): Promise<void> {
  // 初始化组件
  const cookiePath = options.cookiePath || DEFAULT_COOKIE_PATH;
  const cookieManager = new CookieManager(cookiePath);
  const browserManager = new BrowserManager(cookieManager);
  const loginDetector = new LoginDetector();
  const cloudDevLauncher = new CloudDevLauncher();

  // 计算超时时间（CLI 参数是秒，转换为毫秒）
  const loginTimeoutMs = options.timeout * 1000;

  try {
    // Step 1: 初始化浏览器
    // 需求 1.1: 启动可见的浏览器窗口
    // 需求 1.3: 加载已保存的 cookies
    // 需求 1.4: 启动失败时输出错误信息并退出
    logger.info('正在启动浏览器...');
    await browserManager.initialize(options.headless);
    logger.success('浏览器启动成功');

    // Step 2: 导航到 CNB 平台
    // 需求 1.2: 导航到 https://cnb.cool/
    logger.info(`正在导航到 ${CNB_BASE_URL}...`);
    await browserManager.navigateTo(CNB_BASE_URL);
    logger.success('已打开 CNB 平台');

    // 获取页面对象
    const page = browserManager.getPage();

    // Step 3: 检测登录状态
    // 需求 2.1-2.4: 检测页面登录状态
    logger.info('正在检测登录状态...');
    let loginStatus = await loginDetector.checkLoginStatus(page);
    
    // Step 4: 处理登录流程
    if (loginStatus === LoginStatus.NOT_LOGGED_IN) {
      // 需求 3.1: 提示用户扫码登录
      logger.warning('检测到未登录状态');
      logger.info('请使用微信扫码登录');
      
      // 需求 3.2: 点击微信登录按钮触发登录弹窗
      logger.info('正在触发微信登录...');
      await loginDetector.triggerLogin(page);
      logger.info('已弹出微信登录二维码，请扫码登录');
      
      // 需求 3.3: 每隔 2 秒检测一次登录状态
      // 需求 3.5: 等待超过指定时间仍未登录则提示超时并退出
      logger.info(`等待登录完成（超时时间: ${options.timeout} 秒）...`);
      const loginSuccess = await loginDetector.waitForLogin(page, loginTimeoutMs);
      
      if (!loginSuccess) {
        // 需求 3.5: 超时处理
        logger.error(`登录超时: 等待 ${options.timeout} 秒后仍未检测到登录成功`);
        logger.info('请检查是否已完成微信扫码，或增加 --timeout 参数值');
        throw new Error('登录超时');
      }
      
      // 需求 3.4: 登录成功后保存 cookies
      // 需求 6.1: 将 cookies 序列化并保存到本地文件
      logger.success('登录成功！');
      logger.info('正在保存登录状态...');
      const cookies = await browserManager.getCookies();
      await cookieManager.saveCookies(cookies);
      logger.success('登录状态已保存');
      
    } else if (loginStatus === LoginStatus.LOGGED_IN) {
      logger.success('检测到已登录状态');
    } else {
      // UNKNOWN 状态 - 可能页面还在加载
      logger.warning('无法确定登录状态，尝试继续...');
    }

    // Step 5: 导航到仓库页面
    // 需求 4.1: 导航到指定的仓库页面
    // 需求 4.2: 等待页面元素完全渲染
    // 需求 4.3: 仓库地址无效或 404 时输出错误信息
    const repoUrl = normalizeRepoUrl(options.repo);
    logger.info(`正在导航到仓库: ${repoUrl}...`);
    await cloudDevLauncher.navigateToRepo(page, repoUrl);
    logger.success('已打开仓库页面');

    // Step 6: 启动云原生开发环境
    // 需求 5.1: 定位橙色的"云原生开发"按钮
    // 需求 5.2: 点击该按钮
    // 需求 5.3: 输出"云原生开发环境启动中..."
    // 需求 5.4: 未找到按钮时输出错误信息
    logger.info('正在查找云原生开发按钮...');
    const launchSuccess = await cloudDevLauncher.launchCloudDev(page);
    
    if (!launchSuccess) {
      throw new Error('无法启动云原生开发环境');
    }

    // Step 7: 等待云原生开发环境启动
    logger.info('等待云原生开发环境启动...');
    await cloudDevLauncher.waitForLaunch(page);
    
    logger.success('云原生开发环境启动流程完成！');
    logger.info('浏览器窗口将保持打开状态，您可以继续操作');

    // 注意：不关闭浏览器，让用户继续使用
    // 如果需要自动关闭，可以取消下面的注释
    // await browserManager.close();

  } catch (error) {
    // 错误处理
    const errorMessage = error instanceof Error ? error.message : String(error);
    logger.error(`执行失败: ${errorMessage}`);
    
    // 发生错误时关闭浏览器
    await browserManager.close();
    
    // 重新抛出错误以便调用者处理
    throw error;
  }
}

/**
 * CLI 入口点
 * 解析命令行参数并执行主流程
 */
async function run(): Promise<void> {
  try {
    // 解析命令行参数
    // 需求 7.1-7.4: CLI 参数解析
    const options = parseArgs();
    
    // 执行主流程
    await main(options);
    
  } catch (error) {
    // 处理参数解析错误或执行错误
    const errorMessage = error instanceof Error ? error.message : String(error);
    
    // 如果是参数错误，parseArgs 已经显示了帮助信息
    if (!errorMessage.includes('Missing required option') && 
        !errorMessage.includes('Invalid')) {
      console.error(`\n错误: ${errorMessage}`);
    }
    
    // 退出进程
    process.exit(1);
  }
}

// 当作为主模块运行时执行
// 使用 ES 模块方式检测
const isMainModule = import.meta.url === `file://${process.argv[1]}`;
if (isMainModule) {
  run();
}

// 导出供测试使用
export { run, normalizeRepoUrl };
