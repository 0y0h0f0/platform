import { describe, expect, it } from 'vitest'

import {
  getAllowedStatusTransitions,
  isAllowedStatusTransition,
} from '../../src/hooks/useStatusTransitions'
import { TaskStatus } from '../../src/utils/constants'

describe('useStatusTransitions', () => {
  it('returns legal transitions for each task status', () => {
    expect(getAllowedStatusTransitions(TaskStatus.Todo)).toEqual([
      TaskStatus.Doing,
      TaskStatus.Done,
      TaskStatus.Cancelled,
    ])
    expect(getAllowedStatusTransitions(TaskStatus.Doing)).toEqual([
      TaskStatus.Done,
      TaskStatus.Cancelled,
      TaskStatus.Todo,
    ])
    expect(getAllowedStatusTransitions(TaskStatus.Done)).toEqual([TaskStatus.Doing])
    expect(getAllowedStatusTransitions(TaskStatus.Cancelled)).toEqual([TaskStatus.Todo])
  })

  it('checks transition legality', () => {
    expect(isAllowedStatusTransition(TaskStatus.Todo, TaskStatus.Doing)).toBe(true)
    expect(isAllowedStatusTransition(TaskStatus.Done, TaskStatus.Cancelled)).toBe(false)
  })
})
