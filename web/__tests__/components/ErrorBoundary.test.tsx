import { screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ErrorBoundary } from '../../src/components/common/ErrorBoundary'
import { renderWithProviders } from '../test-utils'

function CrashingView() {
  throw new Error('phase9 crash')
  return null
}

describe('ErrorBoundary', () => {
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders a route-level fallback when a page crashes', () => {
    renderWithProviders(
      <ErrorBoundary>
        <CrashingView />
      </ErrorBoundary>,
    )

    expect(screen.getByText('页面加载失败')).toBeInTheDocument()
    expect(screen.getByText('phase9 crash')).toBeInTheDocument()
  })
})
