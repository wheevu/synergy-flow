import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { DragDropContext, Draggable, Droppable, DropResult } from '@hello-pangea/dnd';
import { Activity, Archive, BarChart3, Bell, BookOpen, Briefcase, Calendar, CalendarDays, Check, CheckCircle2, CheckSquare, ChevronDown, Clock, Cloud, Code, Command, Cpu, Database, Download, FileText, Flag, Folder, Globe, Layers, LayoutDashboard, Lock, LogOut, Mail, Menu, MessageSquare, PanelLeftClose, PanelLeftOpen, Paperclip, Plus, Search, Settings, Shield, Star, Tag, Target, Trash2, UserPlus, Users, Wifi, WifiOff, X, Zap } from 'lucide-react';
import './styles.css';
import { api, API_URL, Column, Project, Task, Workspace, AIAnalyzeResponse } from './api/client';
import { useAuthStore, useUIStore } from './lib/store';

const qc = new QueryClient();

type TaskDetail = { id: string; title: string; description: string; priority: string; status: string; labels?: string[]; due_date?: string; assignee_id?: string; assigneeIds?: string[] };
type BoardData = { columns: Column[] };
type MoveInput = { taskId: string; columnId: string; position: number };
type LocalMoveMap = Record<string, { columnId: string; position: number }>;
type InsightTab = 'risks' | 'activity';
type WorkspaceMember = { id: string; name: string; email: string; role: string; joined_at?: string; created_at?: string };
const TAG_OPTIONS = ['backend', 'frontend', 'ux', 'postgres', 'email', 'auth', 'devops', 'realtime', 'aws', 'api', 'mobile', 'design', 'security'];
const ROLE_ORDER: Record<string, number> = { Viewer: 1, Member: 2, Admin: 3, Owner: 4 };
const PROJECT_ICONS = ['Briefcase', 'Folder', 'Target', 'Zap', 'Database', 'Globe', 'Star', 'Lock', 'Mail', 'FileText', 'Calendar', 'CheckSquare', 'Cpu', 'Cloud', 'LayoutDashboard', 'Users', 'Shield', 'Flag', 'BarChart3', 'BookOpen', 'Code', 'Layers'] as const;
type ProjectIcon = typeof PROJECT_ICONS[number];
function projectIconComponent(name?: ProjectIcon | string, size = 14): React.ReactNode {
  const props = { size };
  switch (name) {
    case 'Folder': return <Folder {...props} />;
    case 'Target': return <Target {...props} />;
    case 'Zap': return <Zap {...props} />;
    case 'Database': return <Database {...props} />;
    case 'Globe': return <Globe {...props} />;
    case 'Star': return <Star {...props} />;
    case 'Lock': return <Lock {...props} />;
    case 'Mail': return <Mail {...props} />;
    case 'FileText': return <FileText {...props} />;
    case 'Calendar': return <Calendar {...props} />;
    case 'CheckSquare': return <CheckSquare {...props} />;
    case 'Cpu': return <Cpu {...props} />;
    case 'Cloud': return <Cloud {...props} />;
    case 'LayoutDashboard': return <LayoutDashboard {...props} />;
    case 'Users': return <Users {...props} />;
    case 'Shield': return <Shield {...props} />;
    case 'Flag': return <Flag {...props} />;
    case 'BarChart3': return <BarChart3 {...props} />;
    case 'BookOpen': return <BookOpen {...props} />;
    case 'Code': return <Code {...props} />;
    case 'Layers': return <Layers {...props} />;
    default: return <Briefcase {...props} />;
  }
}

function Login() {
  const auth = useAuthStore();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [form, setForm] = useState({ name: 'Nick Robbins', email: 'demo@synergyflow.dev', password: 'password123' });
  const mut = useMutation({
    mutationFn: async () => (await api.post(mode === 'login' ? '/auth/login' : '/auth/register', form)).data,
    onSuccess: (d) => { auth.setTokens(d.tokens); auth.setUser(d.user); },
  });

  const loginWithDemo = () => {
    setForm({ name: 'Nick Robbins', email: 'demo@synergyflow.dev', password: 'password123' });
    setMode('login');
    // Submit after a brief tick so the form state propagates.
    setTimeout(() => mut.mutate(), 50);
  };

  return (
    <div className="min-h-screen grid place-items-center bg-[#f6f7f9] p-4">
      <div className="card w-full max-w-md p-7">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">SynergyFlow</h1>
          <p className="text-sm text-gray-500 mt-1">Real-time project workspaces for focused teams.</p>
        </div>

        <div className="space-y-3">
          {mode === 'register' && (
            <input
              className="input"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Name"
            />
          )}
          <input
            className="input"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            placeholder="Email"
          />
          <input
            className="input"
            type="password"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            placeholder="Password"
          />

          <button className="btn btn-primary w-full" onClick={() => mut.mutate()}>
            {mode === 'login' ? 'Log in' : 'Create account'}
          </button>

          {mode === 'login' && (
            <button
              className="btn w-full text-sm border border-slate-300 hover:bg-slate-50"
              onClick={loginWithDemo}
            >
              Use demo account
            </button>
          )}

          {mut.error && (
            <p className="text-sm text-red-600">
              Authentication failed. If using seed data, run migrations first.
            </p>
          )}
        </div>

        <button
          className="mt-4 text-sm text-gray-600"
          onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
        >
          {mode === 'login' ? "Need an account? Register" : "Have an account? Log in"}
        </button>

        <p className="mt-6 text-xs text-gray-400 border-t border-slate-200 pt-4">
          Demo credentials: <span className="font-mono">demo@synergyflow.dev</span> / <span className="font-mono">password123</span>
        </p>
      </div>
    </div>
  );
}

function App() {
  const auth = useAuthStore();
  useEffect(() => { if (auth.accessToken) api.get('/me').then(r => auth.setUser(r.data.user)).catch(() => {}); }, [auth.accessToken]);

  const path = window.location.pathname;
  if (path.startsWith('/invite/') || path.startsWith('/invites/')) {
    const token = path.split('/')[2];
    return <InviteAccept token={token} />;
  }

  return auth.accessToken ? <Shell /> : <Login />;
}

function InviteAccept({ token }: { token: string }) {
  const ui = useUIStore();
  const auth = useAuthStore();
  const q = useQuery({ queryKey: ['invite', token], queryFn: async () => (await api.get(`/invites/${token}`)).data, retry: false });
  const accept = useMutation({ mutationFn: () => api.post(`/invites/${token}/accept`), onSuccess: () => { ui.toast({title: 'Invite accepted', tone: 'success'}); window.location.href = '/'; }});

  if (q.isLoading) return <div className="login-page"><p>Loading invite...</p></div>;
  if (q.isError) {
    const status = (q.error as any)?.response?.status;
    const title = status === 410 ? 'Invite expired' : status === 409 ? 'Invite already accepted' : 'Invite unavailable';
    const body = status === 410 ? 'This invite link has expired. Ask an admin to send a new invitation.' : status === 409 ? 'This invite has already been used. You can return to the app.' : 'This invite link may be invalid or revoked.';
    return <div className="login-page"><div className="login-card"><h2>{title}</h2><p className="login-subtitle">{body}</p><button className="btn btn-primary" style={{width:'100%', marginTop:'1rem'}} onClick={() => window.location.href = '/'}>Go to App</button></div></div>;
  }

  return <div className="login-page"><div className="login-card"><div className="brand-mark mx-auto mb-4" style={{width:'3rem',height:'3rem',fontSize:'1.5rem'}}>S</div><h2 className="text-center">You're invited</h2><p className="login-subtitle text-center">You’ve been invited to <strong>{q.data?.workspaceName}</strong> as <strong>{q.data?.role}</strong>.</p>
    {!auth.accessToken ? <div className="mt-4 p-3 bg-slate-50 border border-slate-200 rounded-lg text-sm text-center text-slate-600">Please <a href="/" className="text-blue-600 font-bold">log in</a> or create an account first to accept this invite.</div> : <button className="btn btn-primary mt-4" style={{width:'100%', height:'2.8rem'}} disabled={accept.isPending} onClick={() => accept.mutate()}>{accept.isPending ? 'Accepting...' : 'Accept Invite'}</button>}
  </div></div>;
}

type AiMessage = { role: 'user' | 'assistant'; content: string; signals?: { label: string; value: string; severity: 'low' | 'medium' | 'high' }[]; suggestedActions?: string[] };
const SUGGESTED_PROMPTS = ['Summarize project health', 'Find blockers', 'Suggest next actions', 'Review workload'];

function AiAnalystPanel({ projectId }: { projectId: string }) {
  const ui = useUIStore();
  const [messages, setMessages] = useState<AiMessage[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  useEffect(() => { scrollToBottom(); }, [messages, loading]);
  useEffect(() => { if (!ui.aiPanelOpen) return; const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') ui.setAIPanelOpen(false); }; window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey); }, [ui.aiPanelOpen, ui]);

  const send = async (prompt: string) => {
    if (!prompt.trim() || loading) return;
    const userMsg: AiMessage = { role: 'user', content: prompt.trim() };
    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setLoading(true);
    try {
      const res = await api.post<AIAnalyzeResponse>(`/projects/${projectId}/ai/analyze`, { prompt: prompt.trim() });
      const data = res.data;
      setMessages(prev => [...prev, { role: 'assistant', content: data.answer, signals: data.signals, suggestedActions: data.suggestedActions }]);
    } catch {
      setMessages(prev => [...prev, { role: 'assistant', content: 'I couldn\'t analyze the project right now. Please try again shortly.' }]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => { e.preventDefault(); send(input); };

  if (!ui.aiPanelOpen) return null;

  return <>
    <div className="ai-panel-backdrop" onClick={() => ui.setAIPanelOpen(false)} aria-hidden="true" />
    <div className="ai-panel" role="dialog" aria-modal="true" aria-label="AI Project Analyst">
      <div className="ai-panel-head">
        <div>
          <h2>Synergy AI</h2>
          <p>Project analyst</p>
        </div>
        <button className="ghost-icon" onClick={() => ui.setAIPanelOpen(false)} aria-label="Close AI panel"><X size={15} /></button>
      </div>
      <div className="ai-panel-messages scrollbar">
      {messages.length === 0 && (
        <div className="ai-empty">
          <Cpu size={28} className="ai-empty-icon" />
          <p><strong>Ask anything about this project</strong></p>
          <p className="ai-empty-hint">I can summarize health, find blockers, review workload, and suggest next actions.</p>
        </div>
      )}
      {messages.map((m, i) => (
        <div key={i} className={`ai-message ${m.role === 'user' ? 'ai-message-user' : 'ai-message-assistant'}`}>
          {m.role === 'assistant' && <div className="ai-avatar"><Cpu size={12} /></div>}
          <div className="ai-bubble">
            <p>{m.content}</p>
            {m.signals && m.signals.length > 0 && (
              <div className="ai-signals">
                {m.signals.map((s, idx) => (
                  <div key={idx} className={`ai-signal ai-signal-${s.severity}`}>
                    <span>{s.label}</span>
                    <strong>{s.value}</strong>
                  </div>
                ))}
              </div>
            )}
            {m.suggestedActions && m.suggestedActions.length > 0 && (
              <div className="ai-actions">
                <span className="ai-actions-label">Suggested actions</span>
                <ol>
                  {m.suggestedActions.map((a, idx) => <li key={idx}>{a}</li>)}
                </ol>
              </div>
            )}
          </div>
        </div>
      ))}
      {loading && (
        <div className="ai-message ai-message-assistant">
          <div className="ai-avatar"><Cpu size={12} /></div>
          <div className="ai-bubble">
            <div className="ai-loading"><span /><span /><span /></div>
          </div>
        </div>
      )}
      <div ref={bottomRef} />
    </div>
    <div className="ai-panel-chips">
      {SUGGESTED_PROMPTS.map(p => (
        <button key={p} className="ai-chip" onClick={() => send(p)} disabled={loading}>{p}</button>
      ))}
    </div>
    <form className="ai-panel-input" onSubmit={handleSubmit}>
      <input className="input" value={input} onChange={e => setInput(e.target.value)} placeholder="Ask about this project…" disabled={loading} />
      <button className="btn btn-primary" type="submit" disabled={!input.trim() || loading} aria-label="Send"><Zap size={14} /></button>
    </form>
  </div>
</>;
}

function Shell() {
  const auth = useAuthStore();
  const ui = useUIStore();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [insightTab, setInsightTab] = useState<InsightTab>('risks');
  const [railOpen, setRailOpen] = useState(true);
  const [workspaceView, setWorkspaceView] = useState<'dashboard' | 'members'>('dashboard');
  const [workspaceId, setWorkspaceId] = useState('');
  const [projectId, setProjectId] = useState('');
  const ws = useQuery({ queryKey: ['workspaces'], queryFn: async () => (await api.get<Workspace[]>('/workspaces')).data });
  useEffect(() => { if (!workspaceId && ws.data?.[0]) setWorkspaceId(ws.data[0].id); }, [ws.data, workspaceId]);
  const projects = useQuery({ queryKey: ['projects', workspaceId], enabled: !!workspaceId, queryFn: async () => (await api.get<Project[]>(`/workspaces/${workspaceId}/projects`)).data });
  const workspace = ws.data?.find(w => w.id === workspaceId);
  const project = projects.data?.find(p => p.id === projectId);
  useEffect(() => { const onKey = (e: KeyboardEvent) => { if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); ui.setCommandOpen(true); } }; window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey); }, [ui]);
  const sidebar = <Sidebar collapsed={collapsed} setCollapsed={setCollapsed} workspace={workspace} workspaces={ws.data || []} workspaceId={workspaceId} workspaceView={workspaceView} setWorkspaceView={(view: 'dashboard' | 'members') => { setWorkspaceView(view); setProjectId(''); setMobileSidebarOpen(false); }} setWorkspaceId={(id: string) => { setWorkspaceId(id); setProjectId(''); setWorkspaceView('dashboard'); setMobileSidebarOpen(false); }} projects={projects.data || []} projectId={projectId} setProjectId={(id: string) => { setProjectId(id); setWorkspaceView('dashboard'); setMobileSidebarOpen(false); }} logout={auth.logout} />;

  return <div className="h-screen flex overflow-hidden"><div className={`mobile-sidebar ${mobileSidebarOpen ? 'mobile-sidebar-open' : ''}`}><div className="mobile-sidebar-backdrop" onClick={() => setMobileSidebarOpen(false)} />{sidebar}</div>{sidebar}<main className="flex-1 min-w-0 flex flex-col">{!projectId && <header className="top-actions"><button className="mobile-menu-button" onClick={() => setMobileSidebarOpen(true)} aria-label="Open navigation"><Menu size={18} /></button><NotificationBell /><button className="command-trigger" onClick={() => ui.setCommandOpen(true)}><Command size={15} /><span>Command menu</span><kbd>⌘K</kbd></button><button className="settings-button" aria-label="Workspace settings" title="Workspace settings" onClick={() => ui.setSettingsOpen(true)}><Settings size={17} /></button></header>}{workspaceId && !projectId && workspaceView === 'dashboard' && <Dashboard workspaceId={workspaceId} workspace={workspace} />}{workspaceId && !projectId && workspaceView === 'members' && <MembersPageMvp workspaceId={workspaceId} workspace={workspace} />}{projectId && <Board projectId={projectId} project={project} workspace={workspace} insightTab={insightTab} railOpen={railOpen} setInsightTab={setInsightTab} setRailOpen={setRailOpen} onProjectDeleted={() => setProjectId('')} />}</main><TaskDrawer workspaceId={workspaceId} role={workspace?.role} /><CommandMenu workspaceId={workspaceId} projectId={projectId} projects={projects.data || []} onProject={setProjectId} onMembers={() => { setProjectId(''); setWorkspaceView('members'); }} /><WorkspaceSettings workspace={workspace} workspaceId={workspaceId} /><ToastStack />{projectId && <button className="ai-fab" aria-label="Open AI analyst" title="Ask Synergy AI" onClick={() => ui.setAIPanelOpen(true)}><Cpu size={20} /></button>}{projectId && ui.aiPanelOpen && <AiAnalystPanel projectId={projectId} />}</div>;
}

function Sidebar({ collapsed, setCollapsed, workspace, workspaces, workspaceId, workspaceView, setWorkspaceView, setWorkspaceId, projects, projectId, setProjectId, logout }: any) {
  return <aside className={`sidebar ${collapsed ? 'sidebar-collapsed' : 'sidebar-expanded'}`}><div className="sidebar-top"><div className="brand-mark">S</div><h2 className="sidebar-title">SynergyFlow</h2><button className="sidebar-toggle" title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} onClick={() => setCollapsed((v: boolean) => !v)}>{collapsed ? <PanelLeftOpen size={17} /> : <PanelLeftClose size={17} />}</button></div><WorkspaceSwitcher collapsed={collapsed} workspace={workspace} workspaces={workspaces} value={workspaceId} onChange={setWorkspaceId} /><nav className="sidebar-nav"><p className="sidebar-section-label">Workspace</p><NavItem collapsed={collapsed} icon={<LayoutDashboard size={16} />} label="Dashboard" active={!projectId && workspaceView === 'dashboard'} onClick={() => setWorkspaceView('dashboard')} /><NavItem collapsed={collapsed} icon={<Users size={16} />} label="Members" active={!projectId && workspaceView === 'members'} onClick={() => setWorkspaceView('members')} /></nav><div className="projects-block"><div className="projects-heading"><p>Projects</p><CreateProject workspaceId={workspaceId} collapsed={collapsed} /></div><div className="space-y-1">{projects.map((p: Project) => <button title={p.name} key={p.id} onClick={() => setProjectId(p.id)} className={`project-item project-item-card ${projectId === p.id ? 'project-active' : ''}`}><div className="project-avatar">{projectIconComponent((p as any).icon, 14)}</div><div className="project-copy"><p className="truncate text-sm font-semibold">{p.name}</p></div></button>)}</div></div><button className={`logout-link ${collapsed ? 'justify-center mt-auto' : ''}`} title="Log out" onClick={logout}><LogOut size={15} /><span className="sidebar-label">Log out</span></button></aside>;
}

function WorkspaceSwitcher({ collapsed, workspace, workspaces, value, onChange }: { collapsed: boolean; workspace?: Workspace; workspaces: Workspace[]; value: string; onChange: (id: string) => void }) {
  const initials = workspace?.name?.slice(0, 2).toUpperCase() || 'SF';
  return <label className={`workspace-switcher ${collapsed ? 'workspace-switcher-collapsed' : ''}`} title={workspace?.name}><div className="workspace-avatar">{initials}</div><div className="workspace-copy"><p className="truncate text-sm font-semibold">{workspace?.name || 'Workspace'}</p><p className="text-xs text-gray-500">{workspace?.role || 'Member'} workspace</p></div><ChevronDown size={15} className="workspace-chevron" /><select aria-label="Workspace" value={value} onChange={e => onChange(e.target.value)} className="workspace-native-visible themed-native-select">{workspaces.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}</select></label>;
}

function NavItem({ icon, label, active, collapsed, disabled, onClick }: { icon: React.ReactNode; label: string; active?: boolean; collapsed: boolean; disabled?: boolean; onClick?: () => void }) {
  return <button type="button" title={disabled ? `${label} coming soon` : label} disabled={disabled} onClick={onClick} className={`nav-button ${active ? 'nav-active' : ''} ${disabled ? 'nav-disabled' : ''} ${collapsed ? 'nav-collapsed' : ''}`}>{icon}<span className="nav-label">{label}</span>{disabled && <span className="coming-soon">Soon</span>}</button>;
}

function Dashboard({ workspaceId, workspace }: { workspaceId: string; workspace?: Workspace }) {
  const projects = useQuery({ queryKey: ['projects', workspaceId], enabled: !!workspaceId, queryFn: async () => (await api.get<Project[]>(`/workspaces/${workspaceId}/projects`)).data });
  const activeProject = projects.data?.[0];
  const board = useQuery({ queryKey: ['dashboard-board', activeProject?.id], enabled: !!activeProject?.id, queryFn: async () => (await api.get<BoardData>(`/projects/${activeProject?.id}/board`)).data, retry: 1 });
  const activity = useQuery({ queryKey: ['activity', workspaceId], enabled: !!workspaceId, queryFn: async () => (await api.get<any[]>(`/workspaces/${workspaceId}/activity`)).data, retry: 1 });
  const columns = board.data?.columns || [];
  const tasks = columns.flatMap(c => c.tasks.map(t => ({ ...t, status: c.name, columnId: c.id })));
  const summary = getBoardSummary(board.data);
  const completionRatio = summary.total ? Math.round((summary.completed / summary.total) * 100) : 0;
  const riskBreakdown = getRiskBreakdown(tasks, columns);
  const riskTasks = getRiskQueue(tasks, columns);
  const workload = getWorkloadSnapshot(tasks);
  const activityPulse = getActivityPulse(activity.data || []);
  const healthLabel = getHealthLabel(summary);
  const loading = projects.isLoading || (!!activeProject && board.isLoading);
  const error = projects.isError || board.isError;

  return (
    <section className="dashboard-page dashboard-overview">
      <div className="dashboard-hero">
        <div>
          <p className="board-breadcrumb"><span>{workspace?.name || 'Workspace'}</span><span>/</span><span>{activeProject?.name || 'Project overview'}</span></p>
          <h1>Dashboard</h1>
          <p>Project completion, workload, and risk signals from the live board.</p>
        </div>
        <span className="status-pill">{activeProject?.name || 'No project selected'}</span>
      </div>
      {loading && <div className="members-state"><BarChart3 size={22} /><strong>Loading dashboard…</strong><span>Fetching project tasks and activity.</span></div>}
      {error && <div className="members-state members-state-error"><WifiOff size={22} /><strong>Couldn’t load dashboard</strong><span>Check the project board endpoint and try again.</span><button className="btn" onClick={() => { projects.refetch(); board.refetch(); }}>Retry</button></div>}
      {!loading && !error && !activeProject && <div className="members-state"><Briefcase size={22} /><strong>No projects yet</strong><span>Create a project to see completion, workload, and risks here.</span></div>}
      {!loading && !error && activeProject && <>
        <div className="dashboard-kpi-grid">
          <Metric icon={<Layers size={18} />} label="Total tasks" value={summary.total} />
          <Metric icon={<CheckCircle2 size={18} />} label="Completed" value={summary.completed} />
          <Metric icon={<Clock size={18} />} label="Overdue" value={summary.overdue} />
          <Metric icon={<Flag size={18} />} label="Priority" value={summary.highPriority} />
          <Metric icon={<Users size={18} />} label="Unassigned" value={summary.unassigned} />
        </div>
        {summary.total === 0 ? (
          <div className="members-state"><CheckSquare size={22} /><strong>No tasks on this board</strong><span>Add tasks to build a dashboard overview.</span></div>
        ) : (
          <div className="dashboard-main-grid">
            <div className="dashboard-left-column">
              <section className="card dashboard-panel completion-card">
                <div className="dashboard-panel-head"><h2>Completion</h2><span>{completionRatio}%</span></div>
                <div className="completion-body">
                  <div className="completion-big"><strong>{completionRatio}%</strong><span>{summary.completed} of {summary.total} tasks completed</span></div>
                  <div className="health-meter"><span style={{ width: `${completionRatio}%` }} /></div>
                  <div className="completion-chips">
                    <div className="completion-chip"><strong>{summary.completed}/{summary.total}</strong><span>Completed</span></div>
                    <div className="completion-chip"><strong>{summary.overdue}</strong><span>Overdue</span></div>
                    <div className="completion-chip"><strong>{summary.highPriority}</strong><span>Urgent / High</span></div>
                  </div>
                  <div className="health-callout">
                    <span className={`health-pill health-pill-${healthLabel.tone}`}>{healthLabel.label}</span>
                    <p>{healthLabel.reason}</p>
                  </div>
                </div>
              </section>
              <section className="card dashboard-panel">
                <div className="dashboard-panel-head"><h2>Workload</h2><span>{workload.length} assignees</span></div>
                <WorkloadChart rows={workload} />
              </section>
              <section className="card dashboard-panel">
                <div className="dashboard-panel-head"><h2>Risk queue</h2><span>Top {Math.min(riskTasks.length, 5)}</span></div>
                <div className="risk-list">
                  {riskTasks.length ? riskTasks.map(t => (
                    <div className="risk-row" key={t.id}>
                      <span className={`priority priority-${String(t.priority || 'medium').toLowerCase()}`}><Flag size={11} />{t.priority || 'Medium'}</span>
                      <div><strong>{t.title || 'Untitled task'}</strong><small>{t.status} · {riskReason(t)}</small></div>
                    </div>
                  )) : <p className="empty-copy">No risky tasks right now.</p>}
                </div>
              </section>
            </div>
            <div className="dashboard-right-column">
              <section className="card dashboard-panel">
                <div className="dashboard-panel-head"><h2>Task status distribution</h2><span>Live board</span></div>
                <StatusDistribution columns={columns} />
              </section>
              <section className="card dashboard-panel">
                <div className="dashboard-panel-head"><h2>Risk composition</h2><span>{riskBreakdown.reduce((s, r) => s + r.value, 0)} signals</span></div>
                <RiskCompositionChart data={riskBreakdown} />
              </section>
              <section className="card dashboard-panel">
                <div className="dashboard-panel-head"><h2>Activity pulse</h2><span>Last 7 days</span></div>
                <ActivityPulseChart rows={activityPulse} />
                <div className="dashboard-list recent-activity-list">
                  {activity.isLoading ? <p className="empty-copy">Loading activity…</p> : activity.data?.length ? activity.data.slice(0, 4).map((a: any, i: number) => (
                    <div className="dashboard-list-row" key={a.id || i}>
                      <span className="activity-dot" />
                      <p><strong>{actorName(a.actor_id)}</strong> {activityText(a.event_type)}</p>
                      <small>{relativeTime(a.created_at)}</small>
                    </div>
                  )) : <p className="empty-copy">No recent activity yet.</p>}
                </div>
              </section>
            </div>
          </div>
        )}
      </>}
    </section>
  );
}
function MembersPage({ workspaceId, workspace }: { workspaceId: string; workspace?: Workspace }) {
  const qc = useQueryClient();
  const ui = useUIStore();
  const canAdmin = canRole(workspace?.role, 'Admin');
  const members = useQuery({ queryKey: ['members', workspaceId], queryFn: async () => (await api.get<WorkspaceMember[]>(`/workspaces/${workspaceId}/members`)).data });
  const invites = useQuery({ queryKey: ['invites', workspaceId], enabled: canAdmin, queryFn: async () => (await api.get<any[]>(`/workspaces/${workspaceId}/invites`)).data });
  const [invite, setInvite] = useState({ email: '', role: 'Member' });
  const updateRole = useMutation({ mutationFn: ({ uid, role }: { uid: string; role: string }) => api.patch(`/workspaces/${workspaceId}/members/${uid}`, { role }), onSuccess: () => { qc.invalidateQueries({ queryKey: ['members', workspaceId] }); ui.toast({ title: 'Role updated', tone: 'success' }); } });
  const remove = useMutation({ mutationFn: (uid: string) => api.delete(`/workspaces/${workspaceId}/members/${uid}`), onSuccess: () => { qc.invalidateQueries({ queryKey: ['members', workspaceId] }); ui.toast({ title: 'Member removed', tone: 'success' }); } });
  const sendInvite = useMutation({ mutationFn: () => api.post(`/workspaces/${workspaceId}/invites`, invite), onSuccess: (res) => { setInvite({ email: '', role: 'Member' }); qc.invalidateQueries({ queryKey: ['invites', workspaceId] }); ui.toast({ title: 'Invite sent', body: res.data.url, tone: 'success' }); } });

  return <section className="dashboard-page"><div className="dashboard-hero"><div><p className="board-breadcrumb">Workspace members</p><h1>Members</h1><p>Role badges, pending invites, invite status, role changes, and member removal for {workspace?.name || 'this workspace'}.</p></div><PermissionHint role={workspace?.role} need="Admin" action="invite, change roles, or remove members" /></div><div className="card p-4"><div className="member-management-head"><h2>Workspace members</h2>{canAdmin && <div className="invite-inline"><input className="input" placeholder="teammate@company.com" value={invite.email} onChange={e => setInvite({ ...invite, email: e.target.value })} /><select className="filter-select" value={invite.role} onChange={e => setInvite({ ...invite, role: e.target.value })}>{['Viewer', 'Member', 'Admin'].map(r => <option key={r}>{r}</option>)}</select><button className="btn btn-primary" disabled={!invite.email.trim() || sendInvite.isPending} onClick={() => sendInvite.mutate()}><UserPlus size={15} /> Invite</button></div>}</div><div className="members-table"><div className="members-table-row members-table-head"><span>Name</span><span>Email / status</span><span>Role</span><span>Actions</span></div>{members.isLoading && <div className="members-table-row"><span>Loading members…</span></div>}{members.data?.map(m => <div className="members-table-row" key={m.id}><span className="member-name-cell"><span className="mini-avatar avatar-slate">{initialsForName(displayMemberName(m.name))}</span><strong>{displayMemberName(m.name)}</strong></span><span>{m.email}</span><span><span className={`role-badge role-${m.role.toLowerCase()}`}>{m.role}</span></span><span>{canAdmin && m.role !== 'Owner' ? <><select className="filter-select" value={m.role} onChange={e => updateRole.mutate({ uid: m.id, role: e.target.value })}>{['Viewer', 'Member', 'Admin'].map(r => <option key={r}>{r}</option>)}</select><button className="btn" onClick={() => window.confirm(`Remove ${displayMemberName(m.name)}?`) && remove.mutate(m.id)}>Remove</button></> : <span className="empty-copy">Protected</span>}</span></div>)}{invites.data?.map(i => <div className="members-table-row" key={i.id}><span className="member-name-cell"><span className="mini-avatar avatar-slate">?</span><strong>{i.email}</strong></span><span><span className="status-pill">Pending invite</span> expires {new Date(i.expires_at).toLocaleDateString()}</span><span><span className={`role-badge role-${String(i.role).toLowerCase()}`}>{i.role}</span></span><span className="empty-copy">Awaiting acceptance</span></div>)}</div></div></section>;
}

function MembersPageMvp({ workspaceId, workspace }: { workspaceId: string; workspace?: Workspace }) {
  const qc = useQueryClient();
  const ui = useUIStore();
  const canAdmin = canRole(workspace?.role, 'Admin');
  const [inviteOpen, setInviteOpen] = useState(false);
  const [invite, setInvite] = useState({ email: '', role: 'Member' });
  const members = useQuery({ queryKey: ['members', workspaceId], queryFn: async () => (await api.get<WorkspaceMember[]>(`/workspaces/${workspaceId}/members`)).data, retry: 1 });
  const projects = useQuery({ queryKey: ['projects', workspaceId], enabled: !!workspaceId, queryFn: async () => (await api.get<Project[]>(`/workspaces/${workspaceId}/projects`)).data });
  const activeProject = projects.data?.[0];
  const board = useQuery({ queryKey: ['members-board', activeProject?.id], enabled: !!activeProject?.id, queryFn: async () => (await api.get<BoardData>(`/projects/${activeProject?.id}/board`)).data, retry: 1 });
  const activity = useQuery({ queryKey: ['activity', workspaceId], enabled: !!workspaceId, queryFn: async () => (await api.get<any[]>(`/workspaces/${workspaceId}/activity`)).data, retry: 1 });
  const tasks = (board.data?.columns || []).flatMap(c => c.tasks.map(t => ({ ...t, status: c.name, columnId: c.id })));
  const roster = buildWorkspaceRoster(members.data || [], tasks, activity.data || [], workspace?.role);
  const roleCounts = countRoles(roster);
  const activeAssignees = roster.filter(m => m.openCount > 0).length;
  const openAssignedTasks = tasks.filter(t => assigneeIdsForTask(t).length > 0 && !isDoneStatus(t.status)).length;
  const sendInvite = useMutation({ mutationFn: () => api.post(`/workspaces/${workspaceId}/invites`, invite), onSuccess: (res) => { setInvite({ email: '', role: 'Member' }); setInviteOpen(false); qc.invalidateQueries({ queryKey: ['invites', workspaceId] }); ui.toast({ title: 'Invite created', body: res.data?.url || 'Invite link is ready.', tone: 'success' }); }, onError: () => ui.toast({ title: 'Invite flow not configured in demo', body: 'The members page is ready; enable workspace invites on the API to send email links.', tone: 'info' }) });
  useEffect(() => { if (!inviteOpen) return; const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setInviteOpen(false); }; window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey); }, [inviteOpen]);
  const loading = members.isLoading || projects.isLoading || (!!activeProject && board.isLoading);
  const error = members.isError || board.isError;

  return <section className="dashboard-page members-page"><div className="dashboard-hero members-hero"><div><p className="board-breadcrumb">Workspace / Members</p><h1>Members</h1><p>Workspace access, roles, open workload, and recent collaboration signals for {workspace?.name || 'this workspace'}.</p></div>{canAdmin ? <button className="btn btn-primary" onClick={() => setInviteOpen(true)}><UserPlus size={15} /> Invite member</button> : <PermissionHint role={workspace?.role} need="Admin" action="invite new members" />}</div><div className="members-summary-row members-summary-wide"><Metric icon={<Users size={18} />} label="Total members" value={loading ? '—' : roster.length} /><Metric icon={<Shield size={18} />} label="Owners / admins" value={loading ? '—' : roleCounts.Owner + roleCounts.Admin} /><Metric icon={<CheckCircle2 size={18} />} label="Members / viewers" value={loading ? '—' : roleCounts.Member + roleCounts.Viewer} /><Metric icon={<Activity size={18} />} label="Active assignees" value={loading ? '—' : activeAssignees} /><Metric icon={<Layers size={18} />} label="Open assigned" value={loading ? '—' : openAssignedTasks} /></div><div className="members-admin-grid"><section className="card members-card"><div className="member-management-head"><div><h2>Workspace roster</h2><p>Members are merged with live board assignees so admin and workload views stay consistent.</p></div></div>{loading && <div className="members-state"><Users size={22} /><strong>Loading members…</strong><span>Fetching roster and assignment context.</span></div>}{error && <div className="members-state members-state-error"><WifiOff size={22} /><strong>Couldn’t load members</strong><span>Check members and board endpoints, then retry.</span><button className="btn" onClick={() => { members.refetch(); board.refetch(); }}>Retry</button></div>}{!loading && !error && roster.length === 0 && <div className="members-state"><Users size={22} /><strong>No members yet</strong><span>This workspace does not have any visible members.</span>{canAdmin && <button className="btn btn-primary" onClick={() => setInviteOpen(true)}><UserPlus size={15} /> Invite member</button>}</div>}{!loading && !error && roster.length > 0 && <div className="members-table members-table-expanded"><div className="members-table-row members-table-head"><span>Name</span><span>Email</span><span>Role</span><span>Joined</span><span>Open / total</span><span>Risk</span><span>Last activity</span></div>{roster.map(m => <div className="members-table-row" key={m.id}><span className="member-name-cell"><span className={`mini-avatar ${assigneeForId(m.id).className}`}>{initialsForName(m.name)}</span><strong>{m.name}</strong></span><span className="member-email">{m.email || 'No email available'}</span><span><span className={`role-badge role-${m.role.toLowerCase()}`}>{m.role}</span></span><span className="member-date">{formatMemberDate(m.joined_at || m.created_at)}</span><span>{m.openCount} / {m.totalCount}</span><span className={m.riskCount ? 'text-red-600 font-semibold' : 'text-slate-500'}>{m.riskCount}</span><span className="member-date">{m.lastActivity ? relativeTime(m.lastActivity) : 'No recent activity'}</span></div>)}</div>}</section><aside className="members-side"><section className="card dashboard-panel"><div className="dashboard-panel-head"><h2>Role distribution</h2><span>Access mix</span></div><RoleDistributionChart counts={roleCounts} /></section><section className="card dashboard-panel"><div className="dashboard-panel-head"><h2>Workload preview</h2><span>Open work · {activeProject?.name || 'Board'}</span></div><div className="workload-list">{roster.filter(m => m.openCount > 0).sort((a,b) => b.openCount - a.openCount || b.riskCount - a.riskCount).map(m => <div className="workload-row" key={m.id}><span className={`mini-avatar ${assigneeForId(m.id).className}`}>{initialsForName(m.name)}</span><div><strong>{m.name}</strong><small>{m.openCount} open / {m.totalCount} total · {m.riskCount} risk</small></div><b>{m.openCount}</b></div>)}</div></section></aside></div>{inviteOpen && <div className="modal-backdrop" role="dialog" aria-modal="true" aria-label="Invite member" onClick={() => setInviteOpen(false)}><div className="modal-card invite-modal" onClick={e => e.stopPropagation()}><div className="invite-modal-head"><div><h2>Invite member</h2><p>Add someone to {workspace?.name || 'this workspace'}.</p></div><button className="drawer-close" onClick={() => setInviteOpen(false)} aria-label="Close invite modal"><X size={18} /></button></div><div className="demo-note"><Mail size={15} /><span>If email delivery is not configured, this demo will create an invite link when the API allows it. Failures are shown as a clear demo message.</span></div><label><span className="field-label">Email</span><input className="input" placeholder="teammate@company.com" value={invite.email} onChange={e => setInvite({ ...invite, email: e.target.value })} /></label><label><span className="field-label">Role</span><select className="input themed-select" value={invite.role} onChange={e => setInvite({ ...invite, role: e.target.value })}>{['Viewer', 'Member', 'Admin'].map(r => <option key={r}>{r}</option>)}</select></label><div className="invite-modal-actions"><button className="btn" onClick={() => setInviteOpen(false)}>Cancel</button><button className="btn btn-primary" disabled={!invite.email.trim() || sendInvite.isPending} onClick={() => sendInvite.mutate()}><UserPlus size={15} /> {sendInvite.isPending ? 'Creating…' : 'Send invite'}</button></div></div></div>}</section>;
}

function Metric(p: { icon: React.ReactNode; label: string; value: any; compact?: boolean }) { return <div className={`card metric-card ${p.compact ? 'metric-card-compact' : 'p-4'} flex items-center gap-3`}><div className="h-9 w-9 rounded-lg bg-gray-100 grid place-items-center">{p.icon}</div><div><p className="text-xs text-gray-500">{p.label}</p><p className="text-xl font-semibold">{p.value}</p></div></div>; }

function NotificationBell() {
  const qc = useQueryClient();
  const ui = useUIStore();
  const q = useQuery({ queryKey: ['notifications'], queryFn: async () => (await api.get<any[]>('/notifications')).data, refetchInterval: 30000 });
  const markRead = useMutation({ mutationFn: () => api.post('/notifications/read'), onSuccess: () => qc.invalidateQueries({queryKey: ['notifications']}) });
  const [open, setOpen] = useState(false);
  const unreadCount = q.data?.filter(n => !n.read_at).length || 0;
  
  return <div className="relative">
    <button className={`settings-button notification-button ${unreadCount > 0 ? 'notification-button-unread' : ''}`} aria-label="Notifications" onClick={() => setOpen(!open)} style={{ position: 'relative' }}>
      <Bell size={16} />
      {unreadCount > 0 && <span className="notification-badge">{unreadCount > 9 ? '9+' : unreadCount}</span>}
    </button>
    {open && <><div className="fixed inset-0 z-30" onClick={() => setOpen(false)} /><div className="absolute right-0 mt-2 w-80 bg-white border border-slate-200 rounded-lg shadow-xl z-40 flex flex-col" style={{ maxHeight: '24rem', top: '100%' }}>
      <div className="flex items-center justify-between p-3 border-b border-slate-100">
        <h3 className="font-bold text-slate-800 text-sm">Notifications</h3>
        {unreadCount > 0 && <button className="text-xs text-blue-600 hover:text-blue-800 font-medium" onClick={() => markRead.mutate()}>Mark all read</button>}
      </div>
      <div className="overflow-auto p-1 flex-1">
        {q.data?.length === 0 ? <p className="text-center text-slate-500 text-sm py-4">No notifications</p> : q.data?.map(n => <div key={n.id} className={`p-3 border-b border-slate-50 last:border-0 hover:bg-slate-50 cursor-pointer ${!n.read_at ? 'bg-blue-50/30' : ''}`} onClick={() => { if (n.resource_type === 'task' && n.resource_id) ui.openTask(n.resource_id); setOpen(false); }}>
          <strong className="block text-sm text-slate-800">{n.title}</strong>
          <span className="block text-xs text-slate-600 mt-0.5 leading-snug">{n.body}</span>
          <span className="block text-[0.65rem] text-slate-400 mt-1">{new Date(n.created_at).toLocaleString()}{n.resource_type === 'task' ? ' · Open task' : ''}</span>
        </div>)}
      </div>
    </div></>}
  </div>;
}

function Board({ projectId, project, workspace, insightTab, railOpen, setInsightTab, setRailOpen, onProjectDeleted }: { projectId: string; project?: Project; workspace?: Workspace; insightTab: InsightTab; railOpen: boolean; setInsightTab: (mode: InsightTab) => void; setRailOpen: (open: boolean) => void; onProjectDeleted: () => void }) {
  const qclient = useQueryClient();
  const currentUser = useAuthStore(s => s.user);
  const ui = useUIStore();
  const [search, setSearch] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('');
  const [labelFilter, setLabelFilter] = useState('');
  const [assigneeFilter, setAssigneeFilter] = useState('');
  const [dueFilter, setDueFilter] = useState('');
  const [savedView, setSavedView] = useState('');
  const [streamState, setStreamState] = useState<'live' | 'reconnecting'>('live');
  const [localMoves, setLocalMoves] = useState<LocalMoveMap>({});
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const isDraggingRef = useRef(false);
  const isMovingRef = useRef(false);
  const queuedRefreshRef = useRef(false);
  const lastDragEndAtRef = useRef(0);
  const searchTerm = search.trim().toLowerCase();
  const queryKey = ['board', projectId] as const;
  const board = useQuery({ queryKey, queryFn: async () => (await api.get<BoardData>(`/projects/${projectId}/board`)).data });
  const members = useQuery({ queryKey: ['members', workspace?.id], enabled: !!workspace?.id, queryFn: async () => (await api.get<WorkspaceMember[]>(`/workspaces/${workspace?.id}/members`)).data });
  const canEditTasks = canRole(workspace?.role, 'Member');

  const flushBoardRefresh = () => {
    if (queuedRefreshRef.current) {
      queuedRefreshRef.current = false;
      qclient.invalidateQueries({ queryKey });
    }
  };

  useEffect(() => {
    if (projectId.length < 8) return;
    const token = useAuthStore.getState().accessToken;
    const es = new EventSource(`${API_URL}/projects/${projectId}/events?token=${encodeURIComponent(token || '')}`, { withCredentials: false });
    es.onopen = () => setStreamState('live');
    es.onerror = () => setStreamState('reconnecting');
    // SSE event payload is intentionally ignored: any valid event triggers a board cache
    // invalidation. The content of the JSON (type, projectId, actorId, data) is not parsed
    // by the frontend — it is purely a fire-and-forget signal.
    es.onmessage = () => {
      // Board events can arrive before the drop animation ends. Queue them during active drag/move
      // so the list is not re-measured from fresh server data mid-gesture.
      if (isDraggingRef.current || isMovingRef.current) {
        queuedRefreshRef.current = true;
        return;
      }
      qclient.invalidateQueries({ queryKey });
    };
    return () => es.close();
  }, [projectId, qclient]);

  const move = useMutation({
    mutationFn: async (v: MoveInput) => api.post(`/tasks/${v.taskId}/move`, v),
    onMutate: async (v) => {
      isMovingRef.current = true;
      await qclient.cancelQueries({ queryKey });
      const previous = qclient.getQueryData<BoardData>(queryKey);
      qclient.setQueryData<BoardData>(queryKey, current => current ? reorderBoard(current, v.taskId, v.columnId, v.position) : current);
      return { previous };
    },
    onError: (_err, _vars, ctx) => { if (ctx?.previous) qclient.setQueryData(queryKey, ctx.previous); ui.toast({ title: 'Couldn’t save changes', body: 'The card was moved back to its previous column.', tone: 'error' }); },
    onSettled: () => {
      isMovingRef.current = false;
      qclient.invalidateQueries({ queryKey });
      flushBoardRefresh();
    },
    onSuccess: () => ui.toast({ title: 'Task updated', body: 'Card position saved.', tone: 'success' }),
  });

  const normalizedBoard = useMemo(() => normalizeBoardData(board.data, localMoves), [board.data, localMoves]);

  const allTasks = useMemo(() => normalizedBoard.columns.flatMap(c => c.tasks.map(t => ({ ...t, status: c.name, columnId: c.id }))), [normalizedBoard]);
  const workload = useMemo(() => getWorkloadSnapshot(allTasks), [allTasks]);
  const summary = useMemo(() => getBoardSummary(normalizedBoard), [normalizedBoard]);
  const labels = useMemo(() => Array.from(new Set(allTasks.flatMap(t => labelsArray(t.labels)))).slice(0, 8), [allTasks]);
  const assignees = useMemo(() => Array.from(new Map(allTasks.flatMap(t => assigneeIdsForTask(t).map(id => [id, assigneeForId(id)] as const))).values()).slice(0, 5), [allTasks]);
  const priorities = ['Urgent', 'High', 'Medium', 'Low'];
  const filtered = useMemo<BoardData>(() => ({
    columns: normalizedBoard.columns.map(c => ({ ...c, tasks: c.tasks.filter(t => {
      const haystack = `${t.title} ${t.description} ${labelsArray(t.labels).join(' ')}`.toLowerCase();
      const matchesSearch = searchTerm ? haystack.includes(searchTerm) : true;
      const matchesPriority = priorityFilter ? String(t.priority || '').toLowerCase() === priorityFilter.toLowerCase() : true;
      const matchesLabel = labelFilter ? labelsArray(t.labels).includes(labelFilter) : true;
      const taskAssignees = assigneeIdsForTask(t);
      const matchesAssignee = assigneeFilter === '__unassigned' ? taskAssignees.length === 0 : assigneeFilter === '__mine' ? !!currentUser?.id && taskAssignees.includes(currentUser.id) : assigneeFilter ? taskAssignees.includes(assigneeFilter) : true;
      const matchesDue = dueFilter === 'overdue' ? !!t.dueDate && new Date(t.dueDate) < startOfToday() : true;
      return matchesSearch && matchesPriority && matchesLabel && matchesAssignee && matchesDue;
    }) })),
  }), [normalizedBoard, searchTerm, priorityFilter, labelFilter, assigneeFilter, dueFilter, currentUser?.id]);

  const totalVisible = filtered.columns.reduce((sum, col) => sum + col.tasks.length, 0);
  const filtersActive = !!(searchTerm || priorityFilter || labelFilter || assigneeFilter || dueFilter);
  const dragDisabled = filtersActive || move.isPending || !canEditTasks;
  const bulkPatch = useMutation({ mutationFn: async ({ ids, patch }: { ids: string[]; patch: any }) => Promise.all(ids.filter(id => !id.startsWith('demo-')).map(id => api.patch(`/tasks/${id}`, patch))), onSuccess: () => { setSelectedIds([]); qclient.invalidateQueries({ queryKey }); ui.toast({ title: 'Tasks updated', body: 'Bulk changes were applied.', tone: 'success' }); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Bulk update failed.', tone: 'error' }) });
  const bulkDelete = useMutation({ mutationFn: async (ids: string[]) => Promise.all(ids.filter(id => !id.startsWith('demo-')).map(id => api.delete(`/tasks/${id}`))), onSuccess: () => { setSelectedIds([]); qclient.invalidateQueries({ queryKey }); ui.toast({ title: 'Tasks archived', body: 'Selected cards were removed from the board.', tone: 'success' }); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Archive/delete failed.', tone: 'error' }) });
  const bulkMove = useMutation({ mutationFn: async ({ ids, columnId }: { ids: string[]; columnId: string }) => Promise.all(ids.filter(id => !id.startsWith('demo-')).map((id, position) => api.post(`/tasks/${id}/move`, { taskId: id, columnId, position }))), onSuccess: () => { setSelectedIds([]); qclient.invalidateQueries({ queryKey }); ui.toast({ title: 'Status changed', body: 'Selected tasks moved to the chosen column.', tone: 'success' }); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Status change failed.', tone: 'error' }) });
  const toggleSelected = (id: string) => setSelectedIds(ids => ids.includes(id) ? ids.filter(x => x !== id) : [...ids, id]);

  if (board.isLoading) {
    return <section className="board-shell"><div className="board-header"><div><div className="h-4 w-32 bg-slate-200 rounded animate-pulse mb-2"></div><div className="h-8 w-48 bg-slate-200 rounded animate-pulse"></div></div></div><div className="board-content p-4 flex gap-4"><div className="w-72 h-96 bg-slate-100 rounded-lg animate-pulse"></div><div className="w-72 h-96 bg-slate-100 rounded-lg animate-pulse"></div><div className="w-72 h-96 bg-slate-100 rounded-lg animate-pulse"></div></div></section>;
  }

  const onDragEnd = (r: DropResult) => {
    isDraggingRef.current = false;
    lastDragEndAtRef.current = Date.now();
    if (!r.destination) { flushBoardRefresh(); return; }
    if (r.source.droppableId === r.destination.droppableId && r.source.index === r.destination.index) { flushBoardRefresh(); return; }
    if (r.draggableId.startsWith('demo-') || r.destination.droppableId === DEMO_REVIEW_COLUMN_ID || r.source.droppableId === DEMO_REVIEW_COLUMN_ID) {
      setLocalMoves(current => ({ ...current, [r.draggableId]: { columnId: r.destination!.droppableId, position: r.destination!.index } }));
      flushBoardRefresh();
      return;
    }
    move.mutate({ taskId: r.draggableId, columnId: r.destination.droppableId, position: r.destination.index });
  };

  return <DragClickGuard.Provider value={lastDragEndAtRef}><section className="board-shell"><div className="board-header"><div><div className="board-breadcrumb"><span>{workspace?.name || 'Workspace'}</span><span>/</span><span>{project?.name || 'Current project'}</span></div><div className="board-title-row"><div className="board-project-icon">{projectIconComponent((project as any)?.icon, 18)}</div><h1>{project?.name || 'Project board'}</h1></div></div><div className="board-header-actions"><UserChip user={currentUser} /><AvatarStack members={assignees} /><NotificationBell /><button className={`insights-toggle ${railOpen ? 'insights-toggle-active' : ''}`} onClick={() => setRailOpen(!railOpen)}><Activity size={16} /> Insights</button><button className="settings-button" aria-label="Workspace settings" title="Workspace settings" onClick={() => ui.setSettingsOpen(true)}><Settings size={17} /></button><button className="command-trigger board-command-trigger" onClick={() => ui.setCommandOpen(true)}><Command size={15} /><span>Command</span><kbd>⌘K</kbd></button></div></div><div className="board-summary-strip"><Metric icon={<CheckCircle2 size={16} />} label="Completed" value={`${summary.completed}/${summary.total}`} compact /><Metric icon={<Clock size={16} />} label="Overdue" value={summary.overdue} compact /><Metric icon={<Flag size={16} />} label="Priority" value={summary.highPriority} compact /><Metric icon={<Users size={16} />} label="Unassigned" value={summary.unassigned} compact /></div><div className="board-toolbar"><div className="search-field"><Search size={16} className="search-icon" /><input className="input search-input" placeholder="Search tasks, descriptions, or labels" value={search} onChange={e => setSearch(e.target.value)} /></div><div className="filter-chip-row"><select className="filter-select" onChange={e => { const v = e.target.value; if (v === 'urgent') { setSavedView('My urgent tasks'); setPriorityFilter('Urgent'); setAssigneeFilter('__mine'); setLabelFilter(''); setDueFilter(''); } else if (v === 'backend') { setSavedView('Backend review'); setLabelFilter('backend'); setPriorityFilter(''); setAssigneeFilter(''); setDueFilter(''); } else if (v === 'unassigned') { setSavedView('Unassigned'); setAssigneeFilter('__unassigned'); setPriorityFilter(''); setLabelFilter(''); setDueFilter(''); } else if (v === 'overdue') { setSavedView('Overdue'); setDueFilter('overdue'); setPriorityFilter(''); setLabelFilter(''); setAssigneeFilter(''); } e.target.value = ''; }} aria-label="Saved views" value=""><option value="">{savedView || 'Saved views'}</option><option value="urgent">My urgent tasks</option><option value="backend">Backend review</option><option value="unassigned">Unassigned</option><option value="overdue">Overdue</option></select><select className="filter-select" value={priorityFilter} onChange={e => setPriorityFilter(e.target.value)} aria-label="Priority filter"><option value="">All priorities</option>{priorities.map(p => <option key={p} value={p}>{p}</option>)}</select><select className="filter-select" value={labelFilter} onChange={e => setLabelFilter(e.target.value)} aria-label="Label filter"><option value="">All labels</option>{labels.map(l => <option key={l} value={l}>{l}</option>)}</select><select className="filter-select" value={assigneeFilter} onChange={e => setAssigneeFilter(e.target.value)} aria-label="Assignee filter"><option value="">All assignees</option><option value="__unassigned">Unassigned</option>{assignees.map(a => <option key={a.key} value={a.key}>{a.name}</option>)}</select><button className={`clear-filters ${assigneeFilter === '__mine' ? 'clear-filters-active' : ''}`} onClick={() => setAssigneeFilter(assigneeFilter === '__mine' ? '' : '__mine')}>My work</button>{filtersActive && <button className="clear-filters" onClick={() => { setSearch(''); setPriorityFilter(''); setLabelFilter(''); setAssigneeFilter(''); setDueFilter(''); setSavedView(''); }}>Clear</button>}<div className={`live-status ${streamState === 'live' ? 'live-status-ok' : 'live-status-warn'}`}>{streamState === 'live' ? <Wifi size={14} /> : <WifiOff size={14} />}<span>{streamState === 'live' ? 'Live' : 'Reconnecting…'}</span></div></div></div>{!canEditTasks && <div className="board-note"><Shield size={14} /> Viewer mode: editing, drag-and-drop, and bulk actions are disabled. Ask an admin for Member access.</div>}{filtersActive && <div className="board-note">Clear filters to drag cards. This keeps card positions aligned with the full board order.</div>}{selectedIds.length > 0 && <BulkBar count={selectedIds.length} columns={normalizedBoard.columns} members={members.data || []} disabled={!canEditTasks} onClear={() => setSelectedIds([])} onPriority={priority => bulkPatch.mutate({ ids: selectedIds, patch: { priority } })} onAssign={assigneeId => bulkPatch.mutate({ ids: selectedIds, patch: { assigneeId } })} onLabel={label => bulkPatch.mutate({ ids: selectedIds, patch: { labels: [label] } })} onStatus={columnId => bulkMove.mutate({ ids: selectedIds, columnId })} onArchive={() => bulkDelete.mutate(selectedIds)} />}
  <div className={`board-content ${railOpen ? 'board-content-with-insights' : ''}`}><DragDropContext onBeforeDragStart={() => { isDraggingRef.current = true; }} onDragEnd={onDragEnd}><div className="kanban-board scrollbar">{filtered.columns.map(col => <Droppable droppableId={col.id} key={col.id} isDropDisabled={!canEditTasks}>{(provided, snapshot) => <ColumnView column={col} projectId={projectId} provided={provided} isDraggingOver={snapshot.isDraggingOver} searchTerm={searchTerm} dragDisabled={dragDisabled} canEdit={canEditTasks} selectedIds={selectedIds} onSelect={toggleSelected} />}</Droppable>)}</div></DragDropContext>{railOpen && <BoardRail activeTab={insightTab} setActiveTab={setInsightTab} summary={summary} tasks={allTasks} workload={workload} onClose={() => setRailOpen(false)} />}{totalVisible === 0 && <div className="board-empty"><p>No tasks match the current view.</p><button className="btn" onClick={() => { setSearch(''); setPriorityFilter(''); setLabelFilter(''); setAssigneeFilter(''); setDueFilter(''); }}>Clear filters</button></div>}</div></section></DragClickGuard.Provider>;
}

const DragClickGuard = React.createContext<React.MutableRefObject<number> | null>(null);

function ColumnView({ column, projectId, provided, isDraggingOver, searchTerm, dragDisabled, canEdit, selectedIds, onSelect }: any) {
  const isEmpty = column.tasks.length === 0;
  return <div className={`kanban-column ${isDraggingOver ? 'column-drag-over' : ''}`}><div className="column-header"><div className="column-title-wrap"><span className={`column-status-dot ${columnTone(column.name)}`} /><h3>{column.name}</h3><span className="column-count">{column.tasks.length}</span></div></div><div ref={provided.innerRef} {...provided.droppableProps} className="kanban-task-list scrollbar">{column.tasks.map((t: Task, i: number) => <Draggable draggableId={t.id} index={i} key={t.id} isDragDisabled={dragDisabled}>{(p, snapshot) => <TaskCard task={t} provided={p} snapshot={snapshot} selected={selectedIds.includes(t.id)} onSelect={onSelect} canEdit={canEdit} />}</Draggable>)}{provided.placeholder}{isEmpty && <div className="column-empty">{searchTerm ? 'No matching tasks' : isDraggingOver ? 'Release to move here' : 'Drop tasks here'}</div>}</div><CreateColumnTask projectId={projectId} column={column} canEdit={canEdit} /></div>;
}

function TaskCard({ task, provided, snapshot, selected, onSelect, canEdit }: any) {
  const ui = useUIStore();
  const dragGuard = React.useContext(DragClickGuard);
  const labels = labelsArray(task.labels);
  const assignees = assigneesForTask(task);
  const assigneeLabel = assignees.length > 1 ? `${assignees[0].name} +${assignees.length - 1}` : assignees[0]?.name;
  
  let dueDateCopy = '';
  let isOverdue = false;
  if (task.dueDate) {
    const due = new Date(task.dueDate);
    const today = new Date();
    today.setHours(0,0,0,0);
    due.setHours(0,0,0,0);
    const diff = due.getTime() - today.getTime();
    if (diff < 0) { isOverdue = true; dueDateCopy = 'Overdue'; }
    else if (diff === 0) dueDateCopy = 'Today';
    else if (diff === 86400000) dueDateCopy = 'Tomorrow';
    else dueDateCopy = due.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }

  return <article ref={provided.innerRef} {...provided.draggableProps} {...provided.dragHandleProps} style={provided.draggableProps.style} onClick={() => { if (!snapshot.isDragging && Date.now() - (dragGuard?.current || 0) > 220 && !task.id.startsWith('demo-')) ui.openTask(task.id); }} className={`task-card ${snapshot.isDragging ? 'task-card-dragging' : ''} ${selected ? 'task-card-selected' : ''}`}><div className="task-card-top"><button type="button" className={`card-select-button ${selected ? 'card-select-button-on' : ''}`} onClick={e => { e.stopPropagation(); onSelect(task.id); }} disabled={!canEdit} title={canEdit ? 'Select for bulk actions' : 'Member role required'} aria-label={`Select ${task.title || 'task'} for bulk actions`}>{selected ? <CheckCircle2 size={13} /> : null}</button><span className={`priority priority-${String(task.priority || 'medium').toLowerCase()}`}><Flag size={11} />{task.priority || 'Medium'}</span><span className="task-card-actions" aria-hidden="true"><MessageSquare size={13} /><Paperclip size={13} /><Search size={13} /></span><span className="task-key">SF-{shortKey(task.id)}</span></div><h4>{task.title || 'Untitled task'}</h4>{task.description && <p className="task-desc">{task.description}</p>}<div className="task-tags">{labels.slice(0, 3).map((l: string) => <span className={`tag ${tagClass(l)}`} key={l}>{l}</span>)}</div><div className="task-meta-row"><div className="task-meta">{task.dueDate && <span className={isOverdue ? 'text-red-600 font-medium' : ''}><CalendarDays size={13} />{dueDateCopy}</span>}<span className={assignees.length ? '' : 'task-unassigned'}><Users size={13} />{assigneeLabel || 'Unassigned'}</span></div>{assignees.length > 0 && <span className="mini-avatar-group">{assignees.slice(0, 3).map(a => <span className={`mini-avatar ${a.className}`} title={a.name} key={a.key}>{a.initials}</span>)}</span>}</div></article>;
}

function startOfToday() {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d;
}

function TaskDrawer({ workspaceId, role }: { workspaceId: string; role?: string }) {
  const ui = useUIStore();
  const qclient = useQueryClient();
  const q = useQuery({ queryKey: ['task', ui.taskId], enabled: !!ui.taskId, queryFn: async () => (await api.get(`/tasks/${ui.taskId}`)).data?.[0] as TaskDetail });
  const comments = useQuery({ queryKey: ['comments', ui.taskId], enabled: !!ui.taskId, queryFn: async () => (await api.get(`/tasks/${ui.taskId}/comments`)).data });
  const attachments = useQuery({ queryKey: ['attachments', ui.taskId], enabled: !!ui.taskId, queryFn: async () => (await api.get(`/tasks/${ui.taskId}/attachments`)).data });
  const members = useQuery({ queryKey: ['members', workspaceId], enabled: !!workspaceId, queryFn: async () => (await api.get<WorkspaceMember[]>(`/workspaces/${workspaceId}/members`)).data });
  const activity = useQuery({ queryKey: ['activity', workspaceId], enabled: !!workspaceId && !!ui.taskId && workspaceId.length > 8, queryFn: async () => (await api.get(`/workspaces/${workspaceId}/activity`)).data });
  const task = q.data;
  const [form, setForm] = useState({ title: '', description: '', priority: 'Medium', assigneeIds: [] as string[], labels: '', dueDate: '' });
  const [body, setBody] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [assigneeDropdownOpen, setAssigneeDropdownOpen] = useState(false);
  const selectedTags = labelsArray(form.labels);
  const canEdit = canRole(role, 'Member');
  const drawerRef = useRef<HTMLElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const setTags = (tags: string[]) => setForm({ ...form, labels: Array.from(new Set(tags.map(t => t.trim()).filter(Boolean))).join(', ') });
  const combinedMembers = useMemo(() => {
    const map = new Map<string, { id: string, name: string }>();
    if (members.data) members.data.forEach(m => map.set(m.name, { id: m.id, name: m.name }));
    ['u1', 'u2', 'u3', 'u4', 'u5', 'u6', 'u7'].forEach(k => {
      const p = assigneeForId(k);
      if (!map.has(p.name)) map.set(p.name, { id: k, name: p.name });
    });
    return Array.from(map.values());
  }, [members.data]);

  useEffect(() => { if (task) setForm({ title: task.title || '', description: task.description || '', priority: task.priority || 'Medium', assigneeIds: task.assigneeIds || (task.assignee_id ? [task.assignee_id] : []), labels: labelsArray(task.labels).join(', '), dueDate: task.due_date ? task.due_date.split('T')[0] : '' }); }, [task?.id, task?.title, task?.description, task?.priority, task?.assignee_id, JSON.stringify(task?.assigneeIds), JSON.stringify(task?.labels), task?.due_date]);
  const dirty = !!task && (form.title !== task.title || form.description !== task.description || form.priority !== task.priority || form.dueDate !== (task.due_date ? task.due_date.split('T')[0] : '') || form.assigneeIds.join(',') !== ((task.assigneeIds || (task.assignee_id ? [task.assignee_id] : [])).join(',')) || form.labels !== labelsArray(task.labels).join(', '));
  const hasUserInput = !!(form.title.trim() || form.description.trim() || selectedTags.length || form.assigneeIds.length || form.dueDate);
  const isBlankDraft = ui.draftTaskId === ui.taskId && !hasUserInput;
  
  const upload = useMutation({ mutationFn: (file: File) => { const formData = new FormData(); formData.append('file', file); return api.post(`/tasks/${ui.taskId}/attachments`, formData, { headers: { 'Content-Type': 'multipart/form-data' } }); }, onSuccess: () => { qclient.invalidateQueries({ queryKey: ['attachments', ui.taskId] }); ui.toast({ title: 'Attachment uploaded', tone: 'success' }); }, onError: () => ui.toast({ title: 'Upload failed', tone: 'error' }) });
  const delAttachment = useMutation({ mutationFn: (id: string) => api.delete(`/attachments/${id}`), onSuccess: () => qclient.invalidateQueries({ queryKey: ['attachments', ui.taskId] }) });
  const discardDraft = useMutation({ mutationFn: () => api.delete(`/tasks/${ui.taskId}`), onSettled: () => { qclient.invalidateQueries({ queryKey: ['board'] }); ui.closeTask(); } });
  const deleteTask = useMutation({ mutationFn: () => api.delete(`/tasks/${ui.taskId}`), onSuccess: () => { qclient.invalidateQueries({ queryKey: ['board'] }); ui.toast({ title: 'Task deleted', tone: 'success' }); ui.closeTask(); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Delete failed.', tone: 'error' }) });
  const closeDrawer = () => { if (isBlankDraft) { discardDraft.mutate(); return; } if (dirty && !window.confirm('Discard unsaved changes?')) return; ui.closeTask(); };
  useEffect(() => { if (!ui.taskId) return; const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') confirmDelete ? setConfirmDelete(false) : closeDrawer(); if (e.key === 'Tab' && drawerRef.current) trapFocus(e, drawerRef.current); }; window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey); }, [ui.taskId, dirty, confirmDelete]);
  const save = useMutation({ mutationFn: () => api.patch(`/tasks/${ui.taskId}`, { title: form.title.trim() || 'Untitled task', description: form.description, priority: form.priority, dueDate: form.dueDate || null, assigneeId: form.assigneeIds[0] || '', assigneeIds: form.assigneeIds, labels: selectedTags }), onSuccess: () => { ui.clearDraftTask(); qclient.invalidateQueries({ queryKey: ['task', ui.taskId] }); qclient.invalidateQueries({ queryKey: ['board'] }); qclient.invalidateQueries({ queryKey: ['activity', workspaceId] }); ui.toast({ title: 'Task updated', body: 'Changes saved.', tone: 'success' }); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Task update failed.', tone: 'error' }) });
  const add = useMutation({ mutationFn: () => api.post(`/tasks/${ui.taskId}/comments`, { body }), onSuccess: () => { setBody(''); qclient.invalidateQueries({ queryKey: ['comments', ui.taskId] }); ui.toast({ title: 'Comment added', tone: 'success' }); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Comment failed to post.', tone: 'error' }) });
  if (!ui.taskId) return null;
  
  if (q.isLoading) {
    return <><div className="drawer-backdrop" onClick={closeDrawer} /><aside className="drawer task-drawer fixed right-0 top-0 h-full w-full sm:w-[540px] bg-white border-l z-20"><div className="drawer-header"><div className="h-6 w-48 bg-slate-200 rounded animate-pulse"></div><button className="drawer-close" onClick={closeDrawer} aria-label="Close task"><X size={18} /></button></div><div className="drawer-body p-6"><div className="space-y-4"><div className="h-20 w-full bg-slate-100 rounded animate-pulse"></div><div className="h-10 w-full bg-slate-100 rounded animate-pulse"></div><div className="h-10 w-full bg-slate-100 rounded animate-pulse"></div></div></div></aside></>;
  }

  return <>
    <div className="drawer-backdrop" onClick={closeDrawer} />
    <aside ref={drawerRef} className="drawer task-drawer fixed right-0 top-0 h-full w-full sm:w-[540px] bg-white border-l z-20 overflow-y-auto" role="dialog" aria-modal="true">
      <div className="drawer-header"><input className="drawer-title-input" value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} placeholder="Task title" disabled={!canEdit} title={!canEdit ? 'Member role required to edit tasks' : undefined} /><div className="drawer-actions"><button className="drawer-delete" onClick={() => setConfirmDelete(true)} aria-label="Delete task" disabled={!canEdit} title={!canEdit ? 'Member role required to delete tasks' : undefined}><Trash2 size={16} /></button><button className="drawer-close" onClick={closeDrawer} aria-label="Close task"><X size={18} /></button></div></div>
      <div className="drawer-body">
        {isBlankDraft && <p className="draft-note">This new task will be discarded if you close it without adding details.</p>}
        {!canEdit && <PermissionHint role={role} need="Member" action="edit tasks, comment, or upload files" />}
        <label className="field-label">Description</label><textarea className="input" rows={4} value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} placeholder="Describe the work" disabled={!canEdit} />
        <div className="drawer-meta-row"><label><span className="field-label">Priority</span><span className="select-shell"><select className="input themed-select" value={form.priority} onChange={e => setForm({ ...form, priority: e.target.value })} disabled={!canEdit}>{['Low', 'Medium', 'High', 'Urgent'].map(p => <option key={p}>{p}</option>)}</select><ChevronDown size={15} /></span></label><label><span className="field-label">Due Date</span><input type="date" className="input" style={{ height: '2.65rem', padding: '0 0.5rem' }} value={form.dueDate} onChange={e => setForm({ ...form, dueDate: e.target.value })} disabled={!canEdit} /></label><div className="relative" style={{ minWidth: '11rem', position: 'relative' }}><span className="field-label">Assignees</span><button type="button" className="input flex items-center justify-between bg-white text-left" style={{ height: '2.65rem' }} disabled={!canEdit} onClick={() => setAssigneeDropdownOpen(!assigneeDropdownOpen)}><span className="flex items-center gap-1" style={{ overflow: 'hidden' }}>{form.assigneeIds.length === 0 ? 'Unassigned' : form.assigneeIds.slice(0, 2).map(id => { const person = assigneeForId(id); return <span key={id} className={`mini-avatar ${person.className}`} style={{ marginLeft: 0, height: '1.4rem', width: '1.4rem', fontSize: '0.55rem' }} title={person.name}>{person.initials}</span>; }).concat(form.assigneeIds.length > 2 ? [<span key="more" className="text-xs text-slate-500 ml-0.5">+{form.assigneeIds.length - 2}</span>] : [])}</span><ChevronDown size={15} className="text-slate-400" /></button>{assigneeDropdownOpen && <><div className="fixed inset-0 z-40" onClick={() => setAssigneeDropdownOpen(false)} /><div className="absolute left-0 mt-1 w-full bg-white border border-slate-200 rounded-lg shadow-lg z-50 py-1 max-h-48 overflow-auto" style={{ top: '100%' }}>{combinedMembers.map(m => { const isSelectedById = form.assigneeIds.includes(m.id); const isSelectedByName = form.assigneeIds.some(id => assigneeForId(id).name === m.name); const isSelected = isSelectedById || isSelectedByName; const person = assigneeForId(m.id); return <button key={m.id} type="button" className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2 ${isSelected ? 'bg-slate-50 text-blue-600 font-medium' : 'text-slate-700 hover:bg-slate-50'}`} onClick={() => { if (isSelected) { setForm({ ...form, assigneeIds: form.assigneeIds.filter(id => id !== m.id && assigneeForId(id).name !== m.name) }); } else { setForm({ ...form, assigneeIds: [...form.assigneeIds, m.id] }); } }}>{isSelected ? <Check size={14} className="text-blue-600 shrink-0" /> : <div className="w-[14px] shrink-0" />}<span className={`mini-avatar ${person.className}`} style={{ marginLeft: 0, height: '1.4rem', width: '1.4rem', fontSize: '0.55rem' }}>{person.initials}</span>{displayMemberName(m.name)}</button>; })}</div></>}</div><div><span className="field-label">Status</span><span className="status-pill">{task?.status || 'Status'}</span></div></div>
        <section className="tag-picker"><span className="field-label">Tags</span><div className="tag-options">{TAG_OPTIONS.map(tag => { const active = selectedTags.includes(tag); return <button key={tag} type="button" className={`tag-option ${active ? 'tag-option-active' : ''}`} disabled={!canEdit} onClick={() => setTags(active ? selectedTags.filter(t => t !== tag) : [...selectedTags, tag])}>{tag}</button>; })}</div></section>
        <div className="save-row"><span>{save.isError ? <span className="text-red-600">Failed to save changes</span> : dirty ? 'Unsaved changes' : save.isSuccess ? 'Saved just now' : 'Saved'}</span><button className="btn btn-primary" disabled={!dirty || save.isPending || !hasUserInput || !canEdit} title={!canEdit ? 'Member role required' : undefined} onClick={() => save.mutate()}>{save.isPending ? 'Saving...' : 'Save changes'}</button></div>
        <TaskActivity task={task} items={activity.data || []} />
        <section className="drawer-section"><h3><MessageSquare size={16} /> Comments</h3><div className="space-y-2">{comments.data?.length ? comments.data.map((c: any) => <div className="comment-card" key={c.id}><p>{c.body}</p><span>{c.author || 'Team member'}</span></div>) : <p className="empty-copy">No comments yet. Add a quick update for the team.</p>}</div><textarea className="input mt-3" rows={3} value={body} onChange={e => setBody(e.target.value)} placeholder="Add a comment" disabled={!canEdit} /><button className="btn btn-primary comment-submit" disabled={!body.trim() || add.isPending || !canEdit} onClick={() => add.mutate()}>{add.isPending ? 'Posting...' : 'Comment'}</button></section>
        <section className="drawer-section"><h3><Paperclip size={16} /> Attachments</h3><input type="file" ref={fileInputRef} className="hidden" disabled={!canEdit || upload.isPending} onChange={e => { if (e.target.files?.[0]) upload.mutate(e.target.files[0]); }} /><button className={`file-picker ${!canEdit ? 'file-picker-disabled' : ''}`} disabled={!canEdit || upload.isPending} title={!canEdit ? 'Member role required to upload attachments' : undefined} onClick={() => fileInputRef.current?.click()}><span>{upload.isPending ? 'Uploading...' : 'Choose attachment'}</span></button>{attachments.data?.length > 0 && <div className="mt-3 space-y-2">{attachments.data.map((a: any) => <div key={a.id} className="flex items-center justify-between p-2 bg-slate-50 border border-slate-200 rounded text-sm"><div className="flex items-center gap-2 overflow-hidden"><Paperclip size={14} className="text-slate-400 shrink-0" /><div className="overflow-hidden"><div className="truncate font-medium text-slate-700">{a.name}</div><div className="text-[0.65rem] text-slate-500">{(a.size / 1024).toFixed(1)} KB • by {a.uploader}</div></div></div><div className="flex gap-1 shrink-0"><button className="p-1.5 text-slate-500 hover:text-slate-800 hover:bg-slate-200 rounded" title="Download" onClick={async () => { const res = await api.get(`/attachments/${a.id}`); window.open(res.data.url, '_blank'); }}><Download size={14} /></button>{canEdit && <button className="p-1.5 text-slate-500 hover:text-red-600 hover:bg-red-50 rounded" title="Delete" onClick={() => { if(window.confirm('Delete attachment?')) delAttachment.mutate(a.id); }}><Trash2 size={14} /></button>}</div></div>)}</div>}</section>
      </div>
    </aside>
    {confirmDelete && <div className="confirm-backdrop" role="dialog" aria-modal="true"><div className="confirm-card"><h3>Delete this task?</h3><p>This removes the card, comments, and attachments from the board. This action cannot be undone.</p><div className="confirm-actions"><button className="btn" onClick={() => setConfirmDelete(false)}>Cancel</button><button className="btn btn-danger" onClick={() => deleteTask.mutate()} disabled={deleteTask.isPending}>{deleteTask.isPending ? 'Deleting...' : 'Delete task'}</button></div></div></div>}
  </>;
}

function CreateProject({ workspaceId, collapsed }: { workspaceId: string; collapsed?: boolean }) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: '', description: '', icon: 'Briefcase' as ProjectIcon });
  const mut = useMutation({ mutationFn: () => api.post(`/workspaces/${workspaceId}/projects`, form), onSuccess: () => { qc.invalidateQueries({ queryKey: ['projects', workspaceId] }); setOpen(false); setForm({name: '', description: '', icon: 'Briefcase'}); } });

  return <>
    <button className="project-add-button" aria-label="New project" title={mut.isPending ? 'Creating project' : 'New project'} disabled={!workspaceId || mut.isPending} onClick={() => setOpen(true)}><Plus size={15} /></button>
    {open && <div className="modal-backdrop" onClick={() => setOpen(false)} role="dialog" aria-modal="true"><div className="modal-card" onClick={e => e.stopPropagation()}>
      <div className="flex justify-between items-center mb-4"><h3 className="font-semibold text-lg">Create project</h3><button onClick={() => setOpen(false)} className="text-slate-400 hover:text-slate-600"><X size={16} /></button></div>
      <div className="space-y-4">
        <label className="block"><span className="block text-sm font-medium text-slate-700 mb-1">Project name</span><input className="input" autoFocus value={form.name} onChange={e => setForm({...form, name: e.target.value})} placeholder="e.g. Q3 Roadmap" /></label>
        <label className="block"><span className="block text-sm font-medium text-slate-700 mb-1">Description <span className="text-slate-400 font-normal">(optional)</span></span><textarea className="input" rows={3} value={form.description} onChange={e => setForm({...form, description: e.target.value})} placeholder="What is this project about?" /></label>
        <label className="block"><span className="block text-sm font-medium text-slate-700 mb-1">Icon</span><div className="icon-picker">{PROJECT_ICONS.map(icon => <button key={icon} type="button" className={`icon-option ${form.icon === icon ? 'icon-option-active' : ''}`} onClick={() => setForm({...form, icon})}>{projectIconComponent(icon, 16)}</button>)}</div></label>
        <div className="flex justify-end gap-2 pt-2"><button className="btn" onClick={() => setOpen(false)}>Cancel</button><button className="btn btn-primary" disabled={!form.name.trim() || mut.isPending} onClick={() => mut.mutate()}>{mut.isPending ? 'Creating...' : 'Create project'}</button></div>
      </div>
    </div></div>}
  </>;
}

function ProjectSettings({ project, workspaceId, role, onDeleted }: { project?: Project; workspaceId: string; role?: string; onDeleted: () => void }) {
  const qc = useQueryClient();
  const ui = useUIStore();
  const [open, setOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [form, setForm] = useState({ name: project?.name || '', description: project?.description || '', icon: (project as any)?.icon || '' });
  const popoverRef = useRef<HTMLDivElement>(null);
  useEffect(() => { setForm({ name: project?.name || '', description: project?.description || '', icon: (project as any)?.icon || '' }); setConfirmDelete(false); }, [project?.id, project?.name, project?.description, (project as any)?.icon]);
  useEffect(() => { if (!open) return; const handler = (e: MouseEvent) => { if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) setOpen(false); }; document.addEventListener('mousedown', handler); return () => document.removeEventListener('mousedown', handler); }, [open]);
  const save = useMutation({ mutationFn: () => api.patch(`/projects/${project?.id}`, form), onSuccess: () => { qc.invalidateQueries({ queryKey: ['projects', workspaceId] }); ui.toast({ title: 'Project settings saved', tone: 'success' }); setOpen(false); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Project update failed.', tone: 'error' }) });
  const del = useMutation({ mutationFn: () => api.delete(`/projects/${project?.id}`), onSuccess: () => { qc.invalidateQueries({ queryKey: ['projects', workspaceId] }); ui.toast({ title: 'Project deleted', tone: 'success' }); onDeleted(); setOpen(false); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Only owners can delete projects.', tone: 'error' }) });
  const canAdmin = canRole(role, 'Admin');
  const canOwner = canRole(role, 'Owner');
  return <div className="project-settings-wrap"><button type="button" className={`settings-button ${open ? 'settings-button-active' : ''}`} aria-label="Project settings" title="Project settings" onClick={() => setOpen(!open)}><Settings size={17} /></button>{open && <div ref={popoverRef} className="settings-popover" role="dialog" aria-label="Project settings"><div className="settings-head"><div><h3>Project settings</h3><p>Edit board details</p></div><button className="ghost-icon" onClick={() => setOpen(false)} aria-label="Close settings"><X size={15} /></button></div><label><span className="field-label">Project name</span><input className="input" value={form.name} disabled={!canAdmin} title={!canAdmin ? 'Admin role required' : undefined} onChange={e => setForm({ ...form, name: e.target.value })} /></label><label><span className="field-label">Description</span><textarea className="input" rows={3} value={form.description} disabled={!canAdmin} onChange={e => setForm({ ...form, description: e.target.value })} /></label><label><span className="field-label">Icon</span><div className="icon-picker">{PROJECT_ICONS.map(icon => <button key={icon} type="button" className={`icon-option ${form.icon === icon ? 'icon-option-active' : ''}`} disabled={!canAdmin} onClick={() => setForm({ ...form, icon })}>{projectIconComponent(icon)}</button>)}</div></label><PermissionHint role={role} need="Admin" action="manage project settings" /><PermissionHint role={role} need="Owner" action="delete projects" /><div className="settings-actions"><button className="btn btn-primary" disabled={!form.name.trim() || save.isPending || !project?.id || !canAdmin} onClick={() => save.mutate()}>{save.isPending ? 'Saving…' : 'Save changes'}</button>{confirmDelete ? <button className="btn btn-danger" disabled={del.isPending || !project?.id || !canOwner} title={!canOwner ? 'Owner role required' : undefined} onClick={() => del.mutate()}>{del.isPending ? 'Deleting…' : 'Confirm delete'}</button> : <button className="btn" disabled={!canOwner} title={!canOwner ? 'Owner role required' : undefined} onClick={() => setConfirmDelete(true)}>Delete project</button>}</div></div>}</div>;
}

function CommandMenu({ workspaceId, projectId, projects, onProject, onMembers }: { workspaceId: string; projectId: string; projects: Project[]; onProject: (id: string) => void; onMembers: () => void }) {
  const ui = useUIStore();
  const qc = useQueryClient();
  const [query, setQuery] = useState('');
  const board = useQuery({ queryKey: ['board', projectId], enabled: !!projectId && ui.commandOpen, queryFn: async () => (await api.get<BoardData>(`/projects/${projectId}/board`)).data });
  const tasks = useMemo(() => normalizeBoardData(board.data).columns.flatMap(c => c.tasks.map(t => ({ ...t, status: c.name, columnId: c.id }))), [board.data]);
  const createTask = useMutation({ mutationFn: async () => { const col = normalizeBoardData(board.data).columns[0]; if (!projectId || !col) throw new Error('Open a project first'); return api.post(`/projects/${projectId}/tasks`, { title: 'Untitled task', columnId: col.id, priority: 'Medium', labels: [] }); }, onSuccess: (res) => { qc.invalidateQueries({ queryKey: ['board', projectId] }); ui.toast({ title: 'Task created', body: 'Draft task opened from the command menu.', tone: 'success' }); ui.setCommandOpen(false); ui.openTask(res.data.id, true); }, onError: () => ui.toast({ title: 'Couldn’t create task', body: 'Open a project and try again.', tone: 'error' }) });
  useEffect(() => { if (!ui.commandOpen) return; const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') ui.setCommandOpen(false); }; window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey); }, [ui.commandOpen, ui]);
  if (!ui.commandOpen) return null;
  const q = query.trim().toLowerCase();
  const filteredProjects = projects.filter(p => !q || p.name.toLowerCase().includes(q)).slice(0, 5);
  const filteredTasks = tasks.filter(t => !q || `${t.title} ${t.description} ${t.status}`.toLowerCase().includes(q)).slice(0, 8);
  return <div className="modal-backdrop command-backdrop" role="dialog" aria-modal="true" aria-label="Command menu" onClick={() => ui.setCommandOpen(false)}><div className="command-menu" onClick={e => e.stopPropagation()}><div className="command-input-row"><Command size={18} /><input autoFocus value={query} onChange={e => setQuery(e.target.value)} placeholder="Create task, jump to project, search tasks, assign user…" aria-label="Command search" /></div><div className="command-list"><p className="command-section">Actions</p><button className="command-item" onClick={() => createTask.mutate()} disabled={!projectId || createTask.isPending}><Plus size={16} /><div><strong>Create task</strong><span>Add a new card to the first column</span></div><kbd>Enter</kbd></button><button className="command-item" onClick={() => { onMembers(); ui.setCommandOpen(false); }} disabled={!workspaceId}><Users size={16} /><div><strong>Go to Members</strong><span>Open workspace members, roles, and invites</span></div></button><button className="command-item" onClick={() => { ui.setSettingsOpen(true); ui.setCommandOpen(false); }} disabled={!workspaceId}><Settings size={16} /><div><strong>Open settings / admin</strong><span>Workspace, project, labels, and priorities</span></div></button><p className="command-section">Filters</p><button className="command-item" onClick={() => ui.setCommandOpen(false)} disabled={!projectId}><Clock size={16} /><div><strong>Filter: Overdue</strong><span>Use the Saved views dropdown to apply overdue tasks</span></div></button><button className="command-item" onClick={() => { ui.setAIPanelOpen(true); ui.setCommandOpen(false); }} disabled={!projectId}><Activity size={16} /><div><strong>Ask AI</strong><span>Open project analyst for blockers and next actions</span></div></button>{filteredProjects.length > 0 && <p className="command-section">Projects</p>}{filteredProjects.map(p => <button className="command-item" key={p.id} onClick={() => { onProject(p.id); ui.setCommandOpen(false); }}><Briefcase size={16} /><div><strong>{p.name}</strong><span>Jump to project</span></div></button>)}{filteredTasks.length > 0 && <p className="command-section">Tasks</p>}{filteredTasks.map(t => <button className="command-item" key={t.id} onClick={() => { ui.openTask(t.id); ui.setCommandOpen(false); }}><Search size={16} /><div><strong>{t.title || 'Untitled task'}</strong><span>{t.status} · assign user or change status from the task drawer</span></div></button>)}</div></div></div>;
}

function WorkspaceSettings({ workspace, workspaceId }: { workspace?: Workspace; workspaceId: string }) {
  const ui = useUIStore();
  const qc = useQueryClient();
  const canAdmin = canRole(workspace?.role, 'Admin');
  const canOwner = canRole(workspace?.role, 'Owner');
  const members = useQuery({ queryKey: ['members', workspaceId], enabled: !!workspaceId && ui.settingsOpen, queryFn: async () => (await api.get<WorkspaceMember[]>(`/workspaces/${workspaceId}/members`)).data });
  const projects = useQuery({ queryKey: ['projects', workspaceId], enabled: !!workspaceId && ui.settingsOpen, queryFn: async () => (await api.get<Project[]>(`/workspaces/${workspaceId}/projects`)).data });
  const invites = useQuery({ queryKey: ['invites', workspaceId], enabled: !!workspaceId && ui.settingsOpen && canAdmin, queryFn: async () => (await api.get<any[]>(`/workspaces/${workspaceId}/invites`)).data });
  
  const [name, setName] = useState(workspace?.name || '');
  const [editingProjectId, setEditingProjectId] = useState('');
  const [projectForm, setProjectForm] = useState({ name: '', description: '', icon: '' });
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [inviteForm, setInviteForm] = useState({ email: '', role: 'Member', show: false });

  const delMember = useMutation({ mutationFn: (uid: string) => api.delete(`/workspaces/${workspaceId}/members/${uid}`), onSuccess: () => qc.invalidateQueries({queryKey: ['members', workspaceId]}) });
  const updRole = useMutation({ mutationFn: ({ uid, role }: {uid: string, role: string}) => api.patch(`/workspaces/${workspaceId}/members/${uid}`, { role }), onSuccess: () => qc.invalidateQueries({queryKey: ['members', workspaceId]}) });
  const doInvite = useMutation({ mutationFn: () => api.post(`/workspaces/${workspaceId}/invites`, { email: inviteForm.email, role: inviteForm.role }), onSuccess: (res) => { qc.invalidateQueries({queryKey: ['invites', workspaceId]}); setInviteForm({email: '', role: 'Member', show: false}); ui.toast({title: 'Invite sent', body: `Invite link: ${res.data.url}`, tone: 'success'}); }, onError: (e: any) => ui.toast({title: 'Error', body: e.response?.data?.error || 'Failed to send invite', tone: 'error'}) });

  useEffect(() => setName(workspace?.name || ''), [workspace?.name]);
  const saveProject = useMutation({ mutationFn: () => api.patch(`/projects/${editingProjectId}`, projectForm), onSuccess: () => { qc.invalidateQueries({ queryKey: ['projects', workspaceId] }); ui.toast({ title: 'Project settings saved', tone: 'success' }); setEditingProjectId(''); setConfirmDelete(false); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Project update failed.', tone: 'error' }) });
  const delProject = useMutation({ mutationFn: () => api.delete(`/projects/${editingProjectId}`), onSuccess: () => { qc.invalidateQueries({ queryKey: ['projects', workspaceId] }); ui.toast({ title: 'Project deleted', tone: 'success' }); setEditingProjectId(''); setConfirmDelete(false); ui.setSettingsOpen(false); }, onError: () => ui.toast({ title: 'Couldn’t delete project', body: 'Only owners can delete projects.', tone: 'error' }) });
  useEffect(() => { if (!ui.settingsOpen) return; const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') ui.setSettingsOpen(false); }; window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey); }, [ui.settingsOpen, ui]);
  if (!ui.settingsOpen) return null;
  const editingProject = projects.data?.find(p => p.id === editingProjectId);
  return <div className="modal-backdrop" role="dialog" aria-modal="true" aria-label="Workspace settings" onClick={() => ui.setSettingsOpen(false)}><section className="admin-panel" onClick={e => e.stopPropagation()}><div className="admin-panel-head"><div><h2>Settings / admin</h2><p>{workspace?.name || 'Workspace'} · signed in as {workspace?.role || 'Member'}</p></div><button className="drawer-close" onClick={() => ui.setSettingsOpen(false)} aria-label="Close settings"><X size={18} /></button></div><div className="admin-grid"><div className="admin-card"><h3><Shield size={16} /> Workspace</h3><label><span className="field-label">Workspace name</span><input className="input" value={name} onChange={e => setName(e.target.value)} disabled={!canAdmin} title={!canAdmin ? 'Admin role required to rename workspace' : undefined} /></label><PermissionHint role={workspace?.role} need="Admin" action="rename the workspace" /><button className="btn btn-primary" disabled={!canAdmin || !name.trim()} title={!canAdmin ? 'Admin role required' : undefined} onClick={() => ui.toast({ title: 'Workspace settings saved', body: 'Demo settings are staged locally.', tone: 'success' })}>Save workspace</button></div><div className="admin-card"><h3><Users size={16} /> Members and roles</h3><div className="member-list">{members.data?.map(m => <div className="member-row" key={m.id}><span className="mini-avatar avatar-slate">{initialsForName(displayMemberName(m.name))}</span><div style={{ flex: 1 }}><strong>{displayMemberName(m.name)}</strong><small>{m.email}</small></div>{canAdmin && m.role !== 'Owner' ? <div className="member-actions"><select className="filter-select" style={{ height: '2rem' }} value={m.role} onChange={e => updRole.mutate({uid: m.id, role: e.target.value})} disabled={updRole.isPending} title="Change role">{['Viewer', 'Member', 'Admin'].map(r => <option key={r}>{r}</option>)}</select><button className="ghost-icon" title="Remove member" onClick={() => delMember.mutate(m.id)}><X size={15} /></button></div> : <span className="status-pill">{m.role}</span>}</div>)}{invites.data?.map((i: any) => <div className="member-row" key={i.id} style={{ opacity: 0.6 }}><span className="mini-avatar avatar-slate">?</span><div style={{ flex: 1 }}><strong>{i.email}</strong><small>Invited as {i.role} (Expires {new Date(i.expires_at).toLocaleDateString()})</small></div><span className="status-pill">Pending</span></div>)}</div>{!inviteForm.show ? <button className="btn" disabled={!canAdmin} title={!canAdmin ? 'Admin role required to invite members' : undefined} onClick={() => setInviteForm({ ...inviteForm, show: true })}><UserPlus size={15} /> Invite member</button> : <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem' }}><input className="input" style={{ flex: 1 }} placeholder="Email address" value={inviteForm.email} onChange={e => setInviteForm({ ...inviteForm, email: e.target.value })} /><select className="filter-select" value={inviteForm.role} onChange={e => setInviteForm({ ...inviteForm, role: e.target.value })}>{['Viewer', 'Member', 'Admin'].map(r => <option key={r}>{r}</option>)}</select><button className="btn btn-primary" disabled={!inviteForm.email || doInvite.isPending} onClick={() => doInvite.mutate()}>Send</button><button className="btn" onClick={() => setInviteForm({ ...inviteForm, show: false })}>Cancel</button></div>}</div><div className="admin-card"><h3><Briefcase size={16} /> Project settings</h3><div className="settings-list">{projects.data?.map(p => <div className="project-settings-row" key={p.id}><div><strong>{p.name}</strong><small>{p.description || 'No description'}</small></div><button className="ghost-icon" aria-label={`Edit ${p.name}`} title="Edit project" disabled={!canAdmin} onClick={() => { setEditingProjectId(p.id); setProjectForm({ name: p.name, description: p.description || '', icon: (p as any).icon || '' }); }}><Settings size={15} /></button></div>)}</div>{editingProject && <div className="project-edit-panel"><label><span className="field-label">Project name</span><input className="input" value={projectForm.name} onChange={e => setProjectForm({ ...projectForm, name: e.target.value })} /></label><label><span className="field-label">Description</span><textarea className="input" rows={3} value={projectForm.description} onChange={e => setProjectForm({ ...projectForm, description: e.target.value })} /></label><label><span className="field-label">Icon</span><div className="icon-picker">{PROJECT_ICONS.map(icon => <button key={icon} type="button" className={`icon-option ${projectForm.icon === icon ? 'icon-option-active' : ''}`} onClick={() => setProjectForm({ ...projectForm, icon })}>{projectIconComponent(icon, 16)}</button>)}</div></label><div className="settings-actions"><button className="btn" onClick={() => { setEditingProjectId(''); setConfirmDelete(false); }}>Cancel</button><div style={{display:'flex',gap:'.5rem'}}>{confirmDelete ? <button className="btn btn-danger" disabled={delProject.isPending || !canOwner} title={!canOwner ? 'Owner role required' : undefined} onClick={() => delProject.mutate()}>{delProject.isPending ? 'Deleting…' : 'Confirm delete'}</button> : <button className="btn" disabled={!canOwner} title={!canOwner ? 'Owner role required' : undefined} onClick={() => setConfirmDelete(true)}>Delete project</button>}<button className="btn btn-primary" disabled={!projectForm.name.trim() || saveProject.isPending} onClick={() => saveProject.mutate()}>{saveProject.isPending ? 'Saving…' : 'Save project'}</button></div></div></div>}<PermissionHint role={workspace?.role} need="Admin" action="edit project settings" /></div><div className="admin-card"><h3><Tag size={16} /> Labels and priorities</h3><div className="tag-options">{TAG_OPTIONS.slice(0, 10).map(tag => <span key={tag} className={`tag ${tagClass(tag)}`}>{tag}</span>)}</div><div className="priority-row">{['Low', 'Medium', 'High', 'Urgent'].map(p => <span key={p} className={`priority priority-${p.toLowerCase()}`}><Flag size={11} />{p}</span>)}</div></div></div></section></div>;
}

function BulkBar({ count, columns, members, disabled, onClear, onPriority, onAssign, onLabel, onStatus, onArchive }: { count: number; columns: Column[]; members: WorkspaceMember[]; disabled: boolean; onClear: () => void; onPriority: (v: string) => void; onAssign: (v: string) => void; onLabel: (v: string) => void; onStatus: (v: string) => void; onArchive: () => void }) {
  return <div className="bulk-bar" role="toolbar" aria-label="Bulk task actions"><strong>{count} selected</strong><select className="filter-select" disabled={disabled} defaultValue="" onChange={e => e.target.value && onStatus(e.target.value)} aria-label="Bulk change status"><option value="">Change status…</option>{columns.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}</select><select className="filter-select" disabled={disabled} defaultValue="" onChange={e => e.target.value && onAssign(e.target.value)} aria-label="Bulk assign user"><option value="">Assign user…</option>{members.map(m => <option key={m.id} value={m.id}>{displayMemberName(m.name)}</option>)}</select><select className="filter-select" disabled={disabled} defaultValue="" onChange={e => e.target.value && onPriority(e.target.value)} aria-label="Bulk priority"><option value="">Priority…</option>{['Low', 'Medium', 'High', 'Urgent'].map(p => <option key={p}>{p}</option>)}</select><select className="filter-select" disabled={disabled} defaultValue="" onChange={e => e.target.value && onLabel(e.target.value)} aria-label="Bulk add label"><option value="">Add label…</option>{TAG_OPTIONS.map(tag => <option key={tag}>{tag}</option>)}</select><button className="btn" disabled={disabled} onClick={onArchive}><Archive size={15} /> Archive/delete</button><button className="ghost-icon" onClick={onClear} aria-label="Clear selection"><X size={15} /></button></div>;
}

function CreateColumnTask({ projectId, column, canEdit = true }: { projectId: string; column: Column; canEdit?: boolean }) { const qc = useQueryClient(); const ui = useUIStore(); const mut = useMutation({ mutationFn: async () => api.post(`/projects/${projectId}/tasks`, { title: '', description: '', columnId: column.id, priority: 'Medium', labels: [] }), onSuccess: (res) => { qc.invalidateQueries({ queryKey: ['board', projectId] }); ui.toast({ title: 'Task created', tone: 'success' }); ui.openTask(res.data.id, true); }, onError: () => ui.toast({ title: 'Couldn’t save changes', body: 'Task creation failed.', tone: 'error' }) }); return <button type="button" className="add-task-row" title={canEdit ? `Add task to ${column.name}` : 'Member role required to create tasks'} disabled={!projectId || mut.isPending || !canEdit} onClick={() => mut.mutate()}><Plus size={15} /><span>{mut.isPending ? 'Adding…' : 'Add task'}</span></button>; }

function reorderBoard(board: BoardData, taskId: string, destinationColumnId: string, destinationIndex: number): BoardData {
  const columns = board.columns.map(col => ({ ...col, tasks: [...col.tasks] }));
  const sourceColumn = columns.find(col => col.tasks.some(task => task.id === taskId));
  const destinationColumn = columns.find(col => col.id === destinationColumnId);
  if (!sourceColumn || !destinationColumn) return board;
  const sourceIndex = sourceColumn.tasks.findIndex(task => task.id === taskId);
  if (sourceIndex < 0) return board;
  const [task] = sourceColumn.tasks.splice(sourceIndex, 1);
  const safeIndex = Math.max(0, Math.min(destinationIndex, destinationColumn.tasks.length));
  destinationColumn.tasks.splice(safeIndex, 0, task);
  return { columns: columns.map(col => ({ ...col, tasks: col.tasks.map((task, position) => ({ ...task, position })) })) };
}

const DEMO_REVIEW_COLUMN_ID = 'demo-in-review';

const DEMO_TASKS: Task[] = [
  { id: 'demo-025', title: 'Map onboarding checklist', description: 'Outline first-run workspace setup and success criteria.', priority: 'Medium', assigneeId: 'u6', assigneeIds: ['u6', 'u2'], dueDate: '2026-05-18', labels: ['ux', 'frontend'], position: 1 },
  { id: 'demo-026', title: 'Define billing handoff events', description: 'List webhook events and retry rules for subscription work.', priority: 'Low', assigneeId: 'u7', dueDate: '2026-05-23', labels: ['api', 'backend'], position: 2 },
  { id: 'demo-027', title: 'Normalize drag reorder payloads', description: 'Keep client positions aligned with server dense ordering.', priority: 'Urgent', assigneeId: 'u1', assigneeIds: ['u1', 'u7'], dueDate: '2026-05-12', labels: ['frontend', 'api'], position: 3 },
  { id: 'demo-028', title: 'Add member invite copy', description: 'Write helper text for pending invites and expired tokens.', priority: 'Medium', assigneeId: 'u6', dueDate: '2026-05-16', labels: ['email', 'ux'], position: 4 },
  { id: 'demo-029', title: 'Implement drawer autosave guard', description: 'Prevent accidental closes from dropping edited task metadata.', priority: 'High', assigneeId: 'u7', dueDate: '2026-05-14', labels: ['frontend', 'ux'], position: 0 },
  { id: 'demo-030', title: 'Backfill activity event actors', description: 'Repair imported rows that are missing actor metadata.', priority: 'Medium', assigneeId: 'u3', dueDate: '2026-05-17', labels: ['postgres', 'backend'], position: 1 },
  { id: 'demo-031', title: 'Review realtime disconnect copy', description: 'Confirm reconnect language is clear in flaky network states.', priority: 'Medium', assigneeId: 'u6', assigneeIds: ['u6', 'u3'], dueDate: '2026-05-13', labels: ['realtime', 'ux'], position: 0 },
  { id: 'demo-032', title: 'Validate project creation defaults', description: 'Make sure new projects include all expected board columns.', priority: 'High', assigneeId: 'u7', dueDate: '2026-05-13', labels: ['api', 'frontend'], position: 1 },
  { id: 'demo-033', title: 'Ship production env example', description: 'Document required variables for API, web, storage, and email.', priority: 'Medium', assigneeId: 'u6', dueDate: '2026-05-10', labels: ['devops', 'docs'], position: 1 },
  { id: 'demo-034', title: 'Close launch-readiness sweep', description: 'Record final launch risks and owner follow-ups.', priority: 'High', assigneeId: 'u1', dueDate: '2026-05-11', labels: ['risk', 'ux'], position: 2 },
];

function normalizeBoardData(board?: BoardData, localMoves: LocalMoveMap = {}): BoardData {
  const sourceColumns = board?.columns || [];
  const columns = sourceColumns.map((col, index) => ({ ...col, name: /^(review)$/i.test(col.name) ? 'In Review' : col.name, position: col.position ?? index, tasks: [...col.tasks] }));
  const hasReview = columns.some(col => /review/i.test(col.name));
  if (!hasReview) {
    const doneIndex = columns.findIndex(col => /done|complete/i.test(col.name));
    const reviewColumn: Column = { id: DEMO_REVIEW_COLUMN_ID, name: 'In Review', position: doneIndex >= 0 ? columns[doneIndex].position : columns.length, tasks: [] };
    columns.splice(doneIndex >= 0 ? doneIndex : columns.length, 0, reviewColumn);
  }
  const taskCount = columns.reduce((sum, col) => sum + col.tasks.length, 0);
  const hasDemoTasks = columns.some(col => col.tasks.some(t => String(t.id).startsWith('demo-')));
  if (taskCount < 8 && hasDemoTasks) {
    const todo = columns.find(col => /to do|todo|backlog/i.test(col.name)) || columns[0];
    const progress = columns.find(col => /progress/i.test(col.name)) || todo;
    const review = columns.find(col => /review/i.test(col.name)) || progress;
    const done = columns.find(col => /done|complete/i.test(col.name)) || review;
    const targetById: Record<string, string> = {
      'demo-025': todo?.id,
      'demo-026': todo?.id,
      'demo-027': todo?.id,
      'demo-028': todo?.id,
      'demo-029': progress?.id,
      'demo-030': progress?.id,
      'demo-031': review?.id,
      'demo-032': review?.id,
      'demo-033': done?.id,
      'demo-034': done?.id,
    };
    for (const task of DEMO_TASKS) {
      const move = localMoves[task.id];
      const columnId = move?.columnId || targetById[task.id];
      const target = columns.find(col => col.id === columnId);
      if (target && !columns.some(col => col.tasks.some(existing => existing.id === task.id))) target.tasks.push({ ...task, position: move?.position ?? task.position });
    }
  }
  for (const [taskId, move] of Object.entries(localMoves)) {
    const from = columns.find(col => col.tasks.some(task => task.id === taskId));
    const to = columns.find(col => col.id === move.columnId);
    if (!from || !to) continue;
    const index = from.tasks.findIndex(task => task.id === taskId);
    const [task] = from.tasks.splice(index, 1);
    to.tasks.splice(Math.max(0, Math.min(move.position, to.tasks.length)), 0, task);
  }
  return { columns: columns.map((col, index) => ({ ...col, position: index, tasks: col.tasks.map((task, position) => ({ ...task, position })) })) };
}

function labelsArray(value: unknown): string[] { if (Array.isArray(value)) return value.map(String); if (typeof value === 'string') { const cleaned = value.replace(/[{}\[\]"]/g, '').trim(); if (!cleaned) return []; return cleaned.split(',').map(s => s.trim()).filter(Boolean); } return []; }
function tagClass(label: string) { const key = label.toLowerCase(); return ({ backend: 'tag-blue', frontend: 'tag-violet', ux: 'tag-pink', postgres: 'tag-amber', email: 'tag-emerald', auth: 'tag-red', devops: 'tag-slate', realtime: 'tag-cyan', aws: 'tag-orange', api: 'tag-blue', mobile: 'tag-violet', design: 'tag-pink', security: 'tag-red', docs: 'tag-slate', risk: 'tag-red', todo: 'tag-slate' } as Record<string, string>)[key] || 'tag-neutral'; }

function shortKey(id: string) { return id.replace(/-/g, '').slice(0, 4).toUpperCase(); }
function assigneeKey(task: Task) { return task.assigneeId || '__unassigned'; }
function assigneeIdsForTask(task: Task) { return Array.from(new Set([...(task.assigneeIds || []), task.assigneeId].filter(Boolean) as string[])); }
function assigneeForTask(task: Task) { return assigneeForId(assigneeKey(task)); }
function assigneesForTask(task: Task) { return assigneeIdsForTask(task).map(assigneeForId); }
function assigneeForId(key: string) { const people: Record<string, { initials: string; name: string; className: string }> = { u1: { initials: 'NR', name: 'Nick Robbins', className: 'avatar-amber' }, u2: { initials: 'AS', name: 'Avery Stone', className: 'avatar-violet' }, u3: { initials: 'PC', name: 'Priya Chen', className: 'avatar-cyan' }, u4: { initials: 'MR', name: 'Mateo Rivera', className: 'avatar-slate' }, u5: { initials: 'JL', name: 'Jordan Lee', className: 'avatar-rose' }, u6: { initials: 'MP', name: 'Mina Patel', className: 'avatar-violet' }, u7: { initials: 'OB', name: 'Owen Brooks', className: 'avatar-cyan' }, '00000000-0000-0000-0000-000000000001': { initials: 'NR', name: 'Nick Robbins', className: 'avatar-amber' }, '00000000-0000-0000-0000-000000000002': { initials: 'AS', name: 'Avery Stone', className: 'avatar-violet' }, '00000000-0000-0000-0000-000000000003': { initials: 'PC', name: 'Priya Chen', className: 'avatar-cyan' }, '00000000-0000-0000-0000-000000000004': { initials: 'MR', name: 'Mateo Rivera', className: 'avatar-slate' }, '00000000-0000-0000-0000-000000000005': { initials: 'JL', name: 'Jordan Lee', className: 'avatar-rose' }, '00000000-0000-0000-0000-000000000006': { initials: 'MP', name: 'Mina Patel', className: 'avatar-violet' }, '00000000-0000-0000-0000-000000000007': { initials: 'OB', name: 'Owen Brooks', className: 'avatar-cyan' } }; return { key, ...(people[key] || { initials: initialsForName(key), name: 'Team member', className: 'avatar-slate' }) }; }
function columnTone(name: string) { if (/done|complete/i.test(name)) return 'column-dot-green'; if (/review/i.test(name)) return 'column-dot-blue'; if (/progress/i.test(name)) return 'column-dot-amber'; return 'column-dot-slate'; }

function getBoardSummary(board?: BoardData) {
  const columns = board?.columns || [];
  const tasks = columns.flatMap(c => c.tasks);
  const doneColumnIds = new Set(columns.filter(c => /done|complete/i.test(c.name)).map(c => c.id));
  const today = startOfToday();
  return {
    total: tasks.length,
    completed: columns.filter(c => doneColumnIds.has(c.id)).reduce((sum, col) => sum + col.tasks.length, 0),
    overdue: tasks.filter(t => t.dueDate && new Date(t.dueDate) < today).length,
    highPriority: tasks.filter(t => ['urgent', 'high'].includes(String(t.priority || '').toLowerCase())).length,
    unassigned: tasks.filter(t => !t.assigneeId).length,
    reviewLoad: columns.filter(c => /review/i.test(c.name)).reduce((sum, col) => sum + col.tasks.length, 0),
    blocked: tasks.filter(t => labelsArray(t.labels).some(l => /block|risk|bug/i.test(l))).length,
    staleTodo: columns.filter(c => /to do|todo|backlog/i.test(c.name)).flatMap(c => c.tasks).filter(t => ['urgent', 'high'].includes(String(t.priority || '').toLowerCase())).length,
  };
}

type DashboardTask = Task & { status?: string; columnId?: string };
function isDoneStatus(status?: string) { return /done|complete/i.test(status || ''); }
function isOverdueTask(task: DashboardTask) { return !!task.dueDate && new Date(task.dueDate) < startOfToday() && !isDoneStatus(task.status); }
function isPriorityTask(task: DashboardTask) { return ['urgent', 'high'].includes(String(task.priority || '').toLowerCase()); }
function getRiskQueue(tasks: DashboardTask[], columns: Column[]) {
  const staleColumnIds = new Set(columns.filter(c => /to do|todo|backlog/i.test(c.name)).map(c => c.id));
  return tasks
    .map(task => ({ task, score: riskScore(task, staleColumnIds) }))
    .filter(row => row.score > 0)
    .sort((a, b) => b.score - a.score || String(a.task.dueDate || '').localeCompare(String(b.task.dueDate || '')))
    .slice(0, 5)
    .map(row => row.task);
}
function riskScore(task: DashboardTask, staleColumnIds: Set<string>) { const priority = String(task.priority || '').toLowerCase(); if (priority === 'urgent' && isOverdueTask(task)) return 100; if (priority === 'high' && isOverdueTask(task)) return 90; if (isOverdueTask(task)) return 80; if (staleColumnIds.has(task.columnId || '') && isPriorityTask(task)) return 60; if (labelsArray(task.labels).some(l => /block|risk|bug/i.test(l))) return 50; return 0; }
function riskReason(task: DashboardTask) { if (isOverdueTask(task) && String(task.priority).toLowerCase() === 'urgent') return 'urgent and overdue'; if (isOverdueTask(task) && String(task.priority).toLowerCase() === 'high') return 'high priority overdue'; if (isOverdueTask(task)) return 'overdue'; if (isPriorityTask(task)) return 'stale priority task'; return 'needs review'; }
function getRiskBreakdown(tasks: DashboardTask[], columns: Column[]) { const staleColumnIds = new Set(columns.filter(c => /to do|todo|backlog/i.test(c.name)).map(c => c.id)); return [{ label: 'Urgent overdue', value: tasks.filter(t => String(t.priority).toLowerCase() === 'urgent' && isOverdueTask(t)).length, tone: 'red' }, { label: 'High overdue', value: tasks.filter(t => String(t.priority).toLowerCase() === 'high' && isOverdueTask(t)).length, tone: 'orange' }, { label: 'Stale priority', value: tasks.filter(t => staleColumnIds.has(t.columnId || '') && isPriorityTask(t)).length, tone: 'amber' }, { label: 'Unassigned', value: tasks.filter(t => !assigneeIdsForTask(t).length && !isDoneStatus(t.status)).length, tone: 'slate' }, { label: 'Blocked/risk labels', value: tasks.filter(t => labelsArray(t.labels).some(l => /block|risk|bug/i.test(l))).length, tone: 'blue' }]; }
function getWorkloadSnapshot(tasks: DashboardTask[]) {
  const map = new Map<string, { key: string; person: ReturnType<typeof assigneeForId>; count: number; total: number; risk: number }>();
  tasks.forEach(task => {
    const ids = assigneeIdsForTask(task);
    if (!ids.length) return;
    ids.forEach(id => {
      if (!map.has(id)) map.set(id, { key: id, person: assigneeForId(id), count: 0, total: 0, risk: 0 });
      const row = map.get(id)!;
      row.total += 1;
      if (!isDoneStatus(task.status)) row.count += 1;
      if (isPriorityTask(task) || isOverdueTask(task)) row.risk += 1;
    });
  });
  return Array.from(map.values()).sort((a, b) => b.count - a.count || b.risk - a.risk).slice(0, 7);
}
function dashboardHealthSummary(summary: ReturnType<typeof getBoardSummary>, ratio: number) { if (!summary.total) return 'No tasks are on the board yet, so health is not established.'; if (summary.overdue > 0 && summary.highPriority > 0) return `Watch the plan closely: ${summary.overdue} overdue task${summary.overdue === 1 ? '' : 's'} and ${summary.highPriority} urgent/high-priority task${summary.highPriority === 1 ? '' : 's'} need attention.`; if (ratio >= 70) return 'Project health is steady: most work is complete and remaining risk is contained.'; if (summary.unassigned > 0) return `Project is moving, but ${summary.unassigned} unassigned task${summary.unassigned === 1 ? '' : 's'} should be owned next.`; return 'Project is active with manageable risk; keep priority tasks moving through review.'; }
function getHealthLabel(summary: ReturnType<typeof getBoardSummary>) {
  if (!summary.total) return { label: 'Not started', tone: 'slate', reason: 'No tasks on the board yet.' };
  const urgentOverdue = summary.overdue > 0 && summary.highPriority > 0;
  if (urgentOverdue) return { label: 'Critical', tone: 'red', reason: `${summary.overdue} overdue and ${summary.highPriority} urgent/high-priority tasks need immediate attention.` };
  if (summary.overdue >= 10) return { label: 'Elevated', tone: 'orange', reason: `${summary.overdue} overdue tasks are piling up.` };
  if (summary.highPriority >= 10) return { label: 'Elevated', tone: 'orange', reason: `${summary.highPriority} urgent/high-priority tasks need attention.` };
  if (summary.overdue > 0) return { label: 'Watch', tone: 'amber', reason: `${summary.overdue} overdue tasks should be addressed soon.` };
  if (summary.highPriority > 0) return { label: 'Watch', tone: 'amber', reason: `${summary.highPriority} urgent/high-priority tasks are open.` };
  if (summary.completed / summary.total >= 0.7) return { label: 'Healthy', tone: 'green', reason: 'Most work is complete and remaining risk is low.' };
  return { label: 'Active', tone: 'blue', reason: 'Project is active with manageable risk.' };
}
function getActivityPulse(items: any[]) { const days = Array.from({ length: 7 }, (_, i) => { const d = new Date(); d.setDate(d.getDate() - (6 - i)); d.setHours(0,0,0,0); return { key: d.toISOString().slice(0,10), label: d.toLocaleDateString(undefined, { weekday: 'short' }), count: 0 }; }); items.forEach(item => { const d = new Date(item.created_at); if (Number.isNaN(d.getTime())) return; const key = d.toISOString().slice(0,10); const row = days.find(x => x.key === key); if (row) row.count += 1; }); return days; }
type RosterMember = WorkspaceMember & { openCount: number; totalCount: number; riskCount: number; lastActivity?: string; role: string; name: string; email: string };
const DEMO_COLLABORATORS = [
  { id: '00000000-0000-0000-0000-000000000001', name: 'Nick Robbins', email: 'nick@synergyflow.dev', role: 'Owner', joined_at: '2026-01-08T10:00:00Z' },
  { id: '00000000-0000-0000-0000-000000000002', name: 'Avery Stone', email: 'avery@synergyflow.dev', role: 'Member', joined_at: '2026-01-10T10:00:00Z' },
  { id: '00000000-0000-0000-0000-000000000003', name: 'Priya Chen', email: 'priya@synergyflow.dev', role: 'Admin', joined_at: '2026-01-12T10:00:00Z' },
  { id: '00000000-0000-0000-0000-000000000004', name: 'Mateo Rivera', email: 'mateo@synergyflow.dev', role: 'Member', joined_at: '2026-01-15T10:00:00Z' },
  { id: '00000000-0000-0000-0000-000000000005', name: 'Jordan Lee', email: 'jordan@synergyflow.dev', role: 'Viewer', joined_at: '2026-01-18T10:00:00Z' },
  { id: '00000000-0000-0000-0000-000000000006', name: 'Mina Patel', email: 'mina@synergyflow.dev', role: 'Member', joined_at: '2026-01-20T10:00:00Z' },
  { id: '00000000-0000-0000-0000-000000000007', name: 'Owen Brooks', email: 'owen@synergyflow.dev', role: 'Member', joined_at: '2026-01-22T10:00:00Z' },
] as const;
function buildWorkspaceRoster(members: WorkspaceMember[], tasks: DashboardTask[], activity: any[], fallbackRole?: string): RosterMember[] { const map = new Map<string, RosterMember>(); const seed = members.length <= 1 ? [...DEMO_COLLABORATORS, ...members] : [...members, ...DEMO_COLLABORATORS.filter(d => !members.some(m => m.id === d.id))]; seed.forEach(m => { const name = displayMemberName(m.name || assigneeForId(m.id).name || m.email || 'Workspace member'); map.set(m.id, { ...m, name, email: displayMemberEmail(name, m.email), role: normalizeRole(m.role || demoRoleFor(m.id) || fallbackRole), joined_at: m.joined_at || demoJoinedAt(m.id), openCount: 0, totalCount: 0, riskCount: 0, lastActivity: demoLastActivityFor(m.id) }); }); tasks.forEach(task => assigneeIdsForTask(task).forEach(id => { const person = assigneeForId(id); if (!map.has(id)) map.set(id, { id, name: person.name, email: demoEmailFor(person.name), role: demoRoleFor(id) || 'Member', joined_at: demoJoinedAt(id), openCount: 0, totalCount: 0, riskCount: 0, lastActivity: demoLastActivityFor(id) }); const row = map.get(id)!; row.totalCount += 1; if (!isDoneStatus(task.status)) row.openCount += 1; if (isPriorityTask(task) || isOverdueTask(task) || labelsArray(task.labels).some(l => /block|risk|bug/i.test(l))) row.riskCount += 1; })); activity.forEach(a => { const id = a.actor_id; if (!id || !map.has(id) || !a.created_at) return; const row = map.get(id)!; if (!row.lastActivity || new Date(a.created_at) > new Date(row.lastActivity)) row.lastActivity = a.created_at; }); return Array.from(map.values()).sort((a,b) => ROLE_ORDER[b.role] - ROLE_ORDER[a.role] || b.openCount - a.openCount || a.name.localeCompare(b.name)); }
function demoRoleFor(id: string) { if (id.endsWith('0001')) return 'Owner'; if (id.endsWith('0003')) return 'Admin'; if (id.endsWith('0005')) return 'Viewer'; return id ? 'Member' : ''; }
function demoJoinedAt(id: string) { const idx = Math.max(1, Number(id.slice(-1)) || 1); return `2026-01-${String(6 + idx * 2).padStart(2, '0')}T10:00:00Z`; }
function demoLastActivityFor(id: string) { const hours: Record<string, number> = { '00000000-0000-0000-0000-000000000001': 12, '00000000-0000-0000-0000-000000000002': 31, '00000000-0000-0000-0000-000000000003': 18, '00000000-0000-0000-0000-000000000004': 42, '00000000-0000-0000-0000-000000000005': 96, '00000000-0000-0000-0000-000000000006': 27, '00000000-0000-0000-0000-000000000007': 34 }; const h = hours[id]; return h ? new Date(Date.now() - h * 3600000).toISOString() : undefined; }
function demoEmailFor(name: string) { return `${name.toLowerCase().replace(/[^a-z]+/g, '.').replace(/^\.|\.$/g, '')}@synergyflow.dev`; }
function countRoles(roster: RosterMember[]) { return roster.reduce((acc, m) => { acc[normalizeRole(m.role) as 'Owner'|'Admin'|'Member'|'Viewer'] += 1; return acc; }, { Owner: 0, Admin: 0, Member: 0, Viewer: 0 } as Record<'Owner'|'Admin'|'Member'|'Viewer', number>); }

function AvatarStack({ members }: { members: ReturnType<typeof assigneeForTask>[] }) { return <div className="avatar-stack" aria-label="Project members">{members.slice(0, 4).map(m => <span key={m.key} title={m.name} className={`mini-avatar ${m.className}`}>{m.initials}</span>)}</div>; }
function UserChip({ user }: { user: any }) { const rawName = user?.name || user?.Name || user?.email || user?.Email || 'Current user'; const name = rawName === 'Demo Owner' ? 'Nick Robbins' : rawName; return <div className="user-chip" title={`Signed in as ${name}`}><span className="mini-avatar avatar-slate">{initialsForName(name)}</span><span>{name}</span></div>; }
function InsightStat({ label, value, tone }: { label: string; value: React.ReactNode; tone: 'green' | 'orange' | 'slate' }) { return <div className={`insight-stat insight-${tone}`}><span>{label}</span><strong>{value}</strong></div>; }

function canRole(role: string | undefined, need: string) { return (ROLE_ORDER[role || 'Viewer'] || 1) >= ROLE_ORDER[need]; }
function normalizeRole(role?: string) { const value = String(role || 'Viewer'); return ['Owner', 'Admin', 'Member', 'Viewer'].includes(value) ? value : 'Viewer'; }
function formatMemberDate(value?: string) { if (!value) return 'Not available'; const date = new Date(value); return Number.isNaN(date.getTime()) ? 'Not available' : date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }); }
function PermissionHint({ role, need, action }: { role?: string; need: string; action: string }) { const ok = canRole(role, need); return <p className={`permission-hint ${ok ? 'permission-ok' : 'permission-locked'}`}><Shield size={13} />{ok ? `${role || 'Member'} can ${action}.` : `${need} role required to ${action}. Current role: ${role || 'Viewer'}.`}</p>; }
function ToastStack() { const ui = useUIStore(); return <div className="toast-stack" aria-live="polite">{ui.toasts.map(t => <div key={t.id} className={`toast toast-${t.tone || 'info'} ${t.exiting ? 'toast-exiting' : ''}`}><Bell size={16} /><div><strong>{t.title}</strong>{t.body && <p>{t.body}</p>}</div><button onClick={() => ui.dismissToast(t.id)} aria-label="Dismiss notification"><X size={14} /></button></div>)}</div>; }
function DonutChart({ rows, totalLabel }: { rows: { label: string; value: number; tone?: string }[]; totalLabel: string }) {
  const total = rows.reduce((s, r) => s + r.value, 0);
  let offset = 25;
  const r = 38;
  const c = 2 * Math.PI * r;
  return <div className="donut-wrap">
    <svg viewBox="0 0 100 100" className="donut-chart" role="img" aria-label={`${total} ${totalLabel}`}>
      <circle cx="50" cy="50" r={r} className="donut-bg" />
      {total > 0 && rows.filter(x => x.value > 0).map((row, i) => {
        const len = (row.value / total) * c;
        const seg = <circle key={row.label} cx="50" cy="50" r={r} className={`donut-segment ${row.tone || `chart-tone-${i % 5}`}`} strokeDasharray={`${len} ${c - len}`} strokeDashoffset={offset} />;
        offset -= len;
        return seg;
      })}
      <text x="50" y="47" textAnchor="middle" className="donut-total">{total}</text>
      <text x="50" y="59" textAnchor="middle" className="donut-label">{totalLabel}</text>
    </svg>
    <div className="chart-legend donut-legend">{rows.map((row, i) => <div key={row.label} className={row.value === 0 ? 'legend-muted' : ''}><span className={`legend-dot ${row.tone || `chart-tone-${i % 5}`}`} /><strong>{row.value}</strong><em>{row.label}</em></div>)}</div>
  </div>;
}
function StatusDistribution({ columns }: { columns: Column[] }) {
  const total = columns.reduce((s, c) => s + c.tasks.length, 0);
  let offset = 25;
  const r = 40;
  const c = 2 * Math.PI * r;
  return (
    <div className="status-distribution">
      <div className="status-donut-area">
        <svg viewBox="0 0 100 100" className="donut-chart donut-chart-large" role="img" aria-label={`${total} tasks`}>
          <circle cx="50" cy="50" r={r} className="donut-bg" />
          {total > 0 && columns.filter(col => col.tasks.length > 0).map((col, i) => {
            const len = (col.tasks.length / total) * c;
            const seg = <circle key={col.id} cx="50" cy="50" r={r} className={`donut-segment chart-tone-${i % 5}`} strokeDasharray={`${len} ${c - len}`} strokeDashoffset={offset} />;
            offset -= len;
            return seg;
          })}
          <text x="50" y="47" textAnchor="middle" className="donut-total">{total}</text>
          <text x="50" y="59" textAnchor="middle" className="donut-label">tasks</text>
        </svg>
        <div className="chart-legend donut-legend">
          {columns.map((col, i) => (
            <div key={col.id}>
              <span className={`legend-dot chart-tone-${i % 5}`} />
              <strong>{col.tasks.length}</strong>
              <em>{col.name}</em>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
function RiskCompositionChart({ data }: { data: { label: string; value: number; tone: string }[] }) {
  const total = data.reduce((s, r) => s + r.value, 0);
  let offset = 25;
  const r = 40;
  const c = 2 * Math.PI * r;
  const topDrivers = data.filter(d => d.value > 0).sort((a, b) => b.value - a.value).slice(0, 4);
  return (
    <div className="risk-composition">
      <div className="status-donut-area">
        <svg viewBox="0 0 100 100" className="donut-chart donut-chart-large" role="img" aria-label={`${total} risk signals`}>
          <circle cx="50" cy="50" r={r} className="donut-bg" />
          {total > 0 && data.filter(d => d.value > 0).map((d, i) => {
            const len = (d.value / total) * c;
            const seg = <circle key={d.label} cx="50" cy="50" r={r} className={`donut-segment risk-tone-${d.tone}`} strokeDasharray={`${len} ${c - len}`} strokeDashoffset={offset} />;
            offset -= len;
            return seg;
          })}
          <text x="50" y="47" textAnchor="middle" className="donut-total">{total}</text>
          <text x="50" y="59" textAnchor="middle" className="donut-label">signals</text>
        </svg>
        <div className="chart-legend donut-legend">
          {data.map((d, i) => (
            <div key={d.label} className={d.value === 0 ? 'legend-muted' : ''}>
              <span className={`legend-dot risk-tone-${d.tone}`} />
              <strong>{d.value}</strong>
              <em>{d.label}</em>
            </div>
          ))}
        </div>
      </div>
      {topDrivers.length > 0 && (
        <div className="risk-drivers">
          <p className="risk-drivers-heading">Top drivers</p>
          {topDrivers.map(d => (
            <div key={d.label} className="risk-driver-row">
              <span className={`risk-driver-dot risk-tone-${d.tone}`} />
              <span className="risk-driver-label">{d.label}</span>
              <span className="risk-driver-value">{d.value}</span>
            </div>
          ))}
        </div>
      )}
      <p className="chart-note">Signals can overlap when one task matches multiple risk rules.</p>
    </div>
  );
}
function WorkloadChart({ rows }: { rows: ReturnType<typeof getWorkloadSnapshot> }) { const max = Math.max(1, ...rows.map(r => r.count)); return <div className="workload-chart">{rows.length ? rows.map(r => <div className="workload-chart-row" key={r.key}><span className={`mini-avatar ${r.person.className}`}>{r.person.initials}</span><div><div className="workload-chart-label"><strong>{r.person.name}</strong><em>{r.count} open / {r.total} total · {r.risk} risk</em></div><i><b style={{ width: `${(r.count / max) * 100}%` }} /></i></div></div>) : <p className="empty-copy">No assigned work yet.</p>}</div>; }
function ActivityPulseChart({ rows }: { rows: ReturnType<typeof getActivityPulse> }) {
  const max = Math.max(1, ...rows.map(r => r.count));
  const hasActivity = rows.some(r => r.count > 0);
  const pts = rows.map((r, i) => `${8 + i * 14},${88 - (r.count / max) * 70}`).join(' ');
  return <div className="line-chart-wrap"><p className="chart-hint">Activity events per day</p><div className={`line-chart ${!hasActivity ? 'line-chart-empty' : ''}`}><svg viewBox="0 0 100 100" preserveAspectRatio="none"><polyline points={pts} /></svg><div className="line-points">{rows.map((r, i) => <span key={r.key} style={{ left: `${8 + i * 14}%`, top: `${88 - (r.count / max) * 70}%` }}><b>{r.count}</b></span>)}</div><div className="line-labels">{rows.map(r => <em key={r.key}>{r.label}</em>)}</div>{!hasActivity && <p className="line-empty-copy">No activity events in the last week.</p>}</div></div>;
}
function RoleDistributionChart({ counts }: { counts: Record<'Owner'|'Admin'|'Member'|'Viewer', number> }) { const roles = ['Owner','Admin','Member','Viewer'] as const; return <DonutChart totalLabel="members" rows={roles.map(r => ({ label: r, value: counts[r], tone: `role-segment role-${r.toLowerCase()}` }))} />; }
function TaskActivity({ task, items }: { task?: TaskDetail; items: any[] }) { const related = items.filter(i => !task?.id || JSON.stringify(i.metadata || '').includes(task.id)).slice(0, 5); const fallback = [{ id: 'a1', actor: 'Priya', text: 'changed priority from Medium → High', time: '12m ago' }, { id: 'a2', actor: 'Mateo', text: 'added attachment', time: '38m ago' }, { id: 'a3', actor: 'Avery', text: `moved task to ${task?.status || 'In Review'}`, time: '1h ago' }]; const rows = related.length ? related.map((i, idx) => ({ id: i.id || idx, actor: actorName(i.actor_id), text: activityText(i.event_type), time: relativeTime(i.created_at) })) : fallback; return <section className="drawer-section"><h3><Activity size={16} /> Activity history</h3><div className="activity-timeline">{rows.map(row => <div className="activity-line" key={row.id}><span className="activity-dot" /><p><strong>{row.actor}</strong> {row.text}</p><small>{row.time}</small></div>)}</div></section>; }
function actorName(id?: string) { return id ? assigneeForId(id).name : 'Team member'; }
function activityText(type?: string) { if (type === 'task.moved') return 'moved this task'; if (type === 'attachment.created') return 'added attachment'; if (type === 'comment.created') return 'added a comment'; if (type === 'task.updated') return 'updated task details'; return 'updated this task'; }
function relativeTime(value?: string) { if (!value) return 'just now'; const diff = Math.max(1, Math.round((Date.now() - new Date(value).getTime()) / 60000)); return diff < 60 ? `${diff}m ago` : `${Math.round(diff / 60)}h ago`; }
function trapFocus(e: KeyboardEvent, root: HTMLElement) { const focusable = Array.from(root.querySelectorAll<HTMLElement>('button,[href],input,select,textarea,[tabindex]:not([tabindex="-1"])')).filter(el => !el.hasAttribute('disabled')); if (!focusable.length) return; const first = focusable[0]; const last = focusable[focusable.length - 1]; if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); } else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); } }

function BoardRail({ activeTab, setActiveTab, summary, tasks, workload, onClose }: { activeTab: InsightTab; setActiveTab: (tab: InsightTab) => void; summary: ReturnType<typeof getBoardSummary>; tasks: (Task & { status?: string; columnId?: string })[]; workload: ReturnType<typeof getWorkloadSnapshot>; onClose: () => void }) {
  const urgentOverdue = tasks.filter(t => String(t.priority).toLowerCase() === 'urgent' && !!t.dueDate && new Date(t.dueDate) < startOfToday()).length;
  const overloaded = workload.filter(w => w.count >= 5).length;
  const activityTasks = tasks.slice(0, 6);
  return <aside className="insights-panel">
    <div className="insights-panel-head">
      <h2>Board triage</h2>
      <button className="ghost-icon" onClick={onClose} title="Close panel"><X size={15} /></button>
    </div>
    <div className="insight-tabs">
      <button className={activeTab === 'risks' ? 'insight-tab-active' : ''} onClick={() => setActiveTab('risks')}>Risks</button>
      <button className={activeTab === 'activity' ? 'insight-tab-active' : ''} onClick={() => setActiveTab('activity')}>Activity</button>
    </div>
    {activeTab === 'risks' ? <>
      <InsightStat label="Urgent overdue" value={urgentOverdue} tone={urgentOverdue > 0 ? 'orange' : 'green'} />
      <InsightStat label="High-priority stale" value={summary.staleTodo} tone={summary.staleTodo > 0 ? 'orange' : 'green'} />
      <InsightStat label="Overloaded assignees" value={overloaded} tone={overloaded > 0 ? 'orange' : 'green'} />
      <InsightStat label="Unassigned open" value={summary.unassigned} tone={summary.unassigned > 0 ? 'slate' : 'green'} />
    </> : <div className="activity-list activity-list-flush">
      {activityTasks.length ? activityTasks.map((t, i) => <div className="activity-item" key={t.id}>
        <span className="activity-dot" />
        <p><strong>{t.assigneeId ? assigneeForTask(t).name : 'Unassigned'}</strong> <span>{t.title || 'Untitled task'}</span></p>
        <small>{t.status} · {i + 1}h ago</small>
      </div>) : <p className="empty-copy">No recent activity.</p>}
    </div>}
  </aside>;
}
function initialsForName(name: string) { return name.split(/\s+/).filter(Boolean).slice(0, 2).map(part => part[0]?.toUpperCase()).join('') || 'U'; }
function displayMemberName(name: string) { return name === 'Demo Owner' || name === 'Nick Robins' ? 'Nick Robbins' : name; }
function displayMemberEmail(name: string, email?: string) { if (displayMemberName(name) === 'Nick Robbins' && (!email || email === 'demo@synergyflow.dev')) return 'nick@synergyflow.dev'; return email || demoEmailFor(name); }

createRoot(document.getElementById('root')!).render(<QueryClientProvider client={qc}><App /></QueryClientProvider>);
