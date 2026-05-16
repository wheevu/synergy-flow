-- +goose Up
-- Additional indexes for hot query paths identified during production audit.

-- Speed up task comment retrieval for the task detail drawer
create index if not exists idx_task_comments_task_created on task_comments(task_id, created_at);

-- Speed up attachment listing per task
create index if not exists idx_attachments_task on task_attachments(task_id);

-- Speed up workspace member queries for permission checks (composite index)
create index if not exists idx_workspace_members_lookup on workspace_members(workspace_id, user_id, role);

-- Speed up unique column position lookups (already has unique constraint, but an index helps FK joins)
create index if not exists idx_project_columns_project on project_columns(project_id, position);

-- Speed up email job worker queries
create index if not exists idx_email_jobs_unsent on email_jobs(sent_at, attempts) where sent_at is null;

-- Speed up recent activity per project
create index if not exists idx_activity_project on activity_events(project_id, created_at desc);

-- Speed up task count per column (used by move task position clamping)
create index if not exists idx_tasks_column on tasks(column_id);

-- +goose Down
drop index if exists idx_task_comments_task_created;
drop index if exists idx_attachments_task;
drop index if exists idx_workspace_members_lookup;
drop index if exists idx_project_columns_project;
drop index if exists idx_email_jobs_unsent;
drop index if exists idx_activity_project;
drop index if exists idx_tasks_column;
