/**
 * ApprovalPage - 审批工作流页面对象
 * 提供审批工作流列表、创建、编辑、审批操作等 UI 交互封装
 */
import { type Page, type Locator, expect } from '@playwright/test';

export class ApprovalPage {
  readonly page: Page;

  // 页面元素
  readonly createButton: Locator;
  readonly workflowTable: Locator;
  readonly searchInput: Locator;
  readonly filterDropdown: Locator;

  // 审批操作按钮
  readonly approveButton: Locator;
  readonly rejectButton: Locator;
  readonly delegateButton: Locator;

  // 审批记录相关
  readonly pendingApprovalsTab: Locator;
  readonly approvedTab: Locator;
  readonly rejectedTab: Locator;
  readonly historyTab: Locator;

  constructor(page: Page) {
    this.page = page;

    // 主页面元素
    this.createButton = page.locator('button:has-text("创建"), button:has-text("新建")').first();
    this.workflowTable = page.locator('table.ant-table, [class*="table"]').first();
    this.searchInput = page.locator('input[placeholder*="搜索"], input[type="search"]').first();
    this.filterDropdown = page.locator('.ant-select, [class*="filter"]').first();

    // 审批操作按钮（工单详情页）
    this.approveButton = page.locator('button:has-text("批准"), button:has-text("通过"), button:has-text("同意")').first();
    this.rejectButton = page.locator('button:has-text("拒绝"), button:has-text("驳回")').first();
    this.delegateButton = page.locator('button:has-text("转交"), button:has-text("委托")').first();

    // 审批记录 Tab
    this.pendingApprovalsTab = page.locator('div.ant-tabs-tab:has-text("待审批"), div.ant-tabs-tab:has-text("待处理")').first();
    this.approvedTab = page.locator('div.ant-tabs-tab:has-text("已通过"), div.ant-tabs-tab:has-text("已批准")').first();
    this.rejectedTab = page.locator('div.ant-tabs-tab:has-text("已拒绝"), div.ant-tabs-tab:has-text("已驳回")').first();
    this.historyTab = page.locator('div.ant-tabs-tab:has-text("历史"), div.ant-tabs-tab:has-text("审批历史")').first();
  }

  /**
   * 导航到审批工作流页面
   */
  async goto(): Promise<void> {
    await this.page.goto('/approval-workflows');
    await this.page.waitForLoadState('domcontentloaded');
  }

  /**
   * 导航到审批记录页面
   */
  async gotoRecords(): Promise<void> {
    await this.page.goto('/approvals');
    await this.page.waitForLoadState('domcontentloaded');
  }

  /**
   * 等待页面加载完成
   */
  async waitForLoad(): Promise<void> {
    await this.page.waitForSelector('table, [class*="table"], [class*="list"]', { timeout: 10000 });
  }

  /**
   * 获取第一行审批记录 ID
   */
  async getFirstRecordId(): Promise<string | null> {
    const firstRow = this.page.locator('table tbody tr').first();
    const idCell = firstRow.locator('td').nth(0);
    if (await idCell.isVisible()) {
      return await idCell.textContent();
    }
    return null;
  }

  /**
   * 点击创建工作流按钮
   */
  async clickCreate(): Promise<void> {
    await this.createButton.click();
    await this.page.waitForLoadState('domcontentloaded');
  }

  /**
   * 审批通过操作
   * @param comment 审批意见
   */
  async approve(comment?: string): Promise<void> {
    if (comment) {
      const commentInput = this.page.locator('textarea, input[class*="comment"]');
      if (await commentInput.isVisible()) {
        await commentInput.fill(comment);
      }
    }
    await this.approveButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 审批拒绝操作
   * @param comment 拒绝原因
   */
  async reject(comment: string): Promise<void> {
    const commentInput = this.page.locator('textarea, input[class*="comment"]');
    if (await commentInput.isVisible()) {
      await commentInput.fill(comment);
    }
    await this.rejectButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 委托操作
   * @param targetUserId 目标用户 ID
   */
  async delegate(targetUserId: number): Promise<void> {
    // 选择委托用户
    const userSelect = this.page.locator('[class*="user"], [class*="approver"] select, .ant-select');
    if (await userSelect.isVisible()) {
      await userSelect.click();
      await this.page.locator(`.ant-select-dropdown:visible li:has-text("${targetUserId}")`).click();
    }
    await this.delegateButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 切换到待审批 Tab
   */
  async switchToPending(): Promise<void> {
    await this.pendingApprovalsTab.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 切换到已通过 Tab
   */
  async switchToApproved(): Promise<void> {
    await this.approvedTab.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 切换到已拒绝 Tab
   */
  async switchToRejected(): Promise<void> {
    await this.rejectedTab.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 切换到历史 Tab
   */
  async switchToHistory(): Promise<void> {
    await this.historyTab.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 搜索审批记录
   * @param keyword 关键词
   */
  async search(keyword: string): Promise<void> {
    await this.searchInput.fill(keyword);
    await this.searchInput.press('Enter');
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 过滤审批记录
   * @param filterType 过滤类型: status, type, priority
   * @param value 过滤值
   */
  async filterBy(filterType: string, value: string): Promise<void> {
    await this.filterDropdown.click();
    const dropdown = this.page.locator('.ant-select-dropdown:visible');
    await dropdown.locator(`li:has-text("${value}")`).click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 获取审批记录数量
   */
  async getRecordCount(): Promise<number> {
    const rows = this.page.locator('table tbody tr');
    return await rows.count();
  }

  /**
   * 验证审批记录存在
   * @param expectedCount 期望的数量
   */
  async expectRecordCount(expectedCount: number): Promise<void> {
    const count = await this.getRecordCount();
    expect(count).toBeGreaterThanOrEqual(expectedCount);
  }

  /**
   * 点击第一行审批记录查看详情
   */
  async clickFirstRecord(): Promise<void> {
    const firstRow = this.page.locator('table tbody tr').first();
    await firstRow.click();
    await this.page.waitForLoadState('domcontentloaded');
  }

  /**
   * 创建审批工作流
   * @param workflowData 工作流数据
   */
  async createWorkflow(workflowData: {
    name: string;
    description?: string;
    ticketType?: string;
    priority?: string;
    nodes: Array<{
      level: number;
      name: string;
      approverType: string;
      approverIds?: number[];
      approvalMode: string;
      allowReject?: boolean;
      allowDelegate?: boolean;
    }>;
  }): Promise<void> {
    // 填写基本信息
    const nameInput = this.page.locator('input[name="name"], input[id="name"]');
    await nameInput.fill(workflowData.name);

    if (workflowData.description) {
      const descInput = this.page.locator('textarea[name="description"], textarea[id="description"]');
      await descInput.fill(workflowData.description);
    }

    // 选择工单类型
    if (workflowData.ticketType) {
      const typeSelect = this.page.locator('[id="ticket_type"], [name="ticket_type"]');
      await typeSelect.click();
      await this.page.locator(`.ant-select-dropdown:visible li:has-text("${workflowData.ticketType}")`).click();
    }

    // 选择优先级
    if (workflowData.priority) {
      const prioritySelect = this.page.locator('[id="priority"], [name="priority"]');
      await prioritySelect.click();
      await this.page.locator(`.ant-select-dropdown:visible li:has-text("${workflowData.priority}")`).click();
    }

    // 添加审批节点
    for (const node of workflowData.nodes) {
      await this.addApprovalNode(node);
    }

    // 提交创建
    const submitButton = this.page.locator('button[type="submit"], button:has-text("提交"), button:has-text("创建")');
    await submitButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * 添加审批节点
   */
  async addApprovalNode(node: {
    level: number;
    name: string;
    approverType: string;
    approverIds?: number[];
    approvalMode: string;
    allowReject?: boolean;
    allowDelegate?: boolean;
  }): Promise<void> {
    // 点击添加节点按钮
    const addNodeButton = this.page.locator('button:has-text("添加节点"), button:has-text("+ 添加")');
    if (await addNodeButton.isVisible()) {
      await addNodeButton.click();
    }

    // 填写节点信息
    const levelInput = this.page.locator('input[name="level"], input[id="level"]');
    await levelInput.fill(String(node.level));

    const nameInput = this.page.locator('input[name="node_name"], input[id="node_name"], input[placeholder*="节点名称"]');
    await nameInput.fill(node.name);

    // 选择审批人类型
    const approverTypeSelect = this.page.locator('[id="approver_type"], [name="approver_type"]');
    await approverTypeSelect.click();
    await this.page.locator(`.ant-select-dropdown:visible li:has-text("${node.approverType}")`).click();

    // 选择审批模式
    const modeSelect = this.page.locator('[id="approval_mode"], [name="approval_mode"]');
    await modeSelect.click();
    await this.page.locator(`.ant-select-dropdown:visible li:has-text("${node.approvalMode}")`).click();

    // 配置拒绝和委托权限
    if (node.allowReject !== undefined) {
      const rejectSwitch = this.page.locator('[id="allow_reject"], [name="allow_reject"] + .ant-switch, [class*="reject"] .ant-switch');
      if (await rejectSwitch.isVisible()) {
        const isChecked = await rejectSwitch.getAttribute('class');
        if ((node.allowReject && !isChecked?.includes('ant-switch-checked')) ||
            (!node.allowReject && isChecked?.includes('ant-switch-checked'))) {
          await rejectSwitch.click();
        }
      }
    }

    if (node.allowDelegate !== undefined) {
      const delegateSwitch = this.page.locator('[id="allow_delegate"], [name="allow_delegate"] + .ant-switch, [class*="delegate"] .ant-switch');
      if (await delegateSwitch.isVisible()) {
        const isChecked = await delegateSwitch.getAttribute('class');
        if ((node.allowDelegate && !isChecked?.includes('ant-switch-checked')) ||
            (!node.allowDelegate && isChecked?.includes('ant-switch-checked'))) {
          await delegateSwitch.click();
        }
      }
    }
  }
}
