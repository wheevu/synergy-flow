-- +goose Up
-- Add a second project to make the dashboard and workspace feel populated.

-- Second project: Quarterly Alignment Accelerator
insert into projects(id,workspace_id,name,description,created_by) values
('20000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000001','Q3 Alignment Accelerator','Cross-functional initiative to reduce meeting gravity and improve stakeholder happiness.','00000000-0000-0000-0000-000000000001')
on conflict do nothing;

insert into project_columns(id,project_id,name,position) values
('30000000-0000-0000-0000-000000000010','20000000-0000-0000-0000-000000000002','Backlog',0),
('30000000-0000-0000-0000-000000000011','20000000-0000-0000-0000-000000000002','Todo',1),
('30000000-0000-0000-0000-000000000012','20000000-0000-0000-0000-000000000002','In Progress',2),
('30000000-0000-0000-0000-000000000013','20000000-0000-0000-0000-000000000002','In Review',3),
('30000000-0000-0000-0000-000000000014','20000000-0000-0000-0000-000000000002','Done',4)
on conflict do nothing;

insert into tasks(id,project_id,column_id,title,description,priority,assignee_id,due_date,labels,created_by,position) values
-- Backlog
('40000000-0000-0000-0000-000000000040','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000010','Reduce meeting gravity','Analyze calendar data and propose a meeting-light async culture.','Medium',NULL,now()+interval '30 days',array['ux','frontend'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000041','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000010','Calibrate executive aura','Define what executive presence means in async-first communication.','Low','00000000-0000-0000-0000-000000000005',now()+interval '45 days',array['design','ux'],'00000000-0000-0000-0000-000000000001',1),
-- Todo
('40000000-0000-0000-0000-000000000042','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000011','Audit stakeholder happiness','Survey top stakeholders and map satisfaction trends.','High','00000000-0000-0000-0000-000000000003',now()+interval '14 days',array['ux','api'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000043','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000011','Measure cross-functional entropy','Identify departments with highest context-switching overhead.','Medium','00000000-0000-0000-0000-000000000004',now()+interval '10 days',array['backend','postgres'],'00000000-0000-0000-0000-000000000001',1),
-- In Progress
('40000000-0000-0000-0000-000000000044','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000012','Build productivity theater dashboard','Create a real-time chart that shows how much work looks like work.','Urgent','00000000-0000-0000-0000-000000000001',now()+interval '3 days',array['frontend','realtime'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000045','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000012','Document synergy risk framework','Write the playbook for identifying alignment debt before it compounds.','High','00000000-0000-0000-0000-000000000006',now()+interval '7 days',array['docs','auth'],'00000000-0000-0000-0000-000000000001',1),
-- In Review
('40000000-0000-0000-0000-000000000046','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000013','Unblock deadline weather patterns','Analyze past project delays to predict future scheduling risks.','Medium','00000000-0000-0000-0000-000000000003',now()+interval '5 days',array['postgres','api'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000047','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000013','Review operational excellence metrics','Sign off on the operations dashboard before the quarterly review.','High','00000000-0000-0000-0000-000000000001',now()+interval '2 days',array['devops','aws'],'00000000-0000-0000-0000-000000000001',1),
-- Done
('40000000-0000-0000-0000-000000000048','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000014','Publish stakeholder turbulence report','The initial survey data has been compiled and shared with leadership.','Medium','00000000-0000-0000-0000-000000000002',now()-interval '5 days',array['design','ux'],'00000000-0000-0000-0000-000000000001',0),
('40000000-0000-0000-0000-000000000049','20000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000014','Ship revenue vibes dashboard','The revenue vibes indicator is live and showing positive alignment energy.','Medium','00000000-0000-0000-0000-000000000006',now()-interval '7 days',array['frontend','api'],'00000000-0000-0000-0000-000000000001',1)
on conflict do nothing;

-- Add some comments for the second project
insert into task_comments(task_id,author_id,body) values
('40000000-0000-0000-0000-000000000044','00000000-0000-0000-0000-000000000003','The productivity theater chart is going to be my magnum opus.'),
('40000000-0000-0000-0000-000000000045','00000000-0000-0000-0000-000000000004','Can we workshop the term "synergy risk" — it sounds like a LinkedIn headline.'),
('40000000-0000-0000-0000-000000000047','00000000-0000-0000-0000-000000000002','Metrics look good. One edge case on the uptime calculation needs a second look.')
on conflict do nothing;

-- Add activity events for the second project
insert into activity_events(workspace_id,project_id,actor_id,event_type,metadata) values
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','task.created','{"task":"Build productivity theater dashboard"}'),
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000003','comment.created','{"task":"Build productivity theater dashboard"}'),
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','task.moved','{"task":"Publish stakeholder turbulence report","to":"Done"}'),
('10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000006','task.updated','{"task":"Ship revenue vibes dashboard"}')
on conflict do nothing;

-- Add notifications for the second project workload
insert into notifications(user_id,type,title,body,resource_type,resource_id) values
('00000000-0000-0000-0000-000000000001','task.assigned','Urgent: Productivity theater dashboard','You have been assigned an urgent task for the Q3 Alignment Accelerator.','task','40000000-0000-0000-0000-000000000044'),
('00000000-0000-0000-0000-000000000006','task.assigned','Document synergy risk framework','Mina, please review the synergy risk playbook draft.','task','40000000-0000-0000-0000-000000000045'),
('00000000-0000-0000-0000-000000000003','comment.created','New comment on productivity theater','Priya commented on your urgent task.','task','40000000-0000-0000-0000-000000000044')
on conflict do nothing;

-- +goose Down
delete from notifications where resource_id in ('40000000-0000-0000-0000-000000000044','40000000-0000-0000-0000-000000000045');
delete from activity_events where project_id='20000000-0000-0000-0000-000000000002';
delete from task_comments where task_id between '40000000-0000-0000-0000-000000000040' and '40000000-0000-0000-0000-000000000049';
delete from tasks where project_id='20000000-0000-0000-0000-000000000002';
delete from project_columns where project_id='20000000-0000-0000-0000-000000000002';
delete from projects where id='20000000-0000-0000-0000-000000000002';
