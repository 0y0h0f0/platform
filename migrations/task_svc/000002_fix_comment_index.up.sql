-- Fix: align idx_task_comments_task with comment pagination query.
-- The query sorts by (created_at, id) and pages forward from an anchor,
-- but the old index was on (task_id, id) without created_at.
-- This caused full scans over all comments for a task.

DROP INDEX IF EXISTS task_svc.idx_task_comments_task;
CREATE INDEX idx_task_comments_task ON task_svc.task_comments(task_id, created_at, id);
