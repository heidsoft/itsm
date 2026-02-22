---
name: "frontend-testing-guide"
description: "Comprehensive guide for frontend testing including unit tests, integration tests, component testing, and E2E testing with Jest and Playwright. Invoke when user needs to write frontend tests, fix test failures, or improve test coverage."
---

# Frontend Testing Guide

This skill provides comprehensive guidance for testing the ITSM frontend application, including unit tests, integration tests, component testing, and end-to-end testing with Jest and Playwright.

## Test Structure

### Unit Tests (`__tests__/` directories)
- Located alongside components in `__tests__/` directories
- Test individual components and functions in isolation
- Use Jest with React Testing Library
- Follow naming convention: `ComponentName.test.tsx`

### Integration Tests (`src/__tests__/pages/`)
- Located in `src/__tests__/pages/`
- Test page-level components and their interactions
- Mock API calls and external dependencies
- Test user interactions and state management

### E2E Tests (`tests/e2e/`)
- Located in `tests/e2e/`
- Test complete user workflows through the browser
- Use Playwright for browser automation
- Test real API interactions and database state

## Test Configuration

### Jest Configuration (`jest.config.js`)
```javascript
module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  testMatch: [
    '**/__tests__/**/*.test.tsx',
    '**/__tests__/**/*.test.ts'
  ],
  moduleNameMapping: {
    '^@/(.*)$': '<rootDir>/src/$1'
  },
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/__tests__/**'
  ]
}
```

### Playwright Configuration (`playwright.config.ts`)
```typescript
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    }
  ]
})
```

## Component Testing Patterns

### Basic Component Test
```typescript
import { render, screen } from '@testing-library/react'
import { TicketCard } from '../TicketCard'

describe('TicketCard', () => {
  it('renders ticket information correctly', () => {
    const ticket = {
      id: 1,
      title: 'Test Ticket',
      priority: 'high',
      status: 'open'
    }
    
    render(<TicketCard ticket={ticket} />)
    
    expect(screen.getByText('Test Ticket')).toBeInTheDocument()
    expect(screen.getByText('high')).toBeInTheDocument()
    expect(screen.getByText('open')).toBeInTheDocument()
  })
})
```

### Testing with User Interactions
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TicketForm } from '../TicketForm'

describe('TicketForm', () => {
  it('submits form with valid data', async () => {
    const mockSubmit = jest.fn()
    const user = userEvent.setup()
    
    render(<TicketForm onSubmit={mockSubmit} />)
    
    await user.type(screen.getByLabelText('Title'), 'New Ticket')
    await user.type(screen.getByLabelText('Description'), 'Ticket description')
    await user.selectOptions(screen.getByLabelText('Priority'), 'high')
    
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    
    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalledWith({
        title: 'New Ticket',
        description: 'Ticket description',
        priority: 'high'
      })
    })
  })
})
```

### Testing with API Mocking
```typescript
import { render, screen, waitFor } from '@testing-library/react'
import { TicketList } from '../TicketList'
import { getTickets } from '@/lib/api/ticket-api'

jest.mock('@/lib/api/ticket-api')
const mockGetTickets = getTickets as jest.MockedFunction<typeof getTickets>

describe('TicketList', () => {
  it('loads and displays tickets', async () => {
    const tickets = [
      { id: 1, title: 'Ticket 1', status: 'open' },
      { id: 2, title: 'Ticket 2', status: 'closed' }
    ]
    mockGetTickets.mockResolvedValue({ data: tickets })
    
    render(<TicketList />)
    
    await waitFor(() => {
      expect(screen.getByText('Ticket 1')).toBeInTheDocument()
      expect(screen.getByText('Ticket 2')).toBeInTheDocument()
    })
  })
  
  it('handles API errors gracefully', async () => {
    mockGetTickets.mockRejectedValue(new Error('API Error'))
    
    render(<TicketList />)
    
    await waitFor(() => {
      expect(screen.getByText('Failed to load tickets')).toBeInTheDocument()
    })
  })
})
```

## Page Testing Patterns

### Testing Page Components
```typescript
import { render, screen } from '@testing-library/react'
import { TicketsPage } from '../../../app/(main)/tickets/page'

jest.mock('@/lib/api/ticket-api', () => ({
  getTickets: jest.fn().mockResolvedValue({
    data: [
      { id: 1, title: 'Test Ticket', status: 'open' }
    ]
  })
}))

describe('TicketsPage', () => {
  it('renders ticket list', async () => {
    render(<TicketsPage />)
    
    expect(screen.getByText('Tickets')).toBeInTheDocument()
    expect(await screen.findByText('Test Ticket')).toBeInTheDocument()
  })
})
```

### Testing with Router
```typescript
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { TicketDetail } from '../../../app/(main)/tickets/[ticketId]/page'

describe('TicketDetail', () => {
  it('displays ticket information', async () => {
    const mockParams = { ticketId: '123' }
    
    render(
      <MemoryRouter initialEntries={['/tickets/123']}>
        <TicketDetail params={mockParams} />
      </MemoryRouter>
    )
    
    expect(await screen.findByText('Ticket #123')).toBeInTheDocument()
  })
})
```

## E2E Testing Patterns

### Basic Navigation Test
```typescript
import { test, expect } from '@playwright/test'

test.describe('Navigation', () => {
  test('should navigate to tickets page', async ({ page }) => {
    await page.goto('/')
    await page.click('text=Tickets')
    
    await expect(page).toHaveURL('/tickets')
    await expect(page.locator('h1')).toContainText('Tickets')
  })
})
```

### Testing User Workflows
```typescript
import { test, expect } from '@playwright/test'

test.describe('Ticket Workflow', () => {
  test('complete ticket lifecycle', async ({ page }) => {
    // Login
    await page.goto('/login')
    await page.fill('input[name="username"]', 'testuser')
    await page.fill('input[name="password"]', 'password')
    await page.click('button[type="submit"]')
    
    // Create ticket
    await page.click('text=Create Ticket')
    await page.fill('input[name="title"]', 'E2E Test Ticket')
    await page.fill('textarea[name="description"]', 'Test description')
    await page.selectOption('select[name="priority"]', 'high')
    await page.click('button:has-text("Submit")')
    
    // Verify ticket created
    await expect(page.locator('.success-message')).toContainText('Ticket created successfully')
    await expect(page.locator('h1')).toContainText('E2E Test Ticket')
  })
})
```

### Testing Form Validation
```typescript
import { test, expect } from '@playwright/test'

test.describe('Form Validation', () => {
  test('should show validation errors', async ({ page }) => {
    await page.goto('/tickets/create')
    await page.click('button:has-text("Submit")')
    
    await expect(page.locator('.error-message')).toContainText('Title is required')
    await expect(page.locator('.error-message')).toContainText('Description is required')
  })
})
```

## Running Tests

### Unit and Integration Tests
```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run specific test file
npm test -- --testNamePattern="TicketCard"

# Run tests for specific component
npm test -- TicketCard.test.tsx
```

### E2E Tests
```bash
# Run all E2E tests
npm run test:e2e

# Run tests in headed mode (see browser)
npm run test:e2e -- --headed

# Run specific test file
npm run test:e2e -- test_navigation.py

# Run tests with specific browser
npm run test:e2e -- --browser=firefox
```

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| Test timeout | Increase timeout: `jest.setTimeout(10000)` |
| Mock not working | Check mock path matches import exactly |
| act() warning | Use `waitFor` for async operations |
| Cannot find element | Use `screen.debug()` to see DOM structure |
| API call not mocked | Mock the specific API function used |
| E2E test flaky | Add proper waits and assertions |
| Component not updating | Check state updates and re-renders |

## Best Practices

1. **Test Organization**: Group related tests in describe blocks
2. **Test Independence**: Each test should be able to run independently
3. **Clear Assertions**: Use specific assertions, not generic ones
4. **Proper Mocking**: Mock external dependencies appropriately
5. **Error Testing**: Test both success and error scenarios
6. **Accessibility**: Test with screen readers and keyboard navigation
7. **Performance**: Test component rendering performance
8. **Coverage**: Aim for >80% test coverage on critical components