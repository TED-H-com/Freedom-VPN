// TEST ONLY: bounded browser probe. Session cookies arrive on stdin and are
// never printed, traced or persisted. Requests are confined to the lab origin.
import { chromium } from '../../apps/portal/node_modules/playwright/index.mjs';
let input = '';
for await (const chunk of process.stdin) input += chunk;
const { port, cookies } = JSON.parse(input);
if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('lab port');
const origin = 'https://portal.example.invalid:8443';
const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server', `--host-resolver-rules=MAP portal.example.invalid:8443 127.0.0.1:${port}, MAP * ~NOTFOUND`] });
try {
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  await context.addCookies(cookies.map(({ Name, Value }) => ({ name: Name, value: Value, url: origin, secure: true, httpOnly: true, sameSite: 'Lax' })));
  await context.route('**/*', route => route.request().url().startsWith(`${origin}/`) ? route.continue() : route.abort());
  const page = await context.newPage();
  const response = await page.goto(`${origin}/billing/return`, { timeout: 15000 });
  await page.getByTestId('member-billing-page').waitFor({ timeout: 15000 });
  const error = await page.getByTestId('member-billing-error').count();
  const pending = await page.getByTestId('checkout-return-pending-banner').count();
  console.log(JSON.stringify({ http_status: response.status(), billing_page_visible: true, api_error: error > 0, pending_confirmation: pending > 0 }));
  if (response.status() !== 200 || error > 0) process.exitCode = 1;
} finally { await browser.close(); }
