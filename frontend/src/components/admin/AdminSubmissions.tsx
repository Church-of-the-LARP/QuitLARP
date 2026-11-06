import { useState } from 'react';
import type { AdminSubmission } from '../../types/admin.ts';

type Props = {
  submissions: AdminSubmission[];
};

function date(value: string) {
  return new Date(value).toLocaleString();
}

export default function AdminSubmissions({ submissions }: Props) {
  const [copiedId, setCopiedId] = useState<number | null>(null);

  const copyUrl = async (submission: AdminSubmission) => {
    try {
      await navigator.clipboard?.writeText(submission.url);
    } catch {
      // The URL remains selectable if clipboard access is unavailable.
    }
    setCopiedId(submission.id);
    window.setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <section>
      <h2 className="mb-3 text-xl font-semibold">Submitted assessments</h2>
      <div className="overflow-x-auto rounded-xl">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-panel text-xs uppercase text-muted">
            <tr>
              <th className="px-4 py-3">Assessment</th>
              <th className="px-4 py-3">Submitted by</th>
              <th className="px-4 py-3">Visibility</th>
              <th className="px-4 py-3">Updated</th>
              <th className="px-4 py-3">Repository</th>
            </tr>
          </thead>
          <tbody className=" bg-surface">
            {submissions.map((submission) => (
              <tr key={submission.id}>
                <td className="px-4 py-3">
                  {submission.assessmentTitle}
                  <div className="mt-1 text-xs text-muted">
                    Assessment #{submission.assessmentId} ·{' '}
                    {submission.commitSha.slice(0, 10)}
                  </div>
                </td>
                <td className="px-4 py-3">
                  {submission.author?.username ?? `User #${submission.userId}`}
                </td>
                <td className="px-4 py-3">
                  {submission.public ? 'Public' : 'Private'}
                </td>
                <td className="px-4 py-3 text-muted">
                  {date(submission.updatedAt)}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <input
                      readOnly
                      value={submission.url}
                      aria-label={`Clone URL for submission ${submission.id}`}
                      onFocus={(event) => event.currentTarget.select()}
                      className="min-w-0 flex-1 rounded bg-panel px-2 py-1 font-mono text-xs text-ink"
                    />
                    <button
                      type="button"
                      onClick={() => void copyUrl(submission)}
                      className="shrink-0 rounded border border-line-hover bg-panel px-3 py-1 text-xs font-semibold text-muted hover:border-dim hover:text-ink"
                    >
                      {copiedId === submission.id ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {submissions.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-10 text-center text-muted">
                  No submitted assessments.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
