import { authHandlers } from './auth.handlers'
import { projectHandlers } from './project.handlers'
import { memberHandlers } from './member.handlers'
import { operationLogHandlers } from './operationLog.handlers'
import { commentHandlers } from './comment.handlers'
import { taskHandlers } from './task.handlers'

export const handlers = [
  ...authHandlers,
  ...projectHandlers,
  ...memberHandlers,
  ...operationLogHandlers,
  ...taskHandlers,
  ...commentHandlers,
]
