-- +goose Up
insert into users(id,name,email,password_hash) values
('00000000-0000-0000-0000-000000000001','Nick Robbins','demo@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000002','Avery Stone','avery@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000003','Priya Chen','priya@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000004','Mateo Rivera','mateo@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000005','Jordan Lee','jordan@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000006','Mina Patel','mina@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000007','Owen Brooks','owen@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK')
on conflict (id) do update set name=excluded.name,email=excluded.email;

insert into workspace_members(workspace_id,user_id,role) values
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','Owner'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','Member'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003','Admin'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000004','Member'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000005','Viewer'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000006','Member'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000007','Member')
on conflict (workspace_id,user_id) do update set role=excluded.role;

update workspace_members set joined_at = case user_id
  when '00000000-0000-0000-0000-000000000001' then '2026-01-08 10:00:00+00'::timestamptz
  when '00000000-0000-0000-0000-000000000002' then '2026-01-10 10:00:00+00'::timestamptz
  when '00000000-0000-0000-0000-000000000003' then '2026-01-12 10:00:00+00'::timestamptz
  when '00000000-0000-0000-0000-000000000004' then '2026-01-15 10:00:00+00'::timestamptz
  when '00000000-0000-0000-0000-000000000005' then '2026-01-18 10:00:00+00'::timestamptz
  when '00000000-0000-0000-0000-000000000006' then '2026-01-20 10:00:00+00'::timestamptz
  when '00000000-0000-0000-0000-000000000007' then '2026-01-22 10:00:00+00'::timestamptz
  else joined_at end
where workspace_id='10000000-0000-0000-0000-000000000001';

insert into activity_events(workspace_id,project_id,actor_id,event_type,metadata,created_at) values
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000005','workspace.reviewed','{"note":"Reviewed launch readiness risks"}',now()-interval '4 days')
on conflict do nothing;

-- +goose Down
delete from workspace_members where workspace_id='10000000-0000-0000-0000-000000000001' and user_id in ('00000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000007');
