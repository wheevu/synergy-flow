-- +goose Up
-- Scenario simulation is deliberately additive. Existing task rows receive a
-- bounded, safe estimate so older installations can opt into analysis without
-- rewriting their boards.
alter table tasks add column if not exists estimate_minutes integer;
update tasks
set estimate_minutes = 60
where estimate_minutes is null or estimate_minutes < 0;
alter table tasks alter column estimate_minutes set default 60;
alter table tasks alter column estimate_minutes set not null;
do $$
begin
  if not exists (
    select 1 from pg_constraint
    where conname = 'tasks_estimate_minutes_nonnegative'
  ) then
    alter table tasks add constraint tasks_estimate_minutes_nonnegative check (estimate_minutes >= 0);
  end if;
end $$;

create table if not exists task_dependencies (
  project_id uuid not null references projects(id) on delete cascade,
  predecessor_task_id uuid not null references tasks(id) on delete cascade,
  successor_task_id uuid not null references tasks(id) on delete cascade,
  created_by uuid references users(id),
  created_at timestamptz not null default now(),
  primary key (project_id, predecessor_task_id, successor_task_id),
  check (predecessor_task_id <> successor_task_id)
);
create index if not exists idx_task_dependencies_successor
  on task_dependencies(project_id, successor_task_id);

create table if not exists scenarios (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  name text not null,
  description text not null default '',
  created_by uuid not null references users(id),
  base_event_id bigint not null default 0,
  base_snapshot jsonb not null,
  base_digest text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index if not exists idx_scenarios_project_updated
  on scenarios(project_id, updated_at desc);

comment on column scenarios.base_event_id is
  'Advisory filesystem event-log watermark observed near snapshot capture; not a transactional reproducibility boundary.';

create table if not exists scenario_events (
  id uuid primary key default gen_random_uuid(),
  scenario_id uuid not null references scenarios(id) on delete cascade,
  sequence bigint not null,
  event_type text not null check (event_type in (
    'task_delay',
    'task_status_change',
    'task_assignee_change',
    'task_estimate_change',
    'dependency_add',
    'dependency_remove'
  )),
  payload jsonb not null,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  unique (scenario_id, sequence)
);
create index if not exists idx_scenario_events_scenario_sequence
  on scenario_events(scenario_id, sequence);

-- The captured board and digest are immutable. base_event_id is only an
-- advisory filesystem event-log watermark: the filesystem append cannot commit
-- atomically with this PostgreSQL snapshot transaction.
create or replace function prevent_scenario_base_mutation() returns trigger
language plpgsql as $$
begin
  if new.project_id <> old.project_id
    or new.base_event_id <> old.base_event_id
    or new.base_snapshot <> old.base_snapshot
    or new.base_digest <> old.base_digest then
    raise exception 'scenario base snapshot is immutable';
  end if;
  return new;
end;
$$;
drop trigger if exists scenarios_base_immutable on scenarios;
create trigger scenarios_base_immutable
before update on scenarios
for each row execute function prevent_scenario_base_mutation();

create or replace function prevent_scenario_event_mutation() returns trigger
language plpgsql as $$
begin
  raise exception 'scenario events are append-only';
end;
$$;
drop trigger if exists scenario_events_append_only on scenario_events;
create trigger scenario_events_append_only
before update or delete on scenario_events
for each row execute function prevent_scenario_event_mutation();

-- +goose Down
drop trigger if exists scenario_events_append_only on scenario_events;
drop function if exists prevent_scenario_event_mutation();
drop trigger if exists scenarios_base_immutable on scenarios;
drop function if exists prevent_scenario_base_mutation();
drop table if exists scenario_events;
drop table if exists scenarios;
drop index if exists idx_task_dependencies_successor;
drop table if exists task_dependencies;
alter table tasks drop constraint if exists tasks_estimate_minutes_nonnegative;
alter table tasks drop column if exists estimate_minutes;
