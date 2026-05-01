const puppeteer = require('puppeteer');
const fs = require('fs');

async function delay(time) {
  return new Promise(function(resolve) { 
      setTimeout(resolve, time)
  });
}

const config = {
  baseURL: 'http://127.0.0.1:4200',
  users: [
    { role: 'SYSTEM_ADMIN', phone: '+254700000000', password: 'password123' },
    { role: 'SACCO_ADMIN', phone: '+254711111111', password: 'password123' },
    { role: 'CREW', phone: '+254722000000', password: 'password123' }
  ],
  views: {
    SYSTEM_ADMIN: [
      '/dashboard', '/crew', '/assignments', '/earnings', '/wallets', 
      '/saccos', '/vehicles', '/routes', '/payroll', '/statutory-rates', 
      '/loans', '/credit', '/insurance', '/notifications', '/documents', '/admin'
    ],
    SACCO_ADMIN: [
      '/dashboard', '/crew', '/assignments', '/earnings', '/wallets', 
      '/saccos', '/vehicles', '/routes', '/payroll', '/loans', '/credit', 
      '/insurance', '/notifications', '/documents'
    ],
    CREW: [
      '/dashboard', '/profile', '/earnings', '/wallets', '/loans', 
      '/credit', '/insurance', '/notifications'
    ]
  }
};

const errorsFound = [];

async function login(page, phone, password) {
  await page.goto(`${config.baseURL}/auth/login`);
  await page.waitForSelector('input[name="phone"]', { timeout: 10000 });
  await page.type('input[name="phone"]', phone);
  await page.type('input[name="password"]', password);
  
  await Promise.all([
    page.click('button[type="submit"]'),
    page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 15000 }).catch(e => {
        // Sometimes it just changes route without full navigation event
    })
  ]);
  
  await delay(2000); // Wait for UI to settle
  const url = page.url();
  if (url.includes('/auth/login')) {
    throw new Error('Login failed. Still on login page.');
  }
}

async function logout(page) {
    // try to find logout button
    try {
        // Click on profile/user menu to reveal logout
        await page.click('.profile-btn, .user-menu-btn, button:has-text("Logout"), a:has-text("Logout")');
        await delay(500);
        const logoutBtn = await page.$('button:has-text("Logout"), a:has-text("Logout")');
        if (logoutBtn) {
            await logoutBtn.click();
            await delay(2000);
        } else {
            // clear localStorage manually
            await page.evaluate(() => {
                localStorage.clear();
            });
            await page.goto(`${config.baseURL}/auth/login`);
        }
    } catch(e) {
        await page.evaluate(() => {
            localStorage.clear();
        });
        await page.goto(`${config.baseURL}/auth/login`);
    }
}

async function testViews() {
  const browser = await puppeteer.launch({ 
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const page = await browser.newPage();

  page.on('console', msg => {
    if (msg.type() === 'error') {
      const location = msg.location();
      errorsFound.push({ type: 'CONSOLE_ERROR', text: msg.text(), url: location.url || 'unknown' });
    }
  });

  page.on('pageerror', err => {
    errorsFound.push({ type: 'PAGE_ERROR', text: err.toString() });
  });

  for (const user of config.users) {
    console.log(`\n--- Testing Role: ${user.role} ---`);
    try {
      await page.goto(`${config.baseURL}/auth/login`);
      await page.evaluate(() => localStorage.clear());
      await login(page, user.phone, user.password);
      console.log(`[${user.role}] Successfully logged in.`);
      
      const viewsToTest = config.views[user.role];
      for (const view of viewsToTest) {
        console.log(`[${user.role}] Testing view: ${view}`);
        try {
          const response = await page.goto(`${config.baseURL}${view}`, { waitUntil: 'networkidle2', timeout: 15000 });
          await delay(2000);
          
          // Look for any toast errors or alert elements
          const errorToasts = await page.$$eval('.toast-error, .alert-danger, snack-bar-container.mat-snack-bar-handset-fallback', els => els.map(e => e.innerText));
          if (errorToasts.length > 0) {
              errorsFound.push({ role: user.role, view: view, type: 'UI_ERROR', text: errorToasts.join(' | ') });
              console.log(`[${user.role}] UI Error found on ${view}:`, errorToasts);
          }
          
          // Specific CRUD test for a few important views (just quick checks, looking for create buttons)
          if (view === '/crew') {
              // try to open create modal
              const addBtn = await page.$('#btn-add-crew');
              if (addBtn) {
                  await addBtn.click();
                  await delay(1000);
                  // check if modal opened
                  const modal = await page.$('.modal-content, mat-dialog-container');
                  if (!modal) {
                     errorsFound.push({ role: user.role, view: view, type: 'CRUD_ERROR', text: 'Create modal did not open' });
                  } else {
                     // close modal
                     await page.keyboard.press('Escape');
                  }
              }
          }

        } catch (e) {
          errorsFound.push({ role: user.role, view: view, type: 'NAVIGATION_ERROR', text: e.message });
          console.log(`[${user.role}] Navigation error on ${view}:`, e.message);
        }
      }
      
      await logout(page);
    } catch (e) {
        errorsFound.push({ role: user.role, view: 'LOGIN', type: 'LOGIN_ERROR', text: e.message });
        console.log(`[${user.role}] Login failed:`, e.message);
    }
  }

  await browser.close();
  
  fs.writeFileSync('e2e_results.json', JSON.stringify(errorsFound, null, 2));
  console.log('\n--- Testing Complete ---');
  console.log(`Errors found: ${errorsFound.length}`);
}

testViews().catch(console.error);
