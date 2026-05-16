-- +goose Up
insert into users(id,name,email,password_hash) values
('00000000-0000-0000-0000-000000000003','Priya Chen','priya@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000004','Mateo Rivera','mateo@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK'),
('00000000-0000-0000-0000-000000000005','Jordan Lee','jordan@synergyflow.dev','$2a$12$QtCsU9gT/gvpJtqo.A5Fxe43hO57JNQB2NEAgnnMmElDqiq0CR7cK')
on conflict do nothing;

insert into workspace_members(workspace_id,user_id,role) values
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003','Admin'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000004','Member'),
('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000005','Viewer')
on conflict do nothing;

insert into tasks(id,project_id,column_id,title,description,priority,assignee_id,due_date,labels,created_by,position) values
('40000000-0000-0000-0000-000000000010','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Draft permissions matrix','Document Owner/Admin/Member/Viewer access for workspaces and project actions.','High','00000000-0000-0000-0000-000000000003',now()+interval '6 days',array['auth','backend'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000011','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Design empty board state','Create calm empty states for new projects and filtered columns.','Medium','00000000-0000-0000-0000-000000000002',now()+interval '8 days',array['frontend','ux'],'00000000-0000-0000-0000-000000000001',1),
('40000000-0000-0000-0000-000000000012','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Add S3 lifecycle notes','Document attachment retention and bucket policy expectations for AWS deployment.','Low','00000000-0000-0000-0000-000000000004',now()+interval '12 days',array['aws','devops'],'00000000-0000-0000-0000-000000000001',2),
('40000000-0000-0000-0000-000000000013','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Plan audit log filters','Define workspace-level filters for actor, event type, and date range.','Medium','00000000-0000-0000-0000-000000000003',now()+interval '10 days',array['backend','api'],'00000000-0000-0000-0000-000000000001',3),
('40000000-0000-0000-0000-000000000014','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','Implement notification read states','Add mark-read and unread count behavior to the notification list.','Medium','00000000-0000-0000-0000-000000000004',now()+interval '4 days',array['frontend','api'],'00000000-0000-0000-0000-000000000001',1),
('40000000-0000-0000-0000-000000000015','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','Write deployment checklist','Capture EC2 firewall, Docker, DNS, Nginx, and TLS setup steps.','High','00000000-0000-0000-0000-000000000003',now()+interval '3 days',array['devops','aws'],'00000000-0000-0000-0000-000000000001',2),
('40000000-0000-0000-0000-000000000016','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000002','Tighten refresh token telemetry','Store user-agent and IP context for session review and revocation.','Urgent','00000000-0000-0000-0000-000000000001',now()+interval '2 days',array['auth','security'],'00000000-0000-0000-0000-000000000001',3),
('40000000-0000-0000-0000-000000000017','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000003','Wire SSE reconnect handling','Show subtle reconnect status and retry when project event stream drops.','High','00000000-0000-0000-0000-000000000002',now()+interval '5 days',array['realtime','frontend'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000018','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000003','Index task search filters','Validate query plans for assignee, priority, label, due date, and full text search.','High','00000000-0000-0000-0000-000000000003',now()+interval '4 days',array['postgres','backend'],'00000000-0000-0000-0000-000000000001',1),
('40000000-0000-0000-0000-000000000019','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000003','Polish mobile board scrolling','Ensure columns scroll predictably on narrow screens without trapping vertical scroll.','Medium','00000000-0000-0000-0000-000000000004',now()+interval '7 days',array['mobile','ux'],'00000000-0000-0000-0000-000000000001',2),
('40000000-0000-0000-0000-000000000020','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000004','Review attachment validation','Confirm file size, MIME type checks, and permission-gated uploads.','Urgent','00000000-0000-0000-0000-000000000001',now()+interval '1 day',array['aws','security'],'00000000-0000-0000-0000-000000000001',1),
('40000000-0000-0000-0000-000000000021','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000004','QA invite acceptance flow','Test invite token lifecycle, expired invites, and accepted membership creation.','High','00000000-0000-0000-0000-000000000004',now()+interval '2 days',array['auth','email'],'00000000-0000-0000-0000-000000000001',2),
('40000000-0000-0000-0000-000000000022','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000004','Review dashboard metrics','Check counts for completed, overdue, priority buckets, and active members.','Medium','00000000-0000-0000-0000-000000000003',now()+interval '3 days',array['api','ux'],'00000000-0000-0000-0000-000000000001',3),
('40000000-0000-0000-0000-000000000023','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000005','Add Docker health checks','Add service health checks and dependency ordering for reliable local startup.','Medium','00000000-0000-0000-0000-000000000003',now()-interval '2 days',array['devops','backend'],'00000000-0000-0000-0000-000000000001',1),
('40000000-0000-0000-0000-000000000024','20000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000005','Document resume bullets','Add architecture summary and interview talking points to README.','Low','00000000-0000-0000-0000-000000000002',now()-interval '3 days',array['docs','ux'],'00000000-0000-0000-0000-000000000001',2)
on conflict do nothing;

insert into task_comments(task_id,author_id,body) values
('40000000-0000-0000-0000-000000000017','00000000-0000-0000-0000-000000000003','Reconnect copy should be quiet and not distract from board work.'),
('40000000-0000-0000-0000-000000000020','00000000-0000-0000-0000-000000000004','Please include at least one oversized-file test case.'),
('40000000-0000-0000-0000-000000000015','00000000-0000-0000-0000-000000000001','Make sure the Nginx SSE buffering note is easy to find.')
on conflict do nothing;

insert into activity_events(workspace_id,project_id,actor_id,event_type,metadata) values
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003','task.created','{"task":"Wire SSE reconnect handling"}'),
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000004','comment.created','{"task":"Review attachment validation"}'),
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','task.moved','{"task":"Review dashboard metrics","to":"In Review"}')
on conflict do nothing;

insert into notifications(user_id,type,title,body) values
('00000000-0000-0000-0000-000000000003','task.assigned','Task assigned','You were assigned dashboard and deployment review tasks.'),
('00000000-0000-0000-0000-000000000004','comment.created','New comment','A teammate commented on an attachment validation task.')
on conflict do nothing;

-- +goose Down
delete from notifications where user_id in ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004');
delete from activity_events where actor_id in ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004');
delete from task_comments where task_id between '40000000-0000-0000-0000-000000000010' and '40000000-0000-0000-0000-000000000024';
delete from tasks where id between '40000000-0000-0000-0000-000000000010' and '40000000-0000-0000-0000-000000000024';
delete from workspace_members where user_id in ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000005');
delete from users where id in ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000005');
