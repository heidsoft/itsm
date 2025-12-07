import React from 'react'
import { render, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import IncidentManagement from '@/components/business/IncidentManagement'
import { IncidentAPI } from '@/lib/api/incident-api'
import { App as AntdApp } from 'antd'

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => ({ currentTenant: { id: 1 } }),
}))

describe('IncidentManagement interactions', () => {
  it('loads incidents via API layer with pagination', async () => {
    const spy = jest.spyOn(IncidentAPI, 'listIncidents').mockResolvedValue({ incidents: [], total: 0 } as any)
    render(
      <AntdApp>
        <IncidentManagement />
      </AntdApp>
    )
    await waitFor(() => expect(spy).toHaveBeenCalled())
    const params = (spy.mock.calls[0] || [])[0] as any
    expect(params.page).toBe(1)
    expect(params.page_size).toBeGreaterThan(0)
    spy.mockRestore()
  })
})

