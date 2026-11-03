import type { AdminLiveSession, AdminSubmission, AdminTab } from '../../types/admin.ts';
import type { User } from '../../types/types.ts';

type Props = {
  users: User[];
  assessmentCount: number;
  sessions: AdminLiveSession[];
  submissions: AdminSubmission[];
  onSelect: (tab: AdminTab) => void;
};

function formatDate(value: string) {
  return new Date(value).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

function CountCard({
  label,
  value,
  detail,
  tab,
  onSelect,
}: {
  label: string;
  value: number;
  detail: string;
  tab: AdminTab;
  onSelect: (tab: AdminTab) => void;
}) {
  return (
    <button
      onClick={() => onSelect(tab)}
      className="rounded-xl border border-line-hover bg-surface p-5 text-left transition hover:border-accent/50 hover:bg-raised/50"
    >
      <div className="text-xs font-semibold uppercase tracking-wide text-muted">
        {label}
      </div>
      <div className="my-3 text-3xl font-semibold text-ink">{value}</div>
      <div className="text-sm text-muted">{detail}</div>
    </button>
  );
}

export default function AdminOverview({
  users,
  assessmentCount,
  sessions,
  submissions,
  onSelect,
}: Props) {
  const latestSubmissions = [...submissions]
    .sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))
    .slice(0, 5);
  const latestSessions = [...sessions]
    .sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))
    .slice(0, 4);
  const unverifiedUsers = users.filter((user) => !user.emailVerified).length;

  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Overview</h2>
        <p className="mt-1 text-sm text-muted">
          A snapshot of platform activity and account health.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <CountCard
          label="Total users"
          value={users.length}
          detail={`${unverifiedUsers} need email verification`}
          tab="users"
          onSelect={onSelect}
        />
        <CountCard
          label="Assessments"
          value={assessmentCount}
          detail="Published assessment records"
          tab="assessments"
          onSelect={onSelect}
        />
        <CountCard
          label="Live sessions"
          value={sessions.length}
          detail="Active gVisor environments"
          tab="sessions"
          onSelect={onSelect}
        />
        <CountCard
          label="Submissions"
          value={submissions.length}
          detail="All stored assessment solutions"
          tab="submissions"
          onSelect={onSelect}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="rounded-xl border border-line-hover bg-surface p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="font-semibold">Live gVisor sessions</h3>
              <p className="mt-1 text-sm text-muted">
                Currently running assessment environments
              </p>
            </div>
            <button
              onClick={() => onSelect('sessions')}
              className="text-sm font-medium text-accent-light hover:underline"
            >
              Manage sessions
            </button>
          </div>
          {latestSessions.length ? (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[560px] text-left text-sm">
                <thead className="text-xs uppercase text-muted">
                  <tr>
                    <th className="pb-2 pr-4 font-medium">Assessment</th>
                    <th className="pb-2 pr-4 font-medium">User</th>
                    <th className="pb-2 pr-4 font-medium">Runtime</th>
                    <th className="pb-2 font-medium">Started</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {latestSessions.map((session) => (
                    <tr key={session.id}>
                      <td className="py-3 pr-4 font-medium text-ink">
                        {session.assessmentTitle}
                      </td>
                      <td className="py-3 pr-4 text-muted">
                        User #{session.userId}
                      </td>
                      <td className="py-3 pr-4 text-muted">
                        {session.runtime}
                      </td>
                      <td className="py-3 text-muted">
                        {formatDate(session.createdAt)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="rounded-lg bg-panel px-4 py-8 text-center text-sm text-muted">
              No live sessions right now.
            </p>
          )}
        </section>

        <section className="rounded-xl border border-line-hover bg-surface p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="font-semibold">Recent submissions</h3>
              <p className="mt-1 text-sm text-muted">
                Latest submitted assessment solutions
              </p>
            </div>
            <button
              onClick={() => onSelect('submissions')}
              className="text-sm font-medium text-accent-light hover:underline"
            >
              View all
            </button>
          </div>
          {latestSubmissions.length ? (
            <ul className="divide-y divide-line">
              {latestSubmissions.map((submission) => (
                <li
                  key={submission.id}
                  className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-ink">
                      {submission.assessmentTitle}
                    </p>
                    <p className="mt-1 truncate text-xs text-muted">
                      {submission.author?.username ?? `User #${submission.userId}`}
                    </p>
                  </div>
                  <time
                    dateTime={submission.createdAt}
                    className="shrink-0 text-xs text-dim"
                  >
                    {formatDate(submission.createdAt)}
                  </time>
                </li>
              ))}
            </ul>
          ) : (
            <p className="rounded-lg bg-panel px-4 py-8 text-center text-sm text-muted">
              No submissions yet.
            </p>
          )}
        </section>
      </div>
    </section>
  );
}
