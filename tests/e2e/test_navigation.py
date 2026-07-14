from playwright.sync_api import Page, expect


def test_login_and_navigation(page: Page):
    failures = []
    # 1. Login
    print("\nNavigating to login page...")
    page.goto("http://localhost:3000/login")

    print("Filling credentials...")
    page.get_by_placeholder("请输入用户名").fill("admin")
    page.get_by_placeholder("请输入密码").fill("admin123")

    print("Clicking login...")
    page.get_by_role("button", name="登录", exact=True).click()

    # Wait for dashboard
    print("Waiting for dashboard...")
    try:
        expect(page).to_have_url(
            "http://localhost:3000/dashboard",
            timeout=5000,
        )
    except Exception as e:
        print(f"Failed to reach dashboard. Current URL: {page.url}")
        page.screenshot(path="login_failed.png")
        raise e

    # 2. Navigation
    # Define menu items and expected URLs with their translation keys
    menu_items = [
        {"name": "工单管理", "path": "/tickets", "key": "tickets"},
        {"name": "事件管理", "path": "/incidents", "key": "incidents"},
        {"name": "问题管理", "path": "/problems", "key": "problems"},
        {"name": "变更管理", "path": "/changes", "key": "changes"},
        {"name": "CMDB", "path": "/cmdb", "key": "cmdb"},
        {
            "name": "服务目录",
            "path": "/service-catalog",
            "key": "serviceCatalog",
        },
        {"name": "知识库", "path": "/knowledge", "key": "knowledgeBase"},
        {"name": "SLA监控", "path": "/sla-dashboard", "key": "sla"},
        {"name": "报表与分析", "path": "/reports", "key": "reports"},
        {"name": "系统管理中心", "path": "/admin", "key": "admin"},
    ]

    # Locate the sidebar menu container to scope our searches
    # Ant Design Menu is usually in a sider
    sidebar = page.locator(".ant-layout-sider")

    for item in menu_items:
        print(f"Navigating to {item['name']}...")
        try:
            # Ensure sidebar is visible
            expect(sidebar).to_be_visible()

            # Find the menu item
            # Use a generic locator that might match text inside menu items
            menu_item = sidebar.get_by_text(item["name"], exact=True)

            # Check if visible, if not, maybe scroll sidebar
            # Try to find the specific scrollable container
            # .mainMenu is for main items, .adminMenu for admin items
            # We try scrolling both just in case
            if not menu_item.is_visible():
                msg = (
                    "  Menu item " + item["name"] + " not visible, trying to scroll..."
                )
                print(msg)
                # Try scrolling the main menu container
                script = "document.querySelector('.Sidebar_mainMenu__TrtX5')"
                script += "?.scrollTo(0, 1000)"
                page.evaluate(script)
                # Fallback: simple scrollIntoView
                try:
                    menu_item.scroll_into_view_if_needed(timeout=2000)
                except Exception:
                    pass

            if not menu_item.is_visible():
                msg = (
                    "  Menu item "
                    + item["name"]
                    + " still not found/visible! Attempting force click..."
                )
                print(msg)
                # Try force click if locator is found but hidden
                menu_item.click(force=True)
            else:
                menu_item.click()

            # Verify URL
            expect(page).to_have_url(f"http://localhost:3000{item['path']}")

            # 3. Specific interactions
            if item["key"] == "tickets":
                print("  [Tickets] Testing Advanced Search...")
                # Click 'Advanced Search' button to toggle panel
                adv_search_btn = page.get_by_role("button", name="高级搜索")
                if adv_search_btn.is_visible():
                    adv_search_btn.click()
                    # Now search input should be visible
                    search_input = page.get_by_placeholder("标题、描述、工单号...")
                    if search_input.is_visible():
                        search_input.fill("Test Ticket")
                        submit_btn = (
                            page.locator("button[type='submit']")
                            .filter(has_text="搜索")
                            .first
                        )
                        submit_btn.click()
                        print("    Search submitted")
                    else:
                        print("    Search input not found in advanced panel")

                    # Close advanced search to clean up? Or leave it.
                    # adv_search_btn.click()
                else:
                    print("  Advanced Search button not found")

                print("  [Tickets] Checking Action Buttons...")
                # Check for "New Ticket" button
                expect(page.get_by_role("button", name="新建工单")).to_be_visible()

                # Check list actions (Edit/Delete)
                # Need to find a row first
                # AntD Table rows
                rows = page.locator(".ant-table-row")
                if rows.count() > 0:
                    count = rows.count()
                    print(f"    Found {count} tickets. Testing row actions...")
                    first_row = rows.first
                    # Find the "More" button (ellipsis)
                    more_btn = first_row.locator(".anticon-more").locator("..")
                    if more_btn.is_visible():
                        more_btn.click()
                        # Now dropdown should be open
                        # Check for "编辑" and "删除"
                        expect(
                            page.get_by_role("menuitem", name="编辑")
                        ).to_be_visible()
                        expect(
                            page.get_by_role("menuitem", name="删除")
                        ).to_be_visible()
                        # Click outside to close
                        page.mouse.click(0, 0)
                    else:
                        print("    More actions button not found in row")
                else:
                    print("    No tickets found in list")

            elif item["key"] == "incidents":
                print("  [Incidents] Testing Search Input...")
                search_input = page.get_by_placeholder("搜索事件ID、标题或描述...")
                if search_input.is_visible():
                    search_input.fill("INC-TEST")
                    # Note: No search trigger implemented in page yet
                    print("    Search input filled (no trigger available)")
                else:
                    print("    Search input not found")

                print("  [Incidents] Checking Buttons...")
                expect(page.get_by_role("button", name="新建事件")).to_be_visible()
                expect(page.get_by_role("button", name="筛选")).to_be_visible()

            elif item["key"] == "sla":
                print("  [SLA] Checking dashboard elements...")
                expect(page.get_by_text("SLA监控仪表盘")).to_be_visible()

                # Test Refresh
                print("  [SLA] Testing Refresh...")
                refresh_btn = page.get_by_role("button", name="刷新")
                if refresh_btn.is_visible():
                    refresh_btn.click()
                    print("    Refresh clicked")

                # Test Date Range Picker existence
                expect(page.locator(".ant-picker-range")).to_be_visible()

            # Add more specific checks for other pages as needed...

        except Exception as e:
            print(f"Error testing {item['name']}: {e}")
            page.screenshot(path=f"error_{item['key']}.png")
            failures.append(f"{item['name']}: {e}")

    assert failures == [], "E2E failures:\n" + "\n".join(failures)
    print("Test completed successfully.")
