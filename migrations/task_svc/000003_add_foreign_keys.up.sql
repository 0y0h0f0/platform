-- Add foreign key constraints to enforce referential integrity within task_svc.
-- Operation logs (append-only audit table) deliberately omit FKs on nullable
-- project_id / task_id to tolerate dangling references for historical records.
-- +migrate Up

ALTER TABLE task_svc.project_members
    ADD CONSTRAINT fk_project_members_project
        FOREIGN KEY (project_id) REFERENCES task_svc.projects(id);

ALTER TABLE task_svc.tasks
    ADD CONSTRAINT fk_tasks_project
        FOREIGN KEY (project_id) REFERENCES task_svc.projects(id);

ALTER TABLE task_svc.task_comments
    ADD CONSTRAINT fk_task_comments_task
        FOREIGN KEY (task_id) REFERENCES task_svc.tasks(id);

-- +migrate Down

ALTER TABLE IF EXISTS task_svc.task_comments
    DROP CONSTRAINT IF EXISTS fk_task_comments_task;

ALTER TABLE IF EXISTS task_svc.tasks
    DROP CONSTRAINT IF EXISTS fk_tasks_project;

ALTER TABLE IF EXISTS task_svc.project_members
    DROP CONSTRAINT IF EXISTS fk_project_members_project;
