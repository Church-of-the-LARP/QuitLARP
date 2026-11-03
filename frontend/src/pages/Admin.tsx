import { useCallback, useEffect, useState } from 'react';
import AdminAssessments from '../components/admin/AdminAssessments.tsx';
import AdminLiveSessions from '../components/admin/AdminLiveSessions.tsx';
import AdminOverview from '../components/admin/AdminOverview.tsx';
import AdminSubmissions from '../components/admin/AdminSubmissions.tsx';
import AdminUsers from '../components/admin/AdminUsers.tsx';
import client, { apiUrl, getErrorText } from '../scripts/api.ts';
import usePageTitle from '../scripts/usePageTitle.ts';
import type {
  AdminAssessment,
  AdminLiveSession,
  AdminSubmission,
  AdminTab,
} from '../types/admin.ts';
import type { User } from '../types/types.ts';

const tabs: { id: AdminTab; label: string }[] = [
  { id: 'overview', label: 'Overview' }, { id: 'users', label: 'Users' },
  { id: 'assessments', label: 'Assessments' }, { id: 'sessions', label: 'Live sessions' },
  { id: 'submissions', label: 'Submissions' },
];

async function adminFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiUrl}${path}`, {
    ...init, credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  });
  if (!response.ok) {
    let detail = `Request failed (HTTP ${response.status})`;
    try { detail = (await response.json()).detail ?? detail; } catch { /* retain HTTP fallback */ }
    throw new Error(detail);
  }
  return response.status === 204 ? (undefined as T) : response.json();
}

export default function Admin() {
  usePageTitle('Admin');
  const [tab, setTab] = useState<AdminTab>('overview');
  const [users, setUsers] = useState<User[]>([]);
  const [assessments, setAssessments] = useState<AdminAssessment[]>([]);
  const [assessmentCount, setAssessmentCount] = useState(0);
  const [sessions, setSessions] = useState<AdminLiveSession[]>([]);
  const [submissions, setSubmissions] = useState<AdminSubmission[]>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState<number | string | null>(null);

  const loadUsers = useCallback(async () => {
    const res = await client.GET('/api/v1/users');
    if (res.error) throw new Error(getErrorText(res.error, 'Could not load users'));
    setUsers(res.data?.users ?? []);
  }, []);
  const loadAssessments = useCallback(async () => {
    const res = await client.GET('/api/v1/assessments', { params: { query: { limit: 100, offset: 0 } } });
    if (res.error) throw new Error(getErrorText(res.error, 'Could not load assessments'));
    setAssessments((res.data?.assessments ?? []) as AdminAssessment[]);
    setAssessmentCount(res.data?.total ?? 0);
  }, []);
  const loadSessions = useCallback(async () => {
    const data = await adminFetch<{ sessions: AdminLiveSession[] }>('/api/v1/admin/sessions');
    setSessions(data.sessions ?? []);
  }, []);
  const loadSubmissions = useCallback(async () => {
    const data = await adminFetch<{ submissions: AdminSubmission[] }>('/api/v1/admin/submissions');
    setSubmissions(data.submissions ?? []);
  }, []);

  const refresh = useCallback(async () => {
    setError('');
    try { await Promise.all([loadUsers(), loadAssessments(), loadSessions(), loadSubmissions()]); }
    catch (e) { setError(e instanceof Error ? e.message : 'Could not load admin data'); }
  }, [loadAssessments, loadSessions, loadSubmissions, loadUsers]);
  useEffect(() => { void refresh(); }, [refresh]);

  const removeAssessment = async (item: AdminAssessment) => {
    if (!window.confirm(`Delete assessment “${item.title}”? This removes the assessment and its nested content.`)) return;
    setBusy(item.id); setError('');
    const res = await client.DELETE('/api/v1/assessments/{id}', { params: { path: { id: item.id } } });
    setBusy(null);
    if (res.error) { setError(getErrorText(res.error, 'Could not delete assessment')); return; }
    setAssessments((current) => current.filter((a) => a.id !== item.id));
    setAssessmentCount((current) => Math.max(0, current - 1));
  };
  const saveTitle = async (item: AdminAssessment, draft: string) => {
    const title = draft.trim();
    if (!title) {
      setError('Assessment title cannot be blank');
      return false;
    }
    setBusy(item.id); setError('');
    const res = await client.PATCH('/api/v1/assessments/{id}', {
      params: { path: { id: item.id } }, body: { title },
    });
    setBusy(null);
    if (res.error) {
      setError(getErrorText(res.error, 'Could not update assessment'));
      return false;
    }
    setAssessments((current) => current.map((a) => a.id === item.id ? { ...a, title } : a));
    return true;
  };
  const endSession = async (session: AdminLiveSession) => {
    if (!window.confirm(`End the live session for ${session.assessmentTitle}?`)) return;
    setBusy(session.id); setError('');
    try {
      await adminFetch(`/api/v1/admin/sessions/${encodeURIComponent(session.id)}`, { method: 'DELETE' });
      setSessions((current) => current.filter((s) => s.id !== session.id));
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not end session'); }
    setBusy(null);
  };

  return (
    <main className="mx-auto w-[90vw] max-w-6xl py-8">
      <div className="mb-7 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-semibold uppercase tracking-widest text-accent-light">
            Control panel
          </p>
          <h1 className="text-3xl font-bold">Administration</h1>
          <p className="mt-2 text-sm text-muted">
            Manage accounts, assessments, live gVisor sessions, and submissions.
          </p>
        </div>
        <button
          onClick={() => void refresh()}
          className="rounded-lg border border-line-hover bg-surface px-4 py-2 text-sm hover:bg-raised"
        >
          Refresh data
        </button>
      </div>
      <nav
        className="mb-6 flex flex-wrap gap-2 border-b border-line pb-3"
        aria-label="Admin sections"
      >
        {tabs.map((item) => (
          <button
            key={item.id}
            onClick={() => setTab(item.id)}
            aria-current={tab === item.id ? 'page' : undefined}
            className={`rounded-lg px-3 py-2 text-sm ${tab === item.id ? 'bg-accent/20 text-accent-light' : 'text-muted hover:bg-surface hover:text-ink'}`}
          >
            {item.label}
          </button>
        ))}
      </nav>
      {error && (
        <div
          role="alert"
          className="mb-5 rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-danger-light"
        >
          {error}
        </div>
      )}

      {tab === 'overview' && (
        <AdminOverview
          users={users}
          assessmentCount={assessmentCount}
          sessions={sessions}
          submissions={submissions}
          onSelect={setTab}
        />
      )}
      {tab === 'users' && <AdminUsers />}
      {tab === 'assessments' && (
        <AdminAssessments
          assessments={assessments}
          busyId={busy}
          onDelete={(item) => void removeAssessment(item)}
          onSaveTitle={saveTitle}
        />
      )}
      {tab === 'sessions' && (
        <AdminLiveSessions
          sessions={sessions}
          busyId={busy}
          onEnd={(session) => void endSession(session)}
        />
      )}
      {tab === 'submissions' && (
        <AdminSubmissions submissions={submissions} />
      )}
    </main>
  );
}
