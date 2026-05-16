-- +goose Up
create extension if not exists pgcrypto;
create extension if not exists citext;

create table users (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  email citext unique not null,
  password_hash text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  refresh_token_hash text unique not null,
  user_agent text,
  ip_address text,
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now()
);

create table workspaces (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text unique not null,
  created_by uuid references users(id),
  created_at timestamptz not null default now()
);
create table workspace_members (
  workspace_id uuid references workspaces(id) on delete cascade,
  user_id uuid references users(id) on delete cascade,
  role text not null check (role in ('Owner','Admin','Member','Viewer')),
  joined_at timestamptz not null default now(),
  primary key(workspace_id,user_id)
);
create table workspace_invites (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid references workspaces(id) on delete cascade,
  email citext not null,
  role text not null check (role in ('Admin','Member','Viewer')),
  token text unique not null,
  created_by uuid references users(id),
  expires_at timestamptz not null,
  accepted_at timestamptz,
  created_at timestamptz not null default now()
);

create table projects (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  name text not null,
  description text not null default '',
  created_by uuid references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create table project_members (
  project_id uuid references projects(id) on delete cascade,
  user_id uuid references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key(project_id,user_id)
);
create table project_columns (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  name text not null,
  position int not null,
  unique(project_id, position)
);
create table tasks (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  column_id uuid not null references project_columns(id) on delete cascade,
  title text not null,
  description text not null default '',
  priority text not null default 'Medium' check (priority in ('Low','Medium','High','Urgent')),
  assignee_id uuid references users(id),
  due_date timestamptz,
  labels text[] not null default '{}',
  position int not null default 0,
  created_by uuid references users(id),
  search_vector tsvector generated always as (to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,''))) stored,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create table task_comments (
  id uuid primary key default gen_random_uuid(),
  task_id uuid not null references tasks(id) on delete cascade,
  author_id uuid not null references users(id),
  body text not null,
  created_at timestamptz not null default now()
);
create table task_attachments (
  id uuid primary key default gen_random_uuid(),
  task_id uuid not null references tasks(id) on delete cascade,
  uploader_id uuid not null references users(id),
  file_name text not null,
  content_type text not null,
  size_bytes bigint not null,
  storage_key text not null,
  created_at timestamptz not null default now()
);
create table activity_events (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references workspaces(id) on delete cascade,
  project_id uuid references projects(id) on delete cascade,
  actor_id uuid references users(id),
  event_type text not null,
  metadata jsonb not null default '{}',
  created_at timestamptz not null default now()
);
create table notifications (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  type text not null,
  title text not null,
  body text not null,
  read_at timestamptz,
  created_at timestamptz not null default now()
);
create table email_jobs (
  id uuid primary key default gen_random_uuid(),
  to_email citext not null,
  subject text not null,
  body text not null,
  attempts int not null default 0,
  last_error text,
  sent_at timestamptz,
  created_at timestamptz not null default now()
);

create index idx_sessions_user on sessions(user_id) where revoked_at is null;
create index idx_workspace_members_user on workspace_members(user_id);
create index idx_projects_workspace on projects(workspace_id);
create index idx_tasks_project_column on tasks(project_id,column_id,position);
create index idx_tasks_assignee on tasks(assignee_id);
create index idx_tasks_due_date on tasks(due_date);
create index idx_tasks_priority on tasks(priority);
create index idx_tasks_search on tasks using gin(search_vector);
create index idx_tasks_labels on tasks using gin(labels);
create index idx_activity_workspace on activity_events(workspace_id, created_at desc);
create index idx_notifications_user on notifications(user_id, read_at, created_at desc);

-- +goose Down
drop table if exists email_jobs, notifications, activity_events, task_attachments, task_comments, tasks, project_columns, project_members, projects, workspace_invites, workspace_members, workspaces, sessions, users cascade;
