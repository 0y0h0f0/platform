DROP INDEX IF EXISTS task_svc.idx_task_comments_task;
CREATE INDEX idx_task_comments_task ON task_svc.task_comments(task_id, id);
