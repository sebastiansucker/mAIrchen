import { test, expect } from '@playwright/test';

test.describe('mAIrchen Story Generation', () => {
  test('should generate a random story', async ({ page }) => {
    // Navigate to the app
    await page.goto('http://localhost:80');
    
    // Wait for page to load
    await expect(page.locator('h1')).toContainText('mAIrchen');
    
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
});