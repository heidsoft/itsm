import { test, expect, type Page, type BrowserContext } from '@playwright/test';

const BASE_URL = 'http://localhost';
const SCREENSHOT_DIR = 'test-results/workflow-test';

// Helper: take screenshot with descriptive name
async function snap(page: Page, name: string) {
  await page.screenshot({ path: `${SCREENSHOT_DIR}/${name}.png`, fullPage: true });
}

// Helper: wait and stabilize
async function settle(ms = 1500) {
  return new Promise(r => setTimeout(r, ms));
}

// Store auth state for reuse
let authCookie: string | undefined;

async function login(page: Page) {
  await page.goto(`${BASE_URL}/login`);
  await page.waitForLoadState('networkidle');
  
  // Fill login form
  await page.fill('input[id="username"], input[placeholder*="用户名"], input[type="text"]', 'admin');
  await page.fill('input[id="password"], input[type="password"]', 'admin123');
  
  // Click login button
  await page.click('button[type="submit"], button:has-text("登录")');
  
  // Wait for redirect after login
  await page.waitForURL('**/dashboard**', { timeout: 15000 });
  await settle(2000);
}

test.describe('工作流节点配置测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('1. 工作流列表页面', async ({ page }) => {
    // Navigate to workflow page
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    await snap(page, '01-workflow-list-page');
    
    // Check page content
    const pageText = await page.textContent('body');
    console.log('[TEST] Workflow page loaded, URL:', page.url());
    
    // Check for workflow list items
    const rows = await page.locator('table tbody tr, .ant-table-row').count();
    console.log(`[TEST] Found ${rows} workflow rows in table`);
    
    // Check for buttons
    const buttons = await page.locator('button').allTextContents();
    console.log('[TEST] Buttons found:', buttons.join(' | '));
    
    // Report any console errors
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
  });

  test('2. 创建新工作流 - 打开设计器', async ({ page }) => {
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    // Try to click "新建" or "创建" button
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建"), button:has-text("New")').first();
    const btnExists = await createBtn.count() > 0;
    
    if (btnExists) {
      await createBtn.click();
      await settle(2000);
      await snap(page, '02-create-workflow-modal');
      
      // Fill in workflow name
      const nameInput = page.locator('input[id*="name"], input[placeholder*="名称"], .ant-modal input').first();
      if (await nameInput.count() > 0) {
        await nameInput.fill('测试工作流-节点配置验证');
      }
      
      // Click confirm/create
      const confirmBtn = page.locator('.ant-modal button:has-text("确定"), .ant-modal button:has-text("创建"), .ant-modal-button-primary').first();
      if (await confirmBtn.count() > 0) {
        await confirmBtn.click();
        await settle(3000);
      }
    } else {
      // Try clicking on an existing workflow
      const firstRow = page.locator('table tbody tr:first-child a, table tbody tr:first-child td:first-child').first();
      if (await firstRow.count() > 0) {
        await firstRow.click();
        await settle(3000);
      } else {
        // Try direct navigation
        await page.goto(`${BASE_URL}/workflow/new`);
        await settle(3000);
      }
    }
    
    await snap(page, '02-workflow-designer-opened');
    console.log('[TEST] Designer page URL:', page.url());
  });

  test('3. BPMN 设计器 - 拖拽用户任务节点', async ({ page }) => {
    // First navigate to workflow and open/create one
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    // Try to open existing workflow or create new
    const createBtn = page.locator('button:has-text("新建"), button:has-text("创建")').first();
    if (await createBtn.count() > 0) {
      await createBtn.click();
      await settle(1500);
      
      const nameInput = page.locator('input[id*="name"], input[placeholder*="名称"], .ant-modal input').first();
      if (await nameInput.count() > 0) {
        await nameInput.fill('节点配置验证工作流');
      }
      
      const confirmBtn = page.locator('.ant-modal button:has-text("确定"), .ant-modal-button-primary').first();
      if (await confirmBtn.count() > 0) {
        await confirmBtn.click();
        await settle(3000);
      }
    } else {
      const firstRow = page.locator('table tbody tr:first-child a').first();
      if (await firstRow.count() > 0) {
        await firstRow.click();
        await settle(3000);
      }
    }
    
    // Look for BPMN designer canvas
    const canvas = page.locator('.bjs-container, .bpmn-designer, canvas, .djs-container, [class*="bpmn"], [class*="designer"]').first();
    await snap(page, '03-bpmn-designer-loaded');
    
    // Look for node palette - common selectors
    const paletteItems = await page.locator('.bjs-palette, .djs-palette, [class*="palette"], [class*="Palette"], .bpmn-palette').count();
    console.log(`[TEST] Palette elements found: ${paletteItems}`);
    
    // Try to find User Task in palette
    const userTaskPalette = page.locator('[data-action*="user"], [title*="用户"], [title*="User"], .bpmn-icon-user-task, [class*="user-task"]').first();
    const hasUserTask = await userTaskPalette.count() > 0;
    console.log(`[TEST] User Task palette item found: ${hasUserTask}`);
    
    if (hasUserTask) {
      // Click on user task to create one
      await userTaskPalette.click();
      await settle(1000);
      
      // Click on canvas to place it
      const canvasEl = page.locator('.djs-container, .bjs-canvas-wrapper, svg').first();
      if (await canvasEl.count() > 0) {
        const box = await canvasEl.boundingBox();
        if (box) {
          // Click in the middle of the canvas
          await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
          await settle(1500);
        }
      }
    }
    
    await snap(page, '03-user-task-added');
  });

  test('4. 节点属性面板 - 用户任务审批语义', async ({ page }) => {
    // Navigate to workflow designer
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    // Open first existing workflow (fastest path)
    const firstLink = page.locator('table tbody tr:first-child a, table tbody tr:first-child td:first-child a, table tbody tr:first-child .ant-btn').first();
    if (await firstLink.count() > 0) {
      await firstLink.click();
      await settle(3000);
    } else {
      // Create new
      const createBtn = page.locator('button:has-text("新建"), button:has-text("创建")').first();
      if (await createBtn.count() > 0) {
        await createBtn.click();
        await settle(1500);
        const nameInput = page.locator('input[id*="name"], input[placeholder*="名称"], .ant-modal input').first();
        if (await nameInput.count() > 0) await nameInput.fill('属性验证工作流');
        const confirmBtn = page.locator('.ant-modal button:has-text("确定"), .ant-modal-button-primary').first();
        if (await confirmBtn.count() > 0) { await confirmBtn.click(); await settle(3000); }
      }
    }
    
    await snap(page, '04-designer-loaded-for-test');
    
    // Click on a user task node in the canvas (SVG elements)
    const userTaskNode = page.locator('[data-element-id*="UserTask"], [data-element-id*="Task"], .djs-element rect, .djs-group').first();
    if (await userTaskNode.count() > 0) {
      await userTaskNode.click();
      await settle(1500);
      console.log('[TEST] Clicked on user task node');
    } else {
      // Try clicking any visible node/shape
      const shape = page.locator('.djs-shape, .djs-element').first();
      if (await shape.count() > 0) {
        await shape.click();
        await settle(1500);
        console.log('[TEST] Clicked on first available shape');
      }
    }
    
    await snap(page, '04-node-selected-properties');
    
    // Check for "审批语义" panel
    const approvalPanel = page.locator('text=审批语义');
    const hasApprovalPanel = await approvalPanel.count() > 0;
    console.log(`[TEST] 审批语义 panel visible: ${hasApprovalPanel}`);
    
    if (hasApprovalPanel) {
      // Try changing to approval mode
      const taskPurposeSelect = page.locator('text=审批任务').first();
      if (await taskPurposeSelect.count() > 0) {
        await taskPurposeSelect.click();
        await settle(1000);
      } else {
        // Try the select dropdown
        const selectEl = page.locator('.ant-select:near(text=审批语义)').first();
        if (await selectEl.count() > 0) {
          await selectEl.click();
          await settle(500);
          const approvalOption = page.locator('.ant-select-dropdown .ant-select-item:has-text("审批")').first();
          if (await approvalOption.count() > 0) {
            await approvalOption.click();
            await settle(1000);
          }
        }
      }
      
      await snap(page, '04-approval-mode-selected');
      
      // Check for approval mode dropdown
      const approvalModes = ['单人审批', '任一通过', '全部通过', '比例/阈值通过', '顺序会签'];
      for (const mode of approvalModes) {
        const modeEl = page.locator(`text=${mode}`);
        const visible = await modeEl.count() > 0;
        console.log(`[TEST] Approval mode "${mode}": ${visible ? 'VISIBLE' : 'NOT FOUND'}`);
      }
      
      // Check for reject strategy dropdown
      const rejectStrategies = ['终止流程', '退回发起人', '进入拒绝分支'];
      for (const strategy of rejectStrategies) {
        const el = page.locator(`text=${strategy}`);
        const visible = await el.count() > 0;
        console.log(`[TEST] Reject strategy "${strategy}": ${visible ? 'VISIBLE' : 'NOT FOUND'}`);
      }
      
      // Check for disabled features
      const delegateSwitch = page.locator('text=委托').first();
      const addApproverSwitch = page.locator('text=加签').first();
      const timeoutSelect = page.locator('text=仅提醒').first();
      
      console.log(`[TEST] 委托 (delegate) found: ${await delegateSwitch.count() > 0}`);
      console.log(`[TEST] 加签 (addApprover) found: ${await addApproverSwitch.count() > 0}`);
      console.log(`[TEST] 超时操作 found: ${await timeoutSelect.count() > 0}`);
      
      // Try clicking the threshold mode
      const thresholdOption = page.locator('.ant-select-item:has-text("阈值"), .ant-select-item:has-text("比例")').first();
      if (await thresholdOption.count() > 0) {
        await thresholdOption.click();
        await settle(1000);
        await snap(page, '04-threshold-mode');
        
        // Check if threshold input appears
        const thresholdInput = page.locator('text=通过人数');
        console.log(`[TEST] Threshold input visible: ${await thresholdInput.count() > 0}`);
      }
    }
    
    // Check for assignee/candidate fields
    const assigneeField = page.locator('text=直接指派, text=Assignee, text=assignee, [placeholder*="指派"]').first();
    const candidateUsers = page.locator('text=候选用户, text=candidateUsers').first();
    const candidateGroups = page.locator('text=候选组, text=candidateGroups').first();
    
    console.log(`[TEST] Assignee field: ${await assigneeField.count() > 0}`);
    console.log(`[TEST] Candidate Users field: ${await candidateUsers.count() > 0}`);
    console.log(`[TEST] Candidate Groups field: ${await candidateGroups.count() > 0}`);
    
    await snap(page, '04-final-state');
  });

  test('5. 节点属性 - 表单绑定和文档', async ({ page }) => {
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    // Open first workflow
    const firstLink = page.locator('table tbody tr:first-child a, table tbody tr:first-child td:first-child a').first();
    if (await firstLink.count() > 0) {
      await firstLink.click();
      await settle(3000);
    }
    
    // Click on a node
    const node = page.locator('.djs-element, .djs-shape').first();
    if (await node.count() > 0) {
      await node.click();
      await settle(1500);
    }
    
    // Look for form key field
    const formKeyField = page.locator('text=表单, text=Form, text=formKey').first();
    console.log(`[TEST] Form key field: ${await formKeyField.count() > 0}`);
    
    // Look for documentation field
    const docField = page.locator('text=描述信息, text=Documentation, textarea[placeholder*="描述"]').first();
    console.log(`[TEST] Documentation field: ${await docField.count() > 0}`);
    
    if (await docField.count() > 0) {
      await docField.fill('这是自动化测试添加的节点描述');
      await settle(1000);
      await snap(page, '05-documentation-filled');
    }
    
    // Look for priority field
    const priorityField = page.locator('text=优先级, text=Priority').first();
    console.log(`[TEST] Priority field: ${await priorityField.count() > 0}`);
    
    // Look for due date field
    const dueDateField = page.locator('text=截止日期, text=Due Date, text=到期').first();
    console.log(`[TEST] Due Date field: ${await dueDateField.count() > 0}`);
    
    await snap(page, '05-node-properties-final');
  });

  test('6. 工作流配置 Tab - 审批人和SLA', async ({ page }) => {
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    const firstLink = page.locator('table tbody tr:first-child a').first();
    if (await firstLink.count() > 0) {
      await firstLink.click();
      await settle(3000);
    }
    
    // Look for config tab
    const configTab = page.locator('.ant-tabs-tab:has-text("配置"), .ant-tabs-tab:has-text("Config"), .ant-tabs-tab:has-text("config")').first();
    if (await configTab.count() > 0) {
      await configTab.click();
      await settle(1500);
      await snap(page, '06-config-tab');
      
      // Check for approval config
      const approvalSection = page.locator('text=审批人').first();
      console.log(`[TEST] Approval section: ${await approvalSection.count() > 0}`);
      
      // Check for SLA config
      const slaSection = page.locator('text=SLA配置, text=SLA').first();
      console.log(`[TEST] SLA section: ${await slaSection.count() > 0}`);
      
      // Check for approval type options
      const approvalTypes = ['单人', '并行', '顺序', '条件'];
      for (const type of approvalTypes) {
        const el = page.locator(`text=${type}`).first();
        console.log(`[TEST] Approval type "${type}": ${await el.count() > 0}`);
      }
    }
    
    // Look for validation tab
    const validationTab = page.locator('.ant-tabs-tab:has-text("校验"), .ant-tabs-tab:has-text("Validation")').first();
    if (await validationTab.count() > 0) {
      await validationTab.click();
      await settle(1500);
      await snap(page, '06-validation-tab');
    }
    
    // Look for versions tab
    const versionsTab = page.locator('.ant-tabs-tab:has-text("版本"), .ant-tabs-tab:has-text("Version")').first();
    if (await versionsTab.count() > 0) {
      await versionsTab.click();
      await settle(1500);
      await snap(page, '06-versions-tab');
    }
  });

  test('7. 保存工作流测试', async ({ page }) => {
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    const firstLink = page.locator('table tbody tr:first-child a').first();
    if (await firstLink.count() > 0) {
      await firstLink.click();
      await settle(3000);
    }
    
    // Look for save button
    const saveBtn = page.locator('button:has-text("保存"), button:has-text("Save")').first();
    const hasSaveBtn = await saveBtn.count() > 0;
    console.log(`[TEST] Save button found: ${hasSaveBtn}`);
    
    if (hasSaveBtn) {
      await saveBtn.click();
      await settle(2000);
      await snap(page, '07-after-save');
      
      // Check for success/error message
      const successMsg = page.locator('.ant-message-success, .ant-notification-success');
      const errorMsg = page.locator('.ant-message-error, .ant-notification-error, .ant-alert-error');
      console.log(`[TEST] Success message: ${await successMsg.count() > 0}`);
      console.log(`[TEST] Error message: ${await errorMsg.count() > 0}`);
    }
    
    // Look for deploy button
    const deployBtn = page.locator('button:has-text("部署"), button:has-text("Deploy")').first();
    console.log(`[TEST] Deploy button found: ${await deployBtn.count() > 0}`);
  });

  test('8. 网关和服务任务节点属性', async ({ page }) => {
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    const firstLink = page.locator('table tbody tr:first-child a').first();
    if (await firstLink.count() > 0) {
      await firstLink.click();
      await settle(3000);
    }
    
    // Click on gateway node if exists
    const gatewayNode = page.locator('[data-element-id*="Gateway"], .djs-element:has([data-element-id*="Gateway"])').first();
    if (await gatewayNode.count() > 0) {
      await gatewayNode.click();
      await settle(1500);
      await snap(page, '08-gateway-properties');
      
      // Check that approval panel is NOT shown for gateway
      const approvalPanel = page.locator('text=审批语义');
      console.log(`[TEST] 审批语义 visible for Gateway: ${await approvalPanel.count() > 0} (should be FALSE)`);
      
      // Check for condition expression field
      const conditionField = page.locator('text=条件, text=Condition, text=expression').first();
      console.log(`[TEST] Condition field: ${await conditionField.count() > 0}`);
    }
    
    // Click on start event
    const startEvent = page.locator('[data-element-id*="StartEvent"], .djs-element:has([data-element-id*="Start"])').first();
    if (await startEvent.count() > 0) {
      await startEvent.click();
      await settle(1500);
      await snap(page, '08-start-event-properties');
    }
    
    // Click on end event
    const endEvent = page.locator('[data-element-id*="EndEvent"], .djs-element:has([data-element-id*="End"])').first();
    if (await endEvent.count() > 0) {
      await endEvent.click();
      await settle(1500);
      await snap(page, '08-end-event-properties');
    }
  });

  test('9. i18n 和 UI 一致性检查', async ({ page }) => {
    // Collect console errors during navigation
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });
    
    // Collect network errors
    const networkErrors: { url: string; status: number }[] = [];
    page.on('response', response => {
      if (response.status() >= 400) {
        networkErrors.push({ url: response.url(), status: response.status() });
      }
    });
    
    await page.goto(`${BASE_URL}/workflow`);
    await page.waitForLoadState('networkidle');
    await settle(2000);
    
    const firstLink = page.locator('table tbody tr:first-child a').first();
    if (await firstLink.count() > 0) {
      await firstLink.click();
      await settle(3000);
    }
    
    // Check for i18n issues - look for untranslated keys
    const bodyText = await page.textContent('body');
    const i18nKeyPattern = /[a-z]+\.[a-z]+\.[a-z]+/g;
    const possibleKeys = bodyText?.match(i18nKeyPattern) || [];
    
    // Filter for likely i18n keys (common patterns)
    const suspiciousKeys = possibleKeys.filter(k => 
      k.includes('workflow.') || k.includes('common.') || k.includes('action.')
    );
    
    if (suspiciousKeys.length > 0) {
      console.log('[I18N] Possible untranslated i18n keys:', [...new Set(suspiciousKeys)].join(', '));
    }
    
    await snap(page, '09-i18n-check');
    
    // Report errors
    if (consoleErrors.length > 0) {
      console.log('\n=== CONSOLE ERRORS ===');
      consoleErrors.forEach((err, i) => console.log(`  ${i + 1}. ${err}`));
    }
    
    if (networkErrors.length > 0) {
      console.log('\n=== NETWORK ERRORS ===');
      networkErrors.forEach(err => console.log(`  ${err.status} - ${err.url}`));
    }
  });
});
