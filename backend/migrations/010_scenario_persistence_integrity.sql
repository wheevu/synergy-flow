-- +goose Up
-- Scenario snapshots are captured under PostgreSQL REPEATABLE READ in the
-- application. These constraints make the denormalized project_id on each
-- dependency agree with both endpoint tasks, including for direct SQL writes.
do $$
begin
  if not exists (
    select 1 from pg_constraint
    where conrelid = 'tasks'::regclass
      and conname = 'tasks_project_id_id_unique'
  ) then
    alter table tasks
      add constraint tasks_project_id_id_unique unique (project_id, id);
  end if;
end $$;

do $$
begin
  if not exists (
    select 1 from pg_constraint
    where conrelid = 'task_dependencies'::regclass
      and conname = 'task_dependencies_predecessor_project_fk'
  ) then
    alter table task_dependencies
      add constraint task_dependencies_predecessor_project_fk
      foreign key (project_id, predecessor_task_id)
      references tasks(project_id, id) on delete cascade
      not valid;
  end if;
  if not exists (
    select 1 from pg_constraint
    where conrelid = 'task_dependencies'::regclass
      and conname = 'task_dependencies_successor_project_fk'
  ) then
    alter table task_dependencies
      add constraint task_dependencies_successor_project_fk
      foreign key (project_id, successor_task_id)
      references tasks(project_id, id) on delete cascade
      not valid;
  end if;
end $$;

-- Validation intentionally fails the migration if older rows are inconsistent;
-- silently rewriting dependency ownership would corrupt board semantics.
alter table task_dependencies
  validate constraint task_dependencies_predecessor_project_fk;
alter table task_dependencies
  validate constraint task_dependencies_successor_project_fk;

-- Ordinary UPDATE and DELETE statements remain forbidden. A DELETE caused by
-- the scenario foreign key's ON DELETE CASCADE runs inside a referential-action
-- trigger after the parent row is gone, and is the sole permitted delete path.
create or replace function prevent_scenario_event_mutation() returns trigger
language plpgsql as $$
begin
  if tg_op = 'DELETE'
    and pg_trigger_depth() > 1
    and not exists (
      select 1 from scenarios where id = old.scenario_id
    ) then
    return old;
  end if;
  raise exception 'scenario events are append-only';
end;
$$;

comment on function prevent_scenario_event_mutation() is
  'Blocks ordinary scenario event UPDATE/DELETE while allowing FK cascades after scenario deletion.';

-- Keep the separate filesystem watermark contract explicit at the database
-- boundary: it is diagnostic metadata, never a replay or snapshot boundary.
comment on column scenarios.base_event_id is
  'Advisory filesystem event-log watermark observed near a REPEATABLE READ PostgreSQL snapshot; not a transactional reproducibility boundary.';

-- +goose Down
alter table task_dependencies
  drop constraint if exists task_dependencies_successor_project_fk;
alter table task_dependencies
  drop constraint if exists task_dependencies_predecessor_project_fk;
alter table tasks drop constraint if exists tasks_project_id_id_unique;
