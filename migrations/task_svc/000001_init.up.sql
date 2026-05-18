CREATE SCHEMA IF NOT EXISTS task_svc;

CREATE TABLE task_svc.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner_id UUID NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_projects_owner_name ON task_svc.projects(owner_id, name) WHERE deleted_at IS NULL;

CREATE TABLE task_svc.project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role SMALLINT NOT NULL DEFAULT 2,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_project_members_project_user ON task_svc.project_members(project_id, user_id);
CREATE UNIQUE INDEX idx_project_members_owner ON task_svc.project_members(project_id) WHERE role = 0;
CREATE INDEX idx_project_members_user ON task_svc.project_members(user_id, project_id);

CREATE TABLE task_svc.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 0,
    priority SMALLINT NOT NULL DEFAULT 1,
    assignee_id UUID,
    creator_id UUID NOT NULL,
    due_time TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 0,
    extra JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tasks_default ON task_svc.tasks(project_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status ON task_svc.tasks(project_id, status, created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_assignee ON task_svc.tasks(project_id, assignee_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_time ON task_svc.tasks(due_time) WHERE deleted_at IS NULL AND due_time IS NOT NULL;

CREATE TABLE task_svc.task_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_comments_task ON task_svc.task_comments(task_id, id);

CREATE TABLE task_svc.operation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID,
    task_id UUID,
    operator_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    detail JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_operation_logs_project ON task_svc.operation_logs(project_id, created_at DESC, id DESC);
CREATE INDEX idx_operation_logs_task ON task_svc.operation_logs(task_id, created_at DESC, id DESC);
