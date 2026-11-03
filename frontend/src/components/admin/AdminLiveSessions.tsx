import type { AdminLiveSession } from '../../types/admin.ts';

type Props = {
  sessions: AdminLiveSession[];
  busyId: number | string | null;
  onEnd: (session: AdminLiveSession) => void;
};

function date(value: string) {
  return new Date(value).toLocaleString();
}

export default function AdminLiveSessions({ sessions, busyId, onEnd }: Props) {
  return (
    <section>
      <h2 className="mb-3 text-xl font-semibold">Live assessment sessions</h2>
      <div className="overflow-x-auto rounded-xl">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-panel text-xs uppercase text-muted">
            <tr>
              <th className="px-4 py-3">Assessment / user</th>
              <th className="px-4 py-3">Runtime</th>
              <th className="px-4 py-3">Started</th>
              <th className="px-4 py-3">Deadline</th>
              <th className="px-4 py-3">Action</th>
            </tr>
          </thead>
          <tbody className=" bg-surface">
            {sessions.map((session) => (
              <tr key={session.id}>
                <td className="px-4 py-3">
                  <div>{session.assessmentTitle}</div>
                  <div className="mt-1 text-xs text-muted">
                    Assessment #{session.assessmentId} · User #{session.userId}
                  </div>
                </td>
                <td className="px-4 py-3">{session.runtime}</td>
                <td className="px-4 py-3 text-muted">
                  {date(session.createdAt)}
                </td>
                <td className="px-4 py-3 text-muted">
                  {date(session.deadline)}
                </td>
                <td className="px-4 py-3">
                  <button
                    disabled={busyId === session.id}
                    onClick={() => onEnd(session)}
                    className="text-danger-light disabled:opacity-50"
                  >
                    End session
                  </button>
                </td>
              </tr>
            ))}
            {sessions.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-10 text-center text-muted">
                  No live sessions.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
