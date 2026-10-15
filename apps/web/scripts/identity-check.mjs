import { chromium, expect } from '@playwright/test';
import { mkdir } from 'node:fs/promises';

const origin = process.env.WNT_IDENTITY_ORIGIN || 'http://127.0.0.1:23000';
const base = process.env.WNT_IDENTITY_BASE || '/wishlist';
const output = process.env.WNT_IDENTITY_OUTPUT || '../../work/changes/wnt-163-wishlist-identity/screenshots';
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/chromium', headless: true });
try {
  for (const [size, viewport] of Object.entries({ desktop: { width: 1440, height: 1000 }, mobile: { width: 390, height: 844 } })) {
    const page = await browser.newPage({ viewport });
    for (const [name, route] of [['board', base], ['board-slash', `${base}/`], ['privacy', `${base}/privacy`], ['admin', `${base}/admin`]]) {
      await page.goto(`${origin}${route}`);
      const brand = page.getByRole('link', { name: 'Wasting No Time Wishlist home', exact: true });
      await expect(brand).toBeVisible();
      await expect(brand).toHaveAttribute('href', `${base}/`);
      const maker = page.getByRole('link', { name: 'made by WNT — Wasting No Time home', exact: true });
      await expect(maker).toHaveAttribute('href', 'https://wastingnotime.org/');
      await maker.focus();
      await expect(maker).toBeFocused();
      await page.keyboard.press('Shift+Tab');
      await page.keyboard.press('Tab');
      await expect(maker).toBeFocused();
      expect(await maker.evaluate(el => getComputedStyle(el).outlineWidth)).toBe('2px');
      await page.route('https://wastingnotime.org/', route => route.fulfill({ body: '<title>Parent home</title>' }));
      await page.keyboard.press('Enter');
      await expect(page).toHaveURL('https://wastingnotime.org/');
      await page.goBack();
      for (const img of await page.locator('.brand img, .maker-signature img').all()) {
        expect(await img.evaluate(el => el.complete && el.naturalWidth > 0)).toBe(true);
      }
      const icon = await page.locator('link[rel="icon"][type="image/svg+xml"]').getAttribute('href');
      expect(icon).toBe(`${base}/identity/v1.0.0/favicon.svg`);
      for (const file of ['favicon.svg', 'favicon.ico', 'favicon-32.png', 'apple-touch-icon.png', 'wishlist-social.png']) {
        expect((await page.request.get(`${origin}${base}/identity/v1.0.0/${file}`)).ok()).toBe(true);
      }
      expect(await page.locator('meta[property="og:image"]').getAttribute('content')).toBe(`https://wastingnotime.org${base}/identity/v1.0.0/wishlist-social.png`);
      expect(await page.content()).not.toContain('Community Lab');
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await page.screenshot({ path: `${output}/${size}-${name}.png`, fullPage: true });
      if (size === 'desktop') {
        await page.locator('body').evaluate(el => { el.style.zoom = '2'; });
        await expect(maker).toBeVisible();
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      }
    }
    await page.close();
  }
  console.log('Identity, assets, base entries, keyboard navigation, zoom and screenshots passed.');
} finally { await browser.close(); }
