-- +goose Up
insert into users(id,name,email,password_hash) values
('00000000-0000-0000-0000-000000000001','Nick Robbins','demo@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000002','Avery Stone','avery@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK')
on conflict do nothing;
insert into workspaces(id,name,slug,created_by) values ('10000000-0000-0000-0000-000000000001','Acme Product','acme-product','00000000-0000-0000-0000-000000000001') on conflict do nothing;
insert into workspace_members(workspace_id,user_id,role) values
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','Owner'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','Member') on conflict do nothing;
insert into projects(id,workspace_id,name,description,created_by) values ('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','Launch Roadmap','Public launch plan for SynergyFlow demo.','00000000-0000-0000-0000-000000000001') on conflict do nothing;
insert into project_columns(id,project_id,name,position) values
('30000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','Backlog',0),
('30000000-0000-0000-0000-000000000002','20000000-0000-0000-0000-000000000001','Todo',1),
('30000000-0000-0000-0000-000000000003','20000000-0000-0000-0000-000000000001','In Progress',2),
('30000000-0000-0000-0000-000000000004','20000000-0000-0000-0000-000000000001','In Review',3),
('30000000-0000-0000-0000-000000000005','20000000-0000-0000-0000-000000000001','Done',4) on conflict do nothing;
insert into tasks(id,project_id,column_id,title,description,priority,assignee_id,due_date,labels,created_by,position) values
('40000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','Wire invite email flow','Create Resend template and queued worker handoff.','High','00000000-0000-0000-0000-000000000002',now()+interval '2 days',array['backend','email'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000002','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000003','Tune Kanban movement transaction','Ensure drag/drop preserves dense ordering across columns.','Urgent','00000000-0000-0000-0000-000000000001',now()+interval '1 day',array['backend','postgres'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000003','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000005','Design workspace shell','Responsive sidebar, project switcher, and notification affordance.','Medium','00000000-0000-0000-0000-000000000002',now()-interval '1 day',array['frontend','ux'],'00000000-0000-0000-0000-000000000001',0) on conflict do nothing;
insert into task_comments(task_id,author_id,body) values ('40000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000002','Transaction is working in local testing; needs one more edge-case review.') on conflict do nothing;
insert into activity_events(workspace_id,project_id,actor_id,event_type,metadata) values ('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','task.created','{"task":"Tune Kanban movement transaction"}') on conflict do nothing;

-- +goose Down
truncate activity_events, task_comments, tasks, project_columns, projects, workspace_members, workspaces, users cascade;
