// TEST ONLY: bounded browser probe. Session cookies arrive on stdin and are
// never printed, traced or persisted. Requests are confined to the lab origin.
//
// Success prints {http_status, billing_page_visible, api_error,
// pending_confirmation}. Any failure prints {probe_error: {stage, message}}
// and exits 1, so a record names the failing step (for example a missing
// Playwright browser at stage "launch"). The message is one bounded line with
// cookie values, the worktree and the home directory removed. A page that
// does not render is still a failure.
import { homedir } from 'node:os';
import { fileURLToPath } from 'node:url';

const worktree = fileURLToPath(new URL('../../', import.meta.url)).replace(/\/$/, '');
const secrets = [];
function scrub(error) {
  let text = String(error && error.message ? error.message : error);
  for (const s of secrets) if (s) text = text.split(s).join('[redacted]');
  text = text.split(worktree).join('<worktree>');
  const home = homedir();
  if (home && home !== '/') text = text.split(home).join('<home>');
  const line = text.split('\n').map(l => l.trim()).find(l => l !== '') ?? '';
  return line.length > 400 ? `${line.slice(0, 400)}…` : line;
}

let stage = 'input';
let browser;
try {
  let input = '';
  for await (const chunk of process.stdin) input += chunk;
  const { port, cookies } = JSON.parse(input);
  for (const { Value } of cookies) secrets.push(Value);
  if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('lab port');
  stage = 'import';
  const { chromium } = await import('../../apps/portal/node_modules/playwright/index.mjs');
  const origin = 'https://portal.example.invalid:8443';
  stage = 'launch';
  browser = await chromium.launch({ headless: true, args: ['--no-proxy-server', `--host-resolver-rules=MAP portal.example.invalid:8443 127.0.0.1:${port}, MAP * ~NOTFOUND`] });
  stage = 'context';
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  await context.addCookies(cookies.map(({ Name, Value }) => ({ name: Name, value: Value, url: origin, secure: true, httpOnly: true, sameSite: 'Lax' })));
  await context.route('**/*', route => route.request().url().startsWith(`${origin}/`) ? route.continue() : route.abort());
  const page = await context.newPage();
  stage = 'navigate';
  const response = await page.goto(`${origin}/billing/return`, { timeout: 15000 });
  stage = 'render';
  await page.getByTestId('member-billing-page').waitFor({ timeout: 15000 });
  const error = await page.getByTestId('member-billing-error').count();
  const pending = await page.getByTestId('checkout-return-pending-banner').count();
  console.log(JSON.stringify({ http_status: response.status(), billing_page_visible: true, api_error: error > 0, pending_confirmation: pending > 0 }));
  if (response.status() !== 200 || error > 0) process.exitCode = 1;
} catch (error) {
  console.log(JSON.stringify({ probe_error: { stage, message: scrub(error) } }));
  process.exitCode = 1;
} finally {
  if (browser) await browser.close();
}
