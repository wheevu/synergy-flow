-- +goose Up
update users set name='Nick Robbins', updated_at=now() where id='00000000-0000-0000-0000-000000000001';

update project_columns
set name='In Review'
where id='30000000-0000-0000-0000-000000000004' and name='Review';

insert into users(id,name,email,password_hash) values
('00000000-0000-0000-0000-000000000006','Mina Patel','mina@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000007','Owen Brooks','owen@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK')
on conflict do nothing;

insert into workspace_members(workspace_id,user_id,role) values
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000006','Member'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000007','Member')
on conflict do nothing;

insert into tasks(id,project_id,column_id,title,description,priority,assignee_id,due_date,labels,created_by,position) values
('40000000-0000-0000-0000-000000000025','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Map onboarding checklist','Outline the first-run workspace setup flow and success criteria.','Medium','00000000-0000-0000-0000-000000000006',now()+interval '9 days',array['ux','frontend'],'00000000-0000-0000-0000-000000000001',4),
('40000000-0000-0000-0000-000000000026','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Define billing handoff events','List webhook events and retry rules for future subscription work.','Low','00000000-0000-0000-0000-000000000007',now()+interval '14 days',array['api','backend'],'00000000-0000-0000-0000-000000000001',5),
('40000000-0000-0000-0000-000000000027','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','Normalize drag reorder payloads','Keep client positions aligned with server-side dense ordering.','Urgent','00000000-0000-0000-0000-000000000001',now()+interval '1 day',array['frontend','api'],'00000000-0000-0000-0000-000000000001',4),
('40000000-0000-0000-0000-000000000028','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','Add member invite copy','Write short helper text for pending invites and expired token states.','Medium','00000000-0000-0000-0000-000000000006',now()+interval '5 days',array['email','ux'],'00000000-0000-0000-0000-000000000001',5),
('40000000-0000-0000-0000-000000000029','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000003','Implement drawer autosave guard','Prevent accidental closes from dropping edited task metadata.','High','00000000-0000-0000-0000-000000000007',now()+interval '3 days',array['frontend','ux'],'00000000-0000-0000-0000-000000000001',3),
('40000000-0000-0000-0000-000000000030','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000003','Backfill activity event actors','Repair imported activity rows that are missing actor metadata.','Medium','00000000-0000-0000-0000-000000000003',now()+interval '6 days',array['postgres','backend'],'00000000-0000-0000-0000-000000000001',4),
('40000000-0000-0000-0000-000000000031','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000004','Review realtime disconnect copy','Confirm reconnect language is clear in flaky network states.','Medium','00000000-0000-0000-0000-000000000006',now()+interval '2 days',array['realtime','ux'],'00000000-0000-0000-0000-000000000001',4),
('40000000-0000-0000-0000-000000000032','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000004','Validate project creation defaults','Make sure new projects include all expected board columns.','High','00000000-0000-0000-0000-000000000007',now()+interval '2 days',array['api','frontend'],'00000000-0000-0000-0000-000000000001',5),
('40000000-0000-0000-0000-000000000033','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000005','Ship production env example','Document required environment variables for API, web, storage, and email.','Medium','00000000-0000-0000-0000-000000000006',now()-interval '1 day',array['devops','docs'],'00000000-0000-0000-0000-000000000001',3),
('40000000-0000-0000-0000-000000000034','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000005','Close launch-readiness sweep','Record final launch risks and owner follow-ups.','High','00000000-0000-0000-0000-000000000001',now()-interval '4 hours',array['risk','ux'],'00000000-0000-0000-0000-000000000001',4)
on conflict do nothing;

-- +goose Down
delete from tasks where id between '40000000-0000-0000-0000-000000000025' and '40000000-0000-0000-0000-000000000034';
delete from workspace_members where user_id in ('00000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000007');
delete from users where id in ('00000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000007');
update project_columns set name='Review' where id='30000000-0000-0000-0000-000000000004' and name='In Review';
update users set name='Demo Owner', updated_at=now() where id='00000000-0000-0000-0000-000000000001';
