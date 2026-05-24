import type { InfiniteData } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'

import type { ListTasksData, Task } from '../../src/api/types'
import { upsertTaskInInfiniteData, type TaskListCacheFilters } from '../../src/queries/task.queries'
import { Priority, TaskStatus } from '../../src/utils/constants'

const task: Task = {
  id: 'task-1',
  project_id: 'project-web-console',
  title: '设计登录页',
  content: '完成登录、注册页面',
  status: TaskStatus.Todo,
  priority: Priority.High,
  assignee_id: 'user-1',
  creator_id: 'user-1',
  due_time: '',
  version: 1,
}

function createData(tasks: Task[]): InfiniteData<ListTasksData> {
  return {
    pageParams: [''],
    pages: [
      {
        next_cursor: '',
        tasks,
      },
    ],
  }
}

function createFilters(overrides: Partial<TaskListCacheFilters> = {}): TaskListCacheFilters {
  return {
    assigneeId: '',
    keyword: '',
    projectId: 'project-web-console',
    status: null,
    ...overrides,
  }
}

describe('task optimistic cache helpers', () => {
  it('replaces matching tasks inside infinite query pages', () => {
    const updatedTask = { ...task, title: '设计登录和注册页', version: 2 }
    const result = upsertTaskInInfiniteData(createData([task]), updatedTask, createFilters())

    expect(result?.pages[0].tasks).toEqual([updatedTask])
  })

  it('removes tasks that no longer match a filtered infinite query', () => {
    const updatedTask = { ...task, status: TaskStatus.Doing, version: 2 }
    const result = upsertTaskInInfiniteData(
      createData([task]),
      updatedTask,
      createFilters({ status: TaskStatus.Todo }),
    )

    expect(result?.pages[0].tasks).toEqual([])
  })

  it('inserts tasks that begin matching a cached filtered list', () => {
    const updatedTask = { ...task, status: TaskStatus.Doing, version: 2 }
    const result = upsertTaskInInfiniteData(
      createData([]),
      updatedTask,
      createFilters({ status: TaskStatus.Doing }),
    )

    expect(result?.pages[0].tasks).toEqual([updatedTask])
  })
})
