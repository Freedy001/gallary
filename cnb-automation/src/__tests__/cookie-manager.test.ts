/**
 * CNB Browser Automation - Cookie Manager Property Tests
 * 
 * Feature: cnb-browser-automation
 * Property 1: Cookie 序列化往返一致性
 * 
 * **Validates: Requirements 3.4, 6.1**
 * 
 * For any valid Cookie array, serializing to file and deserializing back
 * should produce an equivalent Cookie array.
 */

import {afterEach, beforeEach, describe, it} from 'vitest';
import * as fc from 'fast-check';
import {CookieManager} from '../cookie-manager.js';
import type {Cookie} from '../types/index.js';
import {existsSync, mkdirSync, unlinkSync} from 'node:fs';
import {join} from 'node:path';
import {tmpdir} from 'node:os';

// Test directory for cookie files
const TEST_DIR = join(tmpdir(), 'cnb-automation-test');
const TEST_COOKIE_PATH = join(TEST_DIR, 'test-cookies.json');

/**
 * Arbitrary generator for valid Cookie objects
 */
const cookieArbitrary: fc.Arbitrary<Cookie> = fc.record({
  name: fc.string({ minLength: 1, maxLength: 50 }).filter(s => !s.includes('\n') && !s.includes('\r')),
  value: fc.string({ maxLength: 200 }).filter(s => !s.includes('\n') && !s.includes('\r')),
  domain: fc.string({ minLength: 1, maxLength: 100 }).filter(s => !s.includes('\n') && !s.includes('\r')),
  path: fc.constantFrom('/', '/api', '/auth', '/dashboard'),
  // Use future timestamps to avoid expiration filtering, or -1 for session cookies
  expires: fc.oneof(
    fc.constant(-1), // session cookie
    fc.integer({ min: Math.floor(Date.now() / 1000) + 3600, max: Math.floor(Date.now() / 1000) + 86400 * 365 }) // future expiry
  ),
  httpOnly: fc.boolean(),
  secure: fc.boolean(),
  sameSite: fc.constantFrom('Strict', 'Lax', 'None') as fc.Arbitrary<'Strict' | 'Lax' | 'None'>
});

describe('CookieManager Property Tests', () => {
  beforeEach(() => {
    // Ensure test directory exists
    if (!existsSync(TEST_DIR)) {
      mkdirSync(TEST_DIR, { recursive: true });
    }
    // Clean up any existing test cookie file
    if (existsSync(TEST_COOKIE_PATH)) {
      unlinkSync(TEST_COOKIE_PATH);
    }
  });

  afterEach(() => {
    // Clean up test cookie file
    if (existsSync(TEST_COOKIE_PATH)) {
      unlinkSync(TEST_COOKIE_PATH);
    }
  });

  /**
   * Feature: cnb-browser-automation, Property 1: Cookie 序列化往返一致性
   * 
   * For any valid Cookie array, serializing (saveCookies) and then
   * deserializing (loadCookies) should return an equivalent array.
   * 
   * **Validates: Requirements 3.4, 6.1**
   */
  it('Property 1: Cookie serialization round-trip consistency', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.array(cookieArbitrary, { minLength: 0, maxLength: 10 }),
        async (cookies) => {
          const manager = new CookieManager(TEST_COOKIE_PATH);

          // Save cookies
          await manager.saveCookies(cookies);

          // Load cookies back
          const loadedCookies = await manager.loadCookies();

          // Verify round-trip consistency
          if (cookies.length === 0) {
            // Empty array should load back as empty array
            return loadedCookies !== null && loadedCookies.length === 0;
          } else {
            if (loadedCookies === null) return false;
            if (loadedCookies.length !== cookies.length) return false;
            
            // Verify each cookie matches
            for (let i = 0; i < cookies.length; i++) {
              if (JSON.stringify(loadedCookies[i]) !== JSON.stringify(cookies[i])) {
                return false;
              }
            }
            return true;
          }
        }
      ),
      { numRuns: 100 }
    );
  });
});
