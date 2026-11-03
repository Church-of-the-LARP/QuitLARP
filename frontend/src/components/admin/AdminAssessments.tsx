import { useState } from 'react';
import { Link } from 'react-router-dom';
import type { AdminAssessment } from '../../types/admin.ts';

type Props = {
  assessments: AdminAssessment[];
  busyId: number | string | null;
  onDelete: (assessment: AdminAssessment) => void;
  onSaveTitle: (assessment: AdminAssessment, title: string) => Promise<boolean>;
};

export default function AdminAssessments({
  assessments,
  busyId,
  onDelete,
  onSaveTitle,
}: Props) {
  const [editingId, setEditingId] = useState<number | null>(null);
  const [titleDraft, setTitleDraft] = useState('');

  const saveTitle = async (assessment: AdminAssessment) => {
    if (await onSaveTitle(assessment, titleDraft)) setEditingId(null);
  };

  return (
    <section>
      <div className="mb-3 flex items-end justify-between">
        <div>
          <h2 className="text-xl font-semibold">Assessments</h2>
          <p className="mt-1 text-sm text-muted">
            Management is at whole-assessment level; nested chapters and tests
            are not managed here.
          </p>
        </div>
        <Link
          to="/create"
          className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell"
        >
          Create assessment
        </Link>
      </div>
      <div className="overflow-x-auto rounded-xl border border-line-hover">
        <table className="w-full min-w-[680px] text-left text-sm">
          <thead className="bg-panel text-xs uppercase text-muted">
            <tr>
              <th className="px-4 py-3">Assessment</th>
              <th className="px-4 py-3">Author</th>
              <th className="px-4 py-3">Difficulty</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line bg-surface">
            {assessments.map((assessment) => (
              <tr key={assessment.id}>
                <td className="px-4 py-3">
                  <div className="font-medium">
                    {editingId === assessment.id ? (
                      <input
                        value={titleDraft}
                        onChange={(event) => setTitleDraft(event.target.value)}
                        className="w-full rounded border border-line-hover bg-panel px-2 py-1"
                        aria-label="Assessment title"
                      />
                    ) : (
                      assessment.title
                    )}
                  </div>
                  <div className="mt-1 text-xs text-muted">
                    #{assessment.id} · {assessment.timeLimitMinutes} min
                  </div>
                </td>
                <td className="px-4 py-3 text-muted">
                  {assessment.author?.username ?? 'Deleted user'}
                </td>
                <td className="px-4 py-3 capitalize">
                  {assessment.difficulty}
                </td>
                <td className="px-4 py-3">
                  <div className="flex gap-2">
                    {editingId === assessment.id ? (
                      <>
                        <button
                          disabled={busyId === assessment.id}
                          onClick={() => void saveTitle(assessment)}
                          className="text-accent-light disabled:opacity-50"
                        >
                          Save
                        </button>
                        <button
                          onClick={() => setEditingId(null)}
                          className="text-muted"
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button
                        onClick={() => {
                          setEditingId(assessment.id);
                          setTitleDraft(assessment.title);
                        }}
                        className="text-accent-light"
                      >
                        Edit title
                      </button>
                    )}
                    <button
                      disabled={busyId === assessment.id}
                      onClick={() => onDelete(assessment)}
                      className="text-danger-light disabled:opacity-50"
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {assessments.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-10 text-center text-muted">
                  No assessments found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
