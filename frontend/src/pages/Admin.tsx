import { useCallback, useEffect, useEffectEvent, useState } from 'react';
import { Link } from 'react-router-dom';
import type { components } from '../api/schema.ts';
import Alert from '../components/Alert.tsx';
import useAuth from '../scripts/useAuth.tsx';
import client, { apiUrl, getErrorText } from '../scripts/api.ts';
import usePageTitle from '../scripts/usePageTitle.ts';
import type { AlertMessage, Role, User } from '../types/types.ts';

type Assessment = components['schemas']['AssessmentSummary'];
type AdminTab = 'users' | 'sessions' | 'assessments';
type AdminSession = {
  id: string;
  containerId: string;
  userId: number;
  assessmentId: number;
  assessmentTitle: string;
  chapterTitle: string;
  runtime: string;
  state: string;
  image: string;
  createdAt?: string;
  deadline?: string;
};

const roleStyles: Record<Role, string> = {
  user: 'bg-raised text-muted',
  admin: 'bg-accent-light/20 text-accent-light',
  superadmin: 'bg-warning/20 text-warning',
};

const instanceStateStyles: Record<string, string> = {
  running: 'bg-success/15 text-success',
  created: 'bg-warning/15 text-warning',
  exited: 'bg-raised text-muted',
  dead: 'bg-danger/15 text-danger-light',
};

async function getAdminSessions(): Promise<AdminSession[]> {
  const response = await fetch(`${apiUrl}/api/v1/admin/sessions`, {
    credentials: 'include',
  });
  const body = (await response.json()) as {
    sessions?: AdminSession[];
    detail?: string;
  };
  if (!response.ok) throw new Error(body.detail ?? 'Could not load live sessions');
  return body.sessions ?? [];
}

async function stopAdminSession(id: string) {
  const response = await fetch(
    `${apiUrl}/api/v1/admin/sessions/${encodeURIComponent(id)}`,
    { method: 'DELETE', credentials: 'include' },
  );
  if (!response.ok) {
    const body = (await response.json()) as { detail?: string };
    throw new Error(body.detail ?? 'Could not stop the live session');
  }
}

function App() {
  usePageTitle('Admin');
  const { user } = useAuth();
  const isSuperadmin = user?.role === 'superadmin';
  const [activeTab, setActiveTab] = useState<AdminTab>('users');
  const [users, setUsers] = useState<User[]>([]);
  const [assessments, setAssessments] = useState<Assessment[]>([]);
  const [assessmentTotal, setAssessmentTotal] = useState(0);
  const [assessmentOffset, setAssessmentOffset] = useState(0);
  const [sessions, setSessions] = useState<AdminSession[]>([]);
  const [filter, setFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [alertMsg, setAlertMsg] = useState<AlertMessage | null>(null);

  const loadData = useCallback(async (offset = 0) => {
    const [usersResult, assessmentsResult, sessionsResult] =
      await Promise.allSettled([
        client.GET('/api/v1/users'),
        client.GET('/api/v1/assessments', {
          params: { query: { limit: 100, offset } },
        }),
        getAdminSessions(),
      ]);

    const errors: string[] = [];
    if (usersResult.status === 'fulfilled') {
      if (usersResult.value.error) {
        errors.push(getErrorText(usersResult.value.error, 'Could not load users'));
      } else {
        setUsers(usersResult.value.data?.users ?? []);
      }
    } else {
      errors.push('Could not load users');
    }

    if (assessmentsResult.status === 'fulfilled') {
      if (assessmentsResult.value.error) {
        errors.push(
          getErrorText(assessmentsResult.value.error, 'Could not load assessments'),
        );
      } else {
        setAssessments(assessmentsResult.value.data?.assessments ?? []);
        setAssessmentTotal(assessmentsResult.value.data?.total ?? 0);
      }
    } else {
      errors.push('Could not load assessments');
    }

    if (sessionsResult.status === 'fulfilled') {
      setSessions(sessionsResult.value);
    } else {
      errors.push(
        sessionsResult.reason instanceof Error
          ? sessionsResult.reason.message
          : 'Could not load live sessions',
      );
    }

    setAlertMsg(
      errors.length > 0 ? { kind: 'error', text: errors.join('. ') } : null,
    );
    setLoading(false);
  }, []);

  const loadInitialData = useEffectEvent(() => {
    void loadData();
  });

  useEffect(() => {
    loadInitialData();
  }, []);

  if (!user) return null;

  const normalizedFilter = filter.trim().toLowerCase();
  const visibleUsers = users.filter((item) =>
    `${item.username} ${item.email} ${item.role}`
      .toLowerCase()
      .includes(normalizedFilter),
  );
  const visibleAssessments = assessments.filter((item) =>
    `${item.title} ${item.author?.username ?? ''} ${item.kind} ${item.tags
      .map((tag) => tag.name)
      .join(' ')}`
      .toLowerCase()
      .includes(normalizedFilter),
  );
  const visibleSessions = sessions.filter((item) =>
    `${item.assessmentTitle} ${item.chapterTitle} ${item.runtime} ${item.state} ${item.userId}`
      .toLowerCase()
      .includes(normalizedFilter),
  );

  const handleRoleChange = async (target: User, role: Role) => {
    setBusyId(`user-${target.id}`);
    const response = await client.PATCH('/api/v1/users/{id}/role', {
      params: { path: { id: target.id } },
      body: { role },
    });
    setBusyId(null);
    if (response.error) {
      setAlertMsg({
        kind: 'error',
        text: getErrorText(response.error, 'Could not change the role'),
      });
      return;
    }
    setLoading(true);
    await loadData(assessmentOffset);
  };

  const handleStopSession = async (session: AdminSession) => {
    if (!window.confirm(`Stop the gVisor instance for ${session.assessmentTitle || `session ${session.id}`}?`)) return;
    setBusyId(`session-${session.id}`);
    try {
      await stopAdminSession(session.id);
      setLoading(true);
      await loadData(assessmentOffset);
    } catch (error) {
      setAlertMsg({
        kind: 'error',
        text: error instanceof Error ? error.message : 'Could not stop the live session',
      });
    } finally {
      setBusyId(null);
    }
  };

  const handleDeleteAssessment = async (assessment: Assessment) => {
    if (!window.confirm(`Delete the entire assessment “${assessment.title}”? This cannot be undone.`)) return;
    setBusyId(`assessment-${assessment.id}`);
    const response = await client.DELETE('/api/v1/assessments/{id}', {
      params: { path: { id: assessment.id } },
    });
    setBusyId(null);
    if (response.error) {
      setAlertMsg({
        kind: 'error',
        text: getErrorText(response.error, 'Could not delete the assessment'),
      });
      return;
    }
    setLoading(true);
    await loadData(assessmentOffset);
  };

  const tabs: { id: AdminTab; label: string; count: number }[] = [
    { id: 'users', label: 'Users', count: users.length },
    { id: 'sessions', label: 'gVisor instances', count: sessions.length },
    { id: 'assessments', label: 'Assessments', count: assessmentTotal },
  ];

  return (
    <main className="mx-auto max-w-6xl px-5 py-8 sm:px-8">
      <div className="flex flex-wrap items-end justify-between gap-4 border-b border-line pb-5">
        <div>
          <p className="text-xs font-semibold uppercase text-accent-light">Operations</p>
          <h1 className="mt-1 text-3xl font-semibold text-ink">Admin</h1>
        </div>
        <button
          type="button"
          onClick={() => {
            setLoading(true);
            void loadData(assessmentOffset);
          }}
          disabled={loading}
          className="rounded-md border border-line-hover px-3 py-2 text-sm font-medium text-muted hover:bg-raised hover:text-ink disabled:opacity-50"
        >
          {loading ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>

      <div className="grid grid-cols-3 divide-x divide-line border-b border-line py-4">
        {tabs.map((tab) => (
          <div key={tab.id} className="px-3 first:pl-0">
            <p className="text-xs text-dim">{tab.label}</p>
            <p className="mt-1 text-xl font-semibold text-ink">{tab.count}</p>
          </div>
        ))}
      </div>

      <div className="mt-5 flex flex-wrap items-center justify-between gap-3 border-b border-line">
        <nav className="flex gap-5" aria-label="Admin sections">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => {
                setActiveTab(tab.id);
                setFilter('');
              }}
              aria-current={activeTab === tab.id ? 'page' : undefined}
              className={`border-b-2 px-1 pb-3 text-sm font-medium ${
                activeTab === tab.id
                  ? 'border-accent-light text-ink'
                  : 'border-transparent text-muted hover:text-ink'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
        <label className="mb-2">
          <span className="sr-only">Filter {activeTab}</span>
          <input
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            placeholder={`Filter ${activeTab}…`}
            className="w-56 rounded-md border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none placeholder:text-dim focus:border-accent"
          />
        </label>
      </div>

      {alertMsg && (
        <div className="mt-4">
          <Alert kind={alertMsg.kind} text={alertMsg.text} />
        </div>
      )}

      {activeTab === 'users' && (
        <section aria-label="Users">
          <div className="flex items-center justify-between py-4">
            <h2 className="text-base font-semibold text-ink">User accounts</h2>
            <span className="text-xs text-dim">{visibleUsers.length} shown</span>
          </div>
          <ul className="divide-y divide-line border-y border-line">
            {visibleUsers.map((item) => (
              <li key={item.id} className="flex flex-wrap items-center justify-between gap-4 py-4">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="truncate text-sm font-semibold text-ink">{item.username}</p>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[item.role]}`}>
                      {item.role}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-muted">{item.email}</p>
                  <p className="mt-1 text-xs text-dim">
                    {item.emailVerified ? 'Verified' : 'Unverified'}
                    {item.googleLinked ? ' · Google linked' : ''}
                  </p>
                </div>
                {isSuperadmin && item.role !== 'superadmin' && (
                  <button
                    type="button"
                    onClick={() => void handleRoleChange(item, item.role === 'user' ? 'admin' : 'user')}
                    disabled={busyId === `user-${item.id}`}
                    className="rounded-md border border-line-hover px-3 py-2 text-xs font-medium text-muted hover:bg-raised hover:text-ink disabled:opacity-50"
                  >
                    {item.role === 'user' ? 'Make admin' : 'Revoke admin'}
                  </button>
                )}
              </li>
            ))}
            {!loading && visibleUsers.length === 0 && (
              <li className="py-10 text-center text-sm text-dim">No matching users</li>
            )}
          </ul>
        </section>
      )}

      {activeTab === 'sessions' && (
        <section aria-label="Live sessions">
          <div className="flex items-center justify-between py-4">
            <h2 className="text-base font-semibold text-ink">gVisor instances</h2>
            <span className="text-xs text-dim">{visibleSessions.length} shown</span>
          </div>
          <ul className="divide-y divide-line border-y border-line">
            {visibleSessions.map((item) => (
              <li key={item.id} className="flex flex-wrap items-center justify-between gap-4 py-4">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-sm font-semibold text-ink">
                      {item.assessmentTitle || `Assessment #${item.assessmentId || 'unknown'}`}
                    </p>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${instanceStateStyles[item.state] ?? 'bg-raised text-muted'}`}>
                      {item.state}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-muted">
                    {item.chapterTitle || 'Session metadata unavailable'} · {item.runtime}
                  </p>
                  <p className="mt-1 break-all text-xs text-dim">
                    User #{item.userId || 'unknown'} · {item.image} · {item.containerId.slice(0, 12)}
                  </p>
                  {item.deadline && (
                    <p className="mt-1 text-xs text-dim">
                      Ends {new Date(item.deadline).toLocaleString()}
                    </p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => void handleStopSession(item)}
                  disabled={busyId === `session-${item.id}`}
                  className="rounded-md border border-danger/40 px-3 py-2 text-xs font-medium text-danger-light hover:bg-danger/10 disabled:opacity-50"
                >
                  {item.state === 'running' ? 'Stop instance' : 'Remove instance'}
                </button>
              </li>
            ))}
            {!loading && visibleSessions.length === 0 && (
              <li className="py-10 text-center text-sm text-dim">No live instances</li>
            )}
          </ul>
        </section>
      )}

      {activeTab === 'assessments' && (
        <section aria-label="Assessments">
          <div className="flex items-center justify-between py-4">
            <h2 className="text-base font-semibold text-ink">Assessments</h2>
            <span className="text-xs text-dim">
              {assessmentTotal === 0
                ? '0 assessments'
                : `${assessmentOffset + 1}–${assessmentOffset + assessments.length} of ${assessmentTotal}`}
            </span>
          </div>
          <ul className="divide-y divide-line border-y border-line">
            {visibleAssessments.map((item) => (
              <li key={item.id} className="flex flex-wrap items-center justify-between gap-4 py-4">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-ink">{item.title}</p>
                  <p className="mt-1 text-sm text-muted">
                    {item.author?.username ?? 'Unknown author'} · {item.kind} · {item.difficulty}
                  </p>
                  <p className="mt-1 text-xs text-dim">
                    {item.tags.map((tag) => tag.name).join(' · ') || 'No tags'}
                    {' · '}Updated {new Date(item.updatedAt).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-3">
                  <Link
                    to={`/assessments/${item.id}/preview`}
                    className="text-xs font-medium text-accent-light hover:text-ink"
                  >
                    View assessment
                  </Link>
                  <button
                    type="button"
                    onClick={() => void handleDeleteAssessment(item)}
                    disabled={busyId === `assessment-${item.id}`}
                    className="rounded-md border border-danger/40 px-3 py-2 text-xs font-medium text-danger-light hover:bg-danger/10 disabled:opacity-50"
                  >
                    Delete
                  </button>
                </div>
              </li>
            ))}
            {!loading && visibleAssessments.length === 0 && (
              <li className="py-10 text-center text-sm text-dim">No matching assessments</li>
            )}
          </ul>
          {assessmentTotal > 100 && (
            <div className="flex items-center justify-between py-4">
              <button
                type="button"
                onClick={() => {
                  const offset = Math.max(0, assessmentOffset - 100);
                  setAssessmentOffset(offset);
                  setLoading(true);
                  void loadData(offset);
                }}
                disabled={loading || assessmentOffset === 0}
                className="rounded-md border border-line-hover px-3 py-2 text-xs font-medium text-muted hover:bg-raised disabled:opacity-40"
              >
                Previous
              </button>
              <button
                type="button"
                onClick={() => {
                  const offset = assessmentOffset + 100;
                  setAssessmentOffset(offset);
                  setLoading(true);
                  void loadData(offset);
                }}
                disabled={loading || assessmentOffset + assessments.length >= assessmentTotal}
                className="rounded-md border border-line-hover px-3 py-2 text-xs font-medium text-muted hover:bg-raised disabled:opacity-40"
              >
                Next
              </button>
            </div>
          )}
        </section>
      )}
    </main>
  );
}

export default App;