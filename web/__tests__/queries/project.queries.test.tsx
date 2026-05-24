import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { useProjectsQuery } from '../../src/queries/project.queries'
import { ProjectStatus } from '../../src/utils/constants'
import { setToken } from '../../src/utils/token'
import { TestProviders } from '../test-utils'

describe('project queries', () => {
  it('lists active projects by default', async () => {
    setToken('mock-access-token')

    const { result } = renderHook(
      () =>
        useProjectsQuery({
          includeArchived: false,
          limit: 10,
          offset: 0,
        }),
      { wrapper: TestProviders },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.projects.length).toBeGreaterThan(0)
    expect(
      result.current.data?.projects.every((project) => project.status === ProjectStatus.Active),
    ).toBe(true)
  })

  it('includes archived projects when requested', async () => {
    setToken('mock-access-token')

    const { result } = renderHook(
      () =>
        useProjectsQuery({
          includeArchived: true,
          limit: 10,
          offset: 0,
        }),
      { wrapper: TestProviders },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(
      result.current.data?.projects.some((project) => project.status === ProjectStatus.Archived),
    ).toBe(true)
  })
})
