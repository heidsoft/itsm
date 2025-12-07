import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { App as AntdApp } from 'antd'
import { TicketList } from '@/components/business/TicketList'

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => ({ currentTenant: { id: 1 } }),
}))

const getTicketsMock = jest.fn(async () => ({ data: [], total: 0 }))

jest.mock('@/lib/services/ticket-service', () => ({
  ticketService: {
    getTickets: (...args: any[]) => getTicketsMock(...args),
    deleteTicket: jest.fn(async () => undefined),
    exportTickets: jest.fn(async () => new Blob(['id,title'], { type: 'text/csv' })),
  },
}))

describe('TicketList interactions', () => {
  it('loads tickets with default pagination and tenant filter', async () => {
    render(
      <AntdApp>
        <TicketList />
      </AntdApp>
    )
    await waitFor(() => expect(getTicketsMock).toHaveBeenCalled())
    const params = getTicketsMock.mock.calls[0][0]
    expect(params.page).toBe(1)
    expect(params.size).toBe(10)
  })

  it('applies search and status filter', async () => {
    render(
      <AntdApp>
        <TicketList />
      </AntdApp>
    )

    const search = screen.getByPlaceholderText('搜索工单标题或描述') as HTMLInputElement
    fireEvent.change(search, { target: { value: '网络' } })
    fireEvent.keyDown(search, { key: 'Enter' })

    const statusSelect = screen.getByPlaceholderText('状态')
    fireEvent.mouseDown(statusSelect)
    const option = await screen.findByText('处理中')
    fireEvent.click(option)

    await waitFor(() => expect(getTicketsMock).toHaveBeenCalledTimes(3))
    const params = getTicketsMock.mock.calls[getTicketsMock.mock.calls.length - 1][0]
    expect(params.search).toBe('网络')
    expect(params.status).toBe('in_progress')
  })
})

