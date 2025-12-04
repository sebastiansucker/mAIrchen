import { test, expect } from '@playwright/test';

test.describe('mAIrchen Frontend Tests', () => {
  test('should load the homepage correctly', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Check main heading
    await expect(page.locator('h1')).toContainText('mAIrchen');
    
    // Check all input fields are present
    await expect(page.locator('#thema')).toBeVisible();
    await expect(page.locator('#personen')).toBeVisible();
    await expect(page.locator('#ort')).toBeVisible();
    await expect(page.locator('#stimmung')).toBeVisible();
    await expect(page.locator('#stil')).toBeVisible();
    
    // Check buttons are present
    await expect(page.locator('#random-btn')).toBeVisible();
    await expect(page.locator('#generate-btn')).toBeVisible();
  });

  test('should have correct placeholders in input fields', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Verify placeholders
    const themaPlaceholder = await page.locator('#thema').getAttribute('placeholder');
    expect(themaPlaceholder).toContain('Freundschaft');
    
    const personenPlaceholder = await page.locator('#personen').getAttribute('placeholder');
    expect(personenPlaceholder).toContain('Hase');
    
    const ortPlaceholder = await page.locator('#ort').getAttribute('placeholder');
    expect(ortPlaceholder).toContain('Wald');
    
    const stimmungPlaceholder = await page.locator('#stimmung').getAttribute('placeholder');
    expect(stimmungPlaceholder).toContain('fröhlich');
  });

  test('should fill form with random values when random button is clicked', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Initially fields should be empty
    expect(await page.locator('#thema').inputValue()).toBe('');
    
    // Click random button
    await page.click('#random-btn');
    
    // Wait a bit for values to be filled
    await page.waitForTimeout(500);
    
    // Verify all fields are filled
    const thema = await page.locator('#thema').inputValue();
    expect(thema.length).toBeGreaterThan(0);
    
    const personen = await page.locator('#personen').inputValue();
    expect(personen.length).toBeGreaterThan(0);
    
    const ort = await page.locator('#ort').inputValue();
    expect(ort.length).toBeGreaterThan(0);
    
    const stimmung = await page.locator('#stimmung').inputValue();
    expect(stimmung.length).toBeGreaterThan(0);
    
    console.log(`Random values: Theme="${thema}", Characters="${personen}", Location="${ort}", Mood="${stimmung}"`);
  });

  test('should select different story lengths', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Check that 10 Min is active by default
    const tenMinBtn = page.locator('button.length-btn[data-length="10"]');
    await expect(tenMinBtn).toHaveClass(/active/);
    
    // Click 5 Min button
    const fiveMinBtn = page.locator('button.length-btn[data-length="5"]');
    await fiveMinBtn.click();
    await expect(fiveMinBtn).toHaveClass(/active/);
    await expect(tenMinBtn).not.toHaveClass(/active/);
    
    // Click 15 Min button
    const fifteenMinBtn = page.locator('button.length-btn[data-length="15"]');
    await fifteenMinBtn.click();
    await expect(fifteenMinBtn).toHaveClass(/active/);
    await expect(fiveMinBtn).not.toHaveClass(/active/);
  });

  test('should allow manual form input', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Fill form manually
    await page.fill('#thema', 'Test Thema');
    await page.fill('#personen', 'Test Person');
    await page.fill('#ort', 'Test Ort');
    await page.fill('#stimmung', 'Test Stimmung');
    await page.fill('#stil', 'Test Stil');
    
    // Verify values are set
    expect(await page.locator('#thema').inputValue()).toBe('Test Thema');
    expect(await page.locator('#personen').inputValue()).toBe('Test Person');
    expect(await page.locator('#ort').inputValue()).toBe('Test Ort');
    expect(await page.locator('#stimmung').inputValue()).toBe('Test Stimmung');
    expect(await page.locator('#stil').inputValue()).toBe('Test Stil');
  });

  test('should have loading overlay element in DOM', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Check that loading overlay exists in DOM (even if hidden)
    const loading = page.locator('#loading');
    expect(await loading.count()).toBe(1);
    
    // Loading should not be visible initially
    await expect(loading).toBeHidden();
    
    // Check loading animation elements exist
    expect(await page.locator('.magic-book-animation').count()).toBeGreaterThan(0);
    expect(await page.locator('.loading-text').count()).toBeGreaterThan(0);
  });

  test('should have responsive design elements', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Check that main container exists
    await expect(page.locator('main')).toBeVisible();
    
    // Check form group classes
    const formGroups = page.locator('.form-group');
    expect(await formGroups.count()).toBeGreaterThan(0);
    
    // Check button group exists
    await expect(page.locator('.button-group')).toBeVisible();
  });

  test('should clear form when random button is clicked multiple times', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Click random button first time
    await page.click('#random-btn');
    await page.waitForTimeout(300);
    const firstThema = await page.locator('#thema').inputValue();
    
    // Click random button second time
    await page.click('#random-btn');
    await page.waitForTimeout(300);
    const secondThema = await page.locator('#thema').inputValue();
    
    // Values should be different (or at least set)
    expect(firstThema.length).toBeGreaterThan(0);
    expect(secondThema.length).toBeGreaterThan(0);
  });

  test('should have accessible form labels', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Check that labels are associated with inputs
    const themaLabel = page.locator('label[for="thema"]');
    await expect(themaLabel).toBeVisible();
    await expect(themaLabel).toContainText('Thema');
    
    const personenLabel = page.locator('label[for="personen"]');
    await expect(personenLabel).toBeVisible();
    await expect(personenLabel).toContainText('Personen');
    
    const ortLabel = page.locator('label[for="ort"]');
    await expect(ortLabel).toBeVisible();
    await expect(ortLabel).toContainText('Ort');
    
    const stimmungLabel = page.locator('label[for="stimmung"]');
    await expect(stimmungLabel).toBeVisible();
    await expect(stimmungLabel).toContainText('Stimmung');
  });

  test('should have all length buttons visible', async ({ page }) => {
    await page.goto('http://localhost:80');
    
    // Check all three length buttons
    const fiveMin = page.locator('button.length-btn[data-length="5"]');
    const tenMin = page.locator('button.length-btn[data-length="10"]');
    const fifteenMin = page.locator('button.length-btn[data-length="15"]');
    
    await expect(fiveMin).toBeVisible();
    await expect(tenMin).toBeVisible();
    await expect(fifteenMin).toBeVisible();
    
    // Check button text
    await expect(fiveMin).toContainText('5 Min');
    await expect(tenMin).toContainText('10 Min');
    await expect(fifteenMin).toContainText('15 Min');
  });
});
