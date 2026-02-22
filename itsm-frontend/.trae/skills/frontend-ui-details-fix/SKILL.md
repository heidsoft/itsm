---
name: "frontend-ui-details-fix"
description: "Fixes UI/UX details like boundary handling, tooltips, settings panels, responsive design, accessibility, and visual consistency. Invoke when core features work but details have issues."
---

# Frontend UI Details Fix Workflow

Use this workflow to fix UI/UX details issues that are often overlooked during core feature development.

## Common Issues Checklist

### 1. Boundary & Edge Handling

#### Overflow Issues
- **Check**: Content that exceeds container bounds
- **Common Problems**:
  - Long text without truncation or ellipsis
  - Tables that don't handle horizontal scroll properly
  - Modal content overflowing on small screens
  - Card content growing without limit
- **Fixes**:
  ```tsx
  // Text truncation
  <div className="truncate" title={longText}>
    {longText}
  </div>

  // Table scroll
  <Table
    scroll={{ x: 1200 }}
    {...props}
  />

  // Modal max-height
  <Modal>
    <div className="max-h-[70vh] overflow-y-auto">
      {content}
    </div>
  </Modal>
  ```
- **CSS Classes**: `truncate`, `text-ellipsis`, `overflow-hidden`, `overflow-auto`, `max-h-[...]`

#### Border & Spacing
- **Check**: Inconsistent borders, padding, margins
- **Fixes**:
  - Use consistent spacing scale (4px, 8px, 16px, 24px, 32px)
  - Add proper borders to cards, modals, panels
  - Handle 0-height/0-width elements
- **Ant Design**: Use `space` component for consistent spacing

### 2. Tooltip & Hover States

#### Tooltips
- **Check**: Missing tooltips on truncated text, icons, or abbreviated content
- **Fixes**:
  ```tsx
  <Tooltip title={fullText}>
    <span className="truncate block max-w-[200px]">
      {fullText}
    </span>
  </Tooltip>

  <Tooltip title="Edit item">
    <Button icon={<EditOutlined />} />
  </Tooltip>

  // With dynamic content
  <Tooltip title={status === 'active' ? 'Active status' : 'Inactive status'}>
    <Tag>{status}</Tag>
  </Tooltip>
  ```
- **Best Practices**:
  - Add tooltips to all icon-only buttons
  - Add tooltips to truncated text
  - Add tooltips to status indicators
  - Keep tooltip text concise (< 100 chars)

#### Hover States
- **Check**: Missing hover effects on interactive elements
- **Fixes**:
  ```tsx
  // Row hover
  <Table
    onRow={(record) => ({
      onMouseEnter: () => setHoveredRow(record.id),
      onMouseLeave: () => setHoveredRow(null),
      className: hoveredRow === record.id ? 'bg-blue-50' : ''
    })}
  />

  // Card hover lift effect
  <Card
    hoverable
    className="transition-all hover:shadow-lg hover:-translate-y-1"
  />
  ```
- **CSS**: `hover:bg-...`, `hover:shadow-...`, `transition-all`, `duration-200`

### 3. Settings & Configuration Panels

#### Settings Layout
- **Check**: Settings pages are often poorly organized
- **Best Practices**:
  - Group related settings with cards or sections
  - Use proper form layouts (vertical, horizontal)
  - Add clear labels and descriptions
  - Show save status (saved, unsaved changes)
- **Example Structure**:
  ```tsx
  <PageHeader title="Settings" />
  <div className="space-y-6">
    {/* Section 1 */}
    <Card title="General Settings">
      <Form layout="vertical">
        <Form.Item label="Setting Name" extra="Description">
          <Input />
        </Form.Item>
      </Form>
    </Card>

    {/* Section 2 */}
    <Card title="Notifications">
      {/* Settings */}
    </Card>
  </div>
  ```

#### Settings Persistence
- **Check**: Settings are not saved or restored properly
- **Fixes**:
  - Show loading state during save
  - Show success/error toasts
  - Debounce auto-save for non-critical settings
  - Confirm before destructive actions
- **Code Pattern**:
  ```tsx
  const handleSave = async () => {
    setLoading(true);
    try {
      await api.updateSettings(values);
      message.success('Settings saved');
    } catch (error) {
      message.error('Failed to save settings');
    } finally {
      setLoading(false);
    }
  };
  ```

### 4. Top Right Notification/Menu Area

#### User Menu
- **Check**: User dropdown is often missing important items
- **Required Items**:
  - User info (name, avatar, role)
  - Settings link
  - Logout button
  - Profile/Account settings
- **Example**:
  ```tsx
  <Dropdown
    menu={{
      items: [
        {
          key: 'profile',
          icon: <UserOutlined />,
          label: 'Profile'
        },
        {
          key: 'settings',
          icon: <SettingOutlined />,
          label: 'Settings'
        },
        { type: 'divider' },
        {
          key: 'logout',
          icon: <LogoutOutlined />,
          label: 'Logout',
          danger: true
        }
      ]
    }}
  >
    <Space className="cursor-pointer">
      <Avatar src={user.avatar} />
      <span>{user.name}</span>
    </Space>
  </Dropdown>
  ```

#### Notifications
- **Check**: Notification badge is often missing or not updated
- **Fixes**:
  - Show unread count badge
  - Show preview of latest notifications
  - Mark as read on click
  - Handle empty state
- **Code Pattern**:
  ```tsx
  <Badge count={unreadCount} offset={[-5, 5]}>
    <BellOutlined className="text-lg" />
  </Badge>
  ```

### 5. Loading & Empty States

#### Loading States
- **Check**: Missing or inconsistent loading states
- **Fixes**:
  ```tsx
  // Page loading
  {isLoading ? (
    <div className="flex justify-center py-12">
      <Spin size="large" />
    </div>
  ) : (
    <Content />
  )}

  // Button loading
  <Button loading={isSubmitting} disabled={isSubmitting}>
    Submit
  </Button>

  // Table loading
  <Table loading={isLoading} />

  // Skeleton loading
  <Skeleton active paragraph={{ rows: 4 }} />
  ```

#### Empty States
- **Check**: Empty states are often missing or poorly designed
- **Fixes**:
  ```tsx
  {isEmpty ? (
    <Empty
      description="No data found"
      image={Empty.PRESENTED_IMAGE_SIMPLE}
    >
      <Button type="primary">Create New</Button>
    </Empty>
  ) : (
    <List items={data} />
  )}
  ```

### 6. Responsive Design

#### Mobile Responsiveness
- **Check**: Layout breaks on mobile screens
- **Common Issues**:
  - Tables not scrollable on mobile
  - Modals too large for small screens
  - Sidebars not collapsible
  - Text too small to read
- **Fixes**:
  ```tsx
  // Responsive table
  <Table
    scroll={{ x: 'max-content' }}
    className="hidden md:table"
  />
  <Table
    size="small"
    className="md:hidden"
  />

  // Responsive grid
  <Row gutter={[16, 16]}>
    <Col xs={24} sm={12} md={8} lg={6}>
      <Card />
    </Col>
  </Row>

  // Responsive modal
  <Modal
    width={800}
    className="md:w-full md:mx-4"
  />

  // Breakpoint checks in CSS
  // sm: 640px, md: 768px, lg: 1024px, xl: 1280px, 2xl: 1536px
  ```

### 7. Accessibility (A11y)

#### Keyboard Navigation
- **Check**: Can't navigate with keyboard
- **Fixes**:
  - Ensure all interactive elements are focusable
  - Add proper focus styles
  - Support Tab/Enter navigation
- **Code**:
  ```tsx
  <button
    tabIndex={0}
    onKeyDown={(e) => {
      if (e.key === 'Enter') onClick();
    }}
  >
  ```

#### Screen Reader Support
- **Check**: Missing ARIA labels
- **Fixes**:
  ```tsx
  <Icon aria-label="Delete" />
  <input aria-label="Search" placeholder="..." />
  <div role="status" aria-live="polite">
    {message}
  </div>
  ```

#### Color Contrast
- **Check**: Text hard to read
- **Fixes**:
  - Use Ant Design's color system
  - WCAG AA: 4.5:1 ratio for normal text
  - WCAG AA: 3:1 ratio for large text

### 8. Error Handling

#### Form Validation
- **Check**: Validation feedback is unclear
- **Fixes**:
  ```tsx
  <Form.Item
    name="email"
    rules={[
      { required: true, message: 'Email is required' },
      { type: 'email', message: 'Invalid email format' }
    ]}
    help="We'll send a confirmation email"
  >
    <Input placeholder="email@example.com" />
  </Form.Item>
  ```

#### Error Boundaries
- **Check**: Unhandled errors crash the page
- **Fixes**:
  ```tsx
  <ErrorBoundary fallback={<ErrorPage />}>
    <Component />
  </ErrorBoundary>
  ```

#### Error Messages
- **Check**: Generic error messages
- **Fixes**:
  - Be specific about what went wrong
  - Provide actionable next steps
  - Show technical details only when expanded
  ```tsx
  message.error('Failed to load data. Please refresh the page.');
  ```

### 9. Performance Issues

#### Unnecessary Re-renders
- **Check**: Components re-render too often
- **Fixes**:
  ```tsx
  const MemoizedComponent = React.memo(Component);

  // Use useMemo for expensive calculations
  const sorted = useMemo(() => sort(data), [data]);

  // Use useCallback for event handlers
  const handleClick = useCallback(() => { ... }, [deps]);
  ```

#### Large Lists
- **Check**: Long lists cause lag
- **Fixes**:
  ```tsx
  <List
    virtual
    height={500}
    itemHeight={60}
    dataSource={largeData}
  />
  ```

## Common Patterns to Apply

### Page Layout Pattern
```tsx
export default function PageName() {
  return (
    <div className="p-6">
      <PageHeader
        title="Page Title"
        extra={[
          <Button key="create">Create</Button>,
          <Button key="refresh">Refresh</Button>
        ]}
      />

      <Card className="mt-4">
        {/* Content */}
      </Card>
    </div>
  );
}
```

### Detail View Pattern
```tsx
export default function DetailView({ id }) {
  const { data, loading } = useDetail(id);

  if (loading) return <Spin />;
  if (!data) return <Empty />;

  return (
    <PageLayout>
      <DescriptionItems
        title="基本信息"
        items={[
          { label: '名称', value: data.name },
          { label: '状态', value: <Tag>{data.status}</Tag> },
          // ...
        ]}
      />

      <Tabs
        items={[
          { key: 'details', label: '详情', children: <Details /> },
          { key: 'history', label: '历史', children: <History /> },
        ]}
      />
    </PageLayout>
  );
}
```

### List View Pattern
```tsx
export default function ListView() {
  const { data, loading, pagination } = useList();

  return (
    <PageLayout>
      <FilterBar />
      <Table
        loading={loading}
        dataSource={data}
        pagination={pagination}
        scroll={{ x: 'max-content' }}
        rowKey="id"
      />
    </PageLayout>
  );
}
```

## Verification Checklist

Before closing a UI details fix task, verify:

- [ ] All truncated text has tooltips
- [ ] All icon-only buttons have tooltips
- [ ] All loading states are visible
- [ ] All empty states are handled
- [ ] All forms have validation messages
- [ ] All error messages are specific and actionable
- [ ] Responsive design works on mobile/tablet
- [ ] Keyboard navigation works
- [ ] Color contrast is adequate
- [ ] Borders and spacing are consistent
- [ ] Hover states are visible on interactive elements
- [ ] Settings pages are properly organized
- [ ] User menu has required items
- [ ] Notification badge shows count

## Common Gotchas

1. **Tooltip Position**: On mobile, tooltips may appear off-screen. Use `placement="bottom"` or handle mobile separately.

2. **Table Row Height**: Fixed row heights can cause content truncation. Use `pagination.pageSize` carefully.

3. **Modal Z-index**: Multiple modals can stack incorrectly. Use `Modal.config({ rootPrefixCls })` if needed.

4. **Date Picker Range**: Range picker needs enough width. `width: 100%` on mobile.

5. **Select Dropdown**: On mobile, native select is often better than Ant Design select.

6. **Icon Imports**: Always import from `@ant-design/icons`, not `antd/es/icons`.

7. **Form.Item Layout**: Use `layout="vertical"` on mobile for better UX.

8. **Notification Auto-close**: Set `duration: 5` for success, `0` for errors (manual close).
