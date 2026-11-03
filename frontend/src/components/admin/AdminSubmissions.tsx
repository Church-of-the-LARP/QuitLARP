import type { AdminSubmission } from '../../types/admin.ts';

type Props = {
  submissions: AdminSubmission[];
};

function date(value: string) {
  return new Date(value).toLocaleString();
}

export default function AdminSubmissions({ submissions }: Props) {
  return (
    <section>
      <h2 className="mb-3 text-xl font-semibold">Submitted assessments</h2>
      <div className="overflow-x-auto rounded-xl border border-line-hover">
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
          <tbody className="divide-y divide-line bg-surface">
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
                  <a
                    href={submission.url}
                    className="text-accent-light hover:underline"
                  >
                    Clone URL
                  </a>
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
