import { test, expect } from '@playwright/test';

test.describe('mAIrchen Story Generation', () => {
  test('should generate a random story', async ({ page }) => {
    // Navigate to the app
    await page.goto('http://localhost:80');
    
    // Wait for page to load
    await expect(page.locator('header h1')).toContainText('mAIrchen');
    
    // Click the random button to fill all fields
    await page.click('#random-btn');
    
    // Wait a bit for random values to be filled
    await page.waitForTimeout(500);
    
    // Verify that fields are filled
    const themaValue = await page.locator('#thema').inputValue();
    expect(themaValue.length).toBeGreaterThan(0);
    
    // Set story length to 5 minutes by clicking the 5 Min button
    await page.click('button.length-btn[data-length="5"]');
    
    // Click the generate story button
    await page.click('#generate-btn');
    
    // Wait for the story display to be visible (max 90 seconds for longer story)
    await page.waitForSelector('#story-display', { state: 'visible', timeout: 90000 });

    // Story text streams in progressively (pen-reveal animation) - wait for
    // the stream to fully drain before reading title/content, otherwise we
    // might only capture partial text
    await page.waitForSelector('#story-display[data-stream-complete="true"]', { timeout: 90000 });

    // Verify story title is present
    const storyTitle = await page.locator('#story-title').textContent();
    expect(storyTitle.length).toBeGreaterThan(0);
    expect(storyTitle).not.toBe('');
    
    // Verify story content is present and long enough for 5 minutes
    const storyContent = await page.locator('#story-content').textContent();
    expect(storyContent.length).toBeGreaterThan(300); // 5 minute story should have at least 300 characters
    
    // Verify Grundwortschatz words are displayed
    const gwsInfo = await page.locator('#info-grundwortschatz').textContent();
    expect(gwsInfo.length).toBeGreaterThan(0);

    // Story text should be left-aligned (not justified) and hyphenate/wrap
    // long German compound words instead of overflowing narrow columns
    const storyTextStyle = await page.locator('#story-content').evaluate((el) => {
      const cs = getComputedStyle(el);
      return { textAlign: cs.textAlign, hyphens: cs.hyphens, overflowWrap: cs.overflowWrap };
    });
    expect(storyTextStyle.textAlign).toBe('left');
    expect(storyTextStyle.hyphens).toBe('auto');
    expect(storyTextStyle.overflowWrap).toBe('break-word');

    console.log(`Story generated successfully with title: "${storyTitle}"`);
    console.log(`Story length: ${storyContent.length} characters`);
    console.log(`Grundwortschatz info: ${gwsInfo}`);
  });

  test('should fill form manually and generate story', async ({ page }) => {
    // Navigate to the app
    await page.goto('http://localhost:80');
    
    // Fill in required fields manually
    await page.fill('#thema', 'Freundschaft');
    await page.fill('#personen', 'Ein kleiner Hase');
    await page.fill('#ort', 'im Wald');
    await page.fill('#stimmung', 'fröhlich');
    await page.fill('#stil', 'Astrid Lindgren');
    
    // Select 10 minute story
    await page.click('button.length-btn[data-length="10"]');
    
    // Click generate button
    await page.click('#generate-btn');
    
    // Wait for story to be generated
    await page.waitForSelector('#story-display', { state: 'visible', timeout: 90000 });

    // Story text streams in progressively - wait for the stream to fully
    // drain before reading title/content
    await page.waitForSelector('#story-display[data-stream-complete="true"]', { timeout: 90000 });

    // Verify story was created
    const storyTitle = await page.locator('#story-title').textContent();
    expect(storyTitle.length).toBeGreaterThan(0);
    
    const storyContent = await page.locator('#story-content').textContent();
    expect(storyContent.length).toBeGreaterThan(200);
    
    // Verify info fields are populated correctly
    const infoThema = await page.locator('#info-thema').textContent();
    expect(infoThema).toBe('Freundschaft');
    
    const infoPersonen = await page.locator('#info-personen').textContent();
    expect(infoPersonen).toBe('Ein kleiner Hase');
    
    console.log(`Manual form test passed with story: "${storyTitle}"`);
  });

  test('should show inline validation errors for empty required fields', async ({ page }) => {
    // Navigate to the app
    await page.goto('http://localhost:80');

    // Click generate without filling in any fields
    await page.click('#generate-btn');

    // No native dialog should appear; inline errors should be shown instead
    const themaError = page.locator('#thema-error');
    await expect(themaError).toBeVisible();
    await expect(themaError).not.toBeEmpty();

    // The first invalid field should be marked and focused
    const thema = page.locator('#thema');
    await expect(thema).toHaveAttribute('aria-invalid', 'true');
    await expect(thema).toBeFocused();

    // All other required fields should also be marked invalid
    for (const id of ['#personen-error', '#ort-error', '#stimmung-error']) {
      await expect(page.locator(id)).toBeVisible();
    }

    // The story form should still be visible (no navigation happened)
    await expect(page.locator('#input-form')).toBeVisible();
    await expect(page.locator('#story-display')).toBeHidden();

    // Filling in a field should clear its own error again
    await page.fill('#thema', 'Freundschaft');
    await expect(themaError).toBeHidden();
    await expect(thema).not.toHaveAttribute('aria-invalid', 'true');
  });
});

test.describe('mAIrchen Story Actions', () => {
  test('copy and print buttons let you copy/print the finished story', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);

    // window.print() would otherwise pop up an OS print dialog that blocks
    // the test; replace it with a spy so we can assert it was invoked
    await page.addInitScript(() => {
      window.__printCalled = false;
      window.print = () => { window.__printCalled = true; };
    });

    await page.goto('http://localhost:80');

    await page.fill('#thema', 'Freundschaft');
    await page.fill('#personen', 'Ein kleiner Hase');
    await page.fill('#ort', 'im Wald');
    await page.fill('#stimmung', 'fröhlich');
    await page.click('button.length-btn[data-length="5"]');
    await page.click('#generate-btn');

    await page.waitForSelector('#story-display', { state: 'visible', timeout: 90000 });
    await page.waitForSelector('#story-display[data-stream-complete="true"]', { timeout: 90000 });

    const storyTitle = await page.locator('#story-title').textContent();
    const storyContent = await page.locator('#story-content').textContent();

    // Copying puts title + text on the clipboard and shows brief feedback
    const copyBtn = page.locator('#copy-btn');
    await expect(copyBtn).toHaveAttribute('aria-label', 'Geschichte in die Zwischenablage kopieren');
    await copyBtn.click();
    await expect(copyBtn).toHaveClass(/copied/);
    await expect(copyBtn).toHaveAttribute('aria-label', 'In die Zwischenablage kopiert');

    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toContain(storyTitle);
    expect(clipboardText).toContain(storyContent.trim().slice(0, 50));

    // Feedback reverts back to the normal copy icon/label after a moment
    await expect(copyBtn).not.toHaveClass(/copied/, { timeout: 5000 });
    await expect(copyBtn).toHaveAttribute('aria-label', 'Geschichte in die Zwischenablage kopieren');

    // Printing calls window.print()
    await page.click('#print-btn');
    expect(await page.evaluate(() => window.__printCalled)).toBe(true);

    // The print stylesheet hides chrome (header buttons, cover, footer) and
    // force-expands the details grid, even though it was never opened
    await page.emulateMedia({ media: 'print' });
    await expect(page.locator('.book-header')).toBeHidden();
    await expect(page.locator('#story-cover')).toBeHidden();
    await expect(page.locator('footer')).toBeHidden();
    await expect(page.locator('#story-display .form-footer')).toBeHidden();
    await expect(page.locator('#story-title')).toBeVisible();
    await expect(page.locator('#story-content')).toBeVisible();
    await expect(page.locator('.story-details-body')).toBeVisible();
    await page.emulateMedia({ media: 'screen' });
  });
});

test.describe('mAIrchen Grundwortschatz Highlighting', () => {
  test('recognized Grundwortschatz words are highlighted in the story text', async ({ page }) => {
    await page.goto('http://localhost:80');

    await page.fill('#thema', 'Freundschaft');
    await page.fill('#personen', 'Ein kleiner Hase');
    await page.fill('#ort', 'im Wald');
    await page.fill('#stimmung', 'fröhlich');
    await page.click('button.length-btn[data-length="5"]');
    await page.click('#generate-btn');

    await page.waitForSelector('#story-display', { state: 'visible', timeout: 90000 });
    await page.waitForSelector('#story-display[data-stream-complete="true"]', { timeout: 90000 });

    const gwsInfo = await page.locator('#info-grundwortschatz').textContent();
    const gwsWords = gwsInfo.split(',').map(w => w.trim().toLowerCase()).filter(Boolean);
    expect(gwsWords.length).toBeGreaterThan(0);

    // Every word the backend reports as found must show up as a highlighted
    // <mark> in the story text - not just listed separately below it. The
    // backend also counts a word as found when it's a prefix of a longer
    // word in the text (regex `\bwort\w*\b`), so a highlight may be a
    // longer word than the reported one (e.g. "ab" found via "abends").
    const highlightedWords = (await page.locator('#story-content mark.gws-highlight').allTextContents())
      .map(w => w.toLowerCase());
    for (const word of gwsWords) {
      expect(
        highlightedWords.some(h => h.startsWith(word)),
        `"${word}" should be highlighted in the story text (found: ${JSON.stringify(highlightedWords)})`
      ).toBe(true);
    }

    // Highlighting must not leak <mark> markup into the copied plain text
    const storyTitle = await page.locator('#story-title').textContent();
    const copyBtn = page.locator('#copy-btn');
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
    await copyBtn.click();
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toContain(storyTitle);
    expect(clipboardText).not.toContain('<mark');
    expect(clipboardText).not.toContain('gws-highlight');
  });
});

test.describe('mAIrchen Accessibility', () => {
  test('grade and length buttons expose their selection state to assistive tech', async ({ page }) => {
    // Navigate to the app
    await page.goto('http://localhost:80');

    // Groups are marked as radiogroups, labelled by their existing text label
    await expect(page.locator('.grade-buttons')).toHaveAttribute('role', 'radiogroup');
    await expect(page.locator('.length-buttons')).toHaveAttribute('role', 'radiogroup');

    const gradeButtons = page.locator('.grade-btn');
    const lengthButtons = page.locator('.length-btn');

    // Every button is a radio and exposes aria-checked
    for (const buttons of [gradeButtons, lengthButtons]) {
      const count = await buttons.count();
      for (let i = 0; i < count; i++) {
        await expect(buttons.nth(i)).toHaveAttribute('role', 'radio');
      }
    }

    // Default selection (3./4. Klasse, 10 Min) is reflected in aria-checked
    await expect(page.locator('.grade-btn[data-grade="34"]')).toHaveAttribute('aria-checked', 'true');
    await expect(page.locator('.grade-btn[data-grade="12"]')).toHaveAttribute('aria-checked', 'false');
    await expect(page.locator('.length-btn[data-length="10"]')).toHaveAttribute('aria-checked', 'true');

    // Clicking a different grade button updates aria-checked on both buttons
    await page.click('.grade-btn[data-grade="12"]');
    await expect(page.locator('.grade-btn[data-grade="12"]')).toHaveAttribute('aria-checked', 'true');
    await expect(page.locator('.grade-btn[data-grade="34"]')).toHaveAttribute('aria-checked', 'false');

    // Clicking a different length button updates aria-checked on all three buttons
    await page.click('.length-btn[data-length="15"]');
    await expect(page.locator('.length-btn[data-length="15"]')).toHaveAttribute('aria-checked', 'true');
    await expect(page.locator('.length-btn[data-length="10"]')).toHaveAttribute('aria-checked', 'false');
    await expect(page.locator('.length-btn[data-length="5"]')).toHaveAttribute('aria-checked', 'false');
  });
});

test.describe('mAIrchen About Page', () => {
  test('reuses the shared design tokens instead of duplicated inline styles', async ({ page }) => {
    // Navigate straight to the About page
    await page.goto('http://localhost:80/about.html');

    // Styling now lives in the shared stylesheet, not an inline <style> block
    const inlineStyleCount = await page.locator('head style').count();
    expect(inlineStyleCount).toBe(0);

    // Section headings pick up the app's branded heading font/size instead
    // of falling back to the browser default h2
    const firstHeading = page.locator('.about-card h2').first();
    await expect(firstHeading).toBeVisible();
    const headingStyle = await firstHeading.evaluate((el) => {
      const computed = window.getComputedStyle(el);
      return { fontFamily: computed.fontFamily, fontSize: computed.fontSize };
    });
    expect(headingStyle.fontFamily).toContain('Quicksand');
    expect(headingStyle.fontSize).not.toBe('');

    // The back link returns to the main app
    await page.click('.back-link');
    await expect(page).toHaveURL(/index\.html$|\/$/);
    await expect(page.locator('header h1')).toContainText('mAIrchen');
  });
});
