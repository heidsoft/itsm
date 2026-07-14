import time
from playwright.sync_api import Page, expect


def test_tickets_management(page: Page):
    """
    Test comprehensive Ticket Management functionality:
    1. Login
    2. Create Ticket
    3. List View (Search, Filter)
    4. View Ticket Details
    5. Edit Ticket
    6. Delete Ticket
    """

    # --- 1. Login ---
    print("\n[Test] 1. Login...")
    page.goto("http://localhost:3000/login")
    page.get_by_placeholder("请输入用户名").fill("admin")
    page.get_by_placeholder("请输入密码").fill("admin123")
    page.get_by_role("button", name="登录", exact=True).click()

    # Wait for dashboard
    try:
        expect(page).to_have_url(
            "http://localhost:3000/dashboard",
            timeout=10000,
        )
    except Exception:
        print(f"Login failed or timeout. Current URL: {page.url}")
        page.screenshot(path="tickets_test_login_fail.png")
        raise

    # --- 2. Navigate to Tickets ---
    print("[Test] 2. Navigate to Tickets...")
    # Handle sidebar scrolling if necessary
    menu_item = page.locator(".ant-menu-item").filter(has_text="工单管理").first
    if not menu_item.is_visible():
        # Using evaluate to scroll the sidebar container
        # Split string to avoid line length limit
        script = "document.querySelector('.Sidebar_mainMenu__TrtX5')"
        script += "?.scrollTo(0, 0)"
        page.evaluate(script)
        time.sleep(0.5)
    menu_item.click()
    expect(page).to_have_url("http://localhost:3000/tickets")

    # --- 3. Create Ticket ---
    print("[Test] 3. Create Ticket...")
    # Click "Create Ticket" button
    # There are two buttons: one in header, one in floating action
    # Let's use the one in the header or the main action area
    create_btn = page.get_by_role("button", name="新建工单").first
    create_btn.click()
    expect(page).to_have_url("http://localhost:3000/tickets/create")

    # Fill form
    timestamp = int(time.time())
    ticket_title = f"E2E Test Ticket {timestamp}"
    ticket_desc = (
        "This is an automated test ticket created by Playwright "
        "E2E test suite. "
        f"Timestamp: {timestamp}. It needs to be at least 10 chars."
    )

    page.get_by_placeholder("例如：VPN 无法连接").fill(ticket_title)
    page.get_by_placeholder("请详细描述问题/需求与影响范围...").fill(ticket_desc)

    # Select Priority
    # AntD Select is tricky. We need to click the selector first.
    # The label is "优先级", find the sibling selector
    # Or navigate by form item
    page.locator("#priority").click()  # AntD form item id usually matches name
    # Select 'High' (高)
    page.get_by_text("高", exact=True).click()

    # Select Category
    page.locator("#category").click()
    # Select '网络' (Network)
    page.get_by_title("网络").nth(1).click()

    # Submit
    page.get_by_role("button", name="创建").click()

    # Wait for redirection to detail page
    # URL pattern: /tickets/\d+
    print("Waiting for ticket creation...")
    page.wait_for_url(r"http://localhost:3000/tickets/\d+", timeout=10000)
    new_ticket_url = page.url
    ticket_id = new_ticket_url.split("/")[-1]
    print(f"Ticket created. ID: {ticket_id}, URL: {new_ticket_url}")

    # Verify Detail Page Content
    expect(page.get_by_text(ticket_title)).to_be_visible()
    expect(page.get_by_text(ticket_desc)).to_be_visible()

    # --- 4. Edit Ticket (in Detail View) ---
    print("[Test] 4. Edit Ticket...")
    # Click Edit button in detail view
    page.get_by_role("button", name="编辑").click()

    # Wait for modal
    expect(page.get_by_text("编辑工单")).to_be_visible()

    # Update title
    updated_title = f"{ticket_title} - Updated"
    page.locator("#title").fill(updated_title)

    # Save
    page.get_by_role("button", name="保存修改").click()

    # Verify update
    expect(page.get_by_text("编辑工单")).not_to_be_visible()  # Modal closed
    expect(page.get_by_text(updated_title)).to_be_visible()
    print("Ticket updated successfully.")

    # --- 5. Return to List and Search ---
    print("[Test] 5. List View & Search...")
    page.get_by_role("button", name="返回列表").click()
    expect(page).to_have_url("http://localhost:3000/tickets")

    # Search for the ticket
    search_input = page.get_by_placeholder("搜索工单标题、描述或工单号")
    search_input.fill(updated_title)
    # Trigger search (usually Enter or wait for debounce)
    # The code uses useDebounce(300ms)
    time.sleep(1)

    # Verify ticket is in list
    expect(page.get_by_text(updated_title)).to_be_visible()
    print("Ticket found in list.")

    # --- 6. Delete Ticket ---
    print("[Test] 6. Delete Ticket...")
    # Find the row with the ticket
    # In AntD table, we look for the row containing the text
    row = page.locator("tr").filter(has_text=updated_title)

    # Click "More" actions (ellipsis)
    # The button usually has an icon, might be hard to select by text.
    # Look for button inside the row
    more_btn = row.locator(".anticon-more").locator("..")
    more_btn.click()

    # Click Delete in dropdown
    page.get_by_role("menuitem", name="删除").click()

    # Confirm delete modal
    expect(page.get_by_text("确认删除")).to_be_visible()
    page.get_by_role("button", name="确认", exact=True).click()

    # Verify deletion
    # Wait for message or row disappearance
    expect(page.get_by_text("删除成功")).to_be_visible()
    expect(page.get_by_text(updated_title)).not_to_be_visible()
    print("Ticket deleted successfully.")

    print("\n[Test] All Ticket Management tests passed!")
