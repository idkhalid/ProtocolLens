import { chromium, Browser, BrowserContext } from 'playwright';

const NAVIGATION_TIMEOUT_MS = 15000;

async function main() {
  const args = process.argv.slice(2);
  let url = '';
  let headed = false;
  let durationMs = 5000;
  let harPath = '';

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--headed') {
      headed = true;
    } else if (args[i] === '--duration') {
      durationMs = parseInt(args[++i], 10);
    } else if (args[i] === '--har') {
      harPath = args[++i];
    } else if (!args[i].startsWith('--')) {
      url = args[i];
    }
  }

  if (!url) {
    console.error('Error: missing capture URL');
    process.exitCode = 1;
    return;
  }

  if (!harPath) {
    console.error('Error: missing --har path');
    process.exitCode = 1;
    return;
  }

  try {
    const parsed = new URL(url);
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      console.error('Error: unsupported URL scheme. Use http or https.');
      process.exitCode = 1;
      return;
    }
  } catch (e) {
    console.error('Error: invalid URL format');
    process.exitCode = 1;
    return;
  }

  let browser: Browser | null = null;
  let context: BrowserContext | null = null;
  let shutdownPromise: Promise<void> | null = null;

  async function shutdown() {
    if (!shutdownPromise) {
      shutdownPromise = (async () => {
        if (context) {
          try {
            await context.close(); // finalizes HAR
          } catch (err) {
            console.error('Error closing context:', err);
          }
        }
        if (browser) {
          try {
            await browser.close();
          } catch (err) {
            console.error('Error closing browser:', err);
          }
        }
      })();
    }
    return shutdownPromise;
  }

  // Handle termination signals
  const handleSignal = async (signal: string) => {
    process.exitCode = 1;
    await shutdown();
    process.exit();
  };

  process.on('SIGINT', handleSignal);
  process.on('SIGTERM', handleSignal);
  process.on('SIGBREAK', handleSignal);

  let failed = false;
  try {
    browser = await chromium.launch({ headless: !headed });
    context = await browser.newContext({
      recordHar: {
        path: harPath,
      },
    });

    const page = await context.newPage();
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: NAVIGATION_TIMEOUT_MS });
    await page.waitForTimeout(durationMs);
  } catch (e: any) {
    failed = true;
    console.error(`Capture failed: ${e.message}`);
  } finally {
    await shutdown();
  }

  if (failed) {
    process.exitCode = 1;
  }
}

main();
