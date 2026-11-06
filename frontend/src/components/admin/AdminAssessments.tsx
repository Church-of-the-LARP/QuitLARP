import { useState } from 'react';
import { Link } from 'react-router-dom';
import type { AdminAssessment } from '../../types/admin.ts';

type Props = {
  assessments: AdminAssessment[];
  busyId: number | string | null;
  onDelete: (assessment: AdminAssessment) => void;
  onSave: (
    assessment: AdminAssessment,
    changes: Pick<AdminAssessment, 'title' | 'description' | 'difficulty' | 'timeLimitMinutes'>,
  ) => Promise<boolean>;
  onLoadRepo: (assessment: AdminAssessment) => Promise<string>;
};

export default function AdminAssessments({
  assessments,
  busyId,
  onDelete,
  onSave,
  onLoadRepo,
}: Props) {
  const [draft, setDraft] = useState<Pick<
    AdminAssessment,
    'title' | 'description' | 'difficulty' | 'timeLimitMinutes'
  > | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [repoLink, setRepoLink] = useState<{ assessmentId: number; url: string } | null>(null);
  const [repoLoadingId, setRepoLoadingId] = useState<number | null>(null);
  const [repoError, setRepoError] = useState('');
  const [copiedAssessmentId, setCopiedAssessmentId] = useState<number | null>(null);

  const saveAssessment = async (assessment: AdminAssessment) => {
    if (draft && (await onSave(assessment, draft))) {
      setEditingId(null);
      setDraft(null);
    }
  };

  const editAssessment = async (assessment: AdminAssessment) => {
    setEditingId(assessment.id);
    setDraft({
      title: assessment.title,
      description: assessment.description,
      difficulty: assessment.difficulty,
      timeLimitMinutes: assessment.timeLimitMinutes,
    });
    setRepoError('');
    setRepoLoadingId(assessment.id);
    try {
      const url = await onLoadRepo(assessment);
      setRepoLink({ assessmentId: assessment.id, url });
    } catch (error) {
      setRepoLink(null);
      setRepoError(
        error instanceof Error ? error.message : 'Could not load repository link',
      );
    } finally {
      setRepoLoadingId(null);
    }
  };

  const copyRepoUrl = async () => {
    if (!repoLink) return;

    try {
      await navigator.clipboard?.writeText(repoLink.url);
    } catch {
      // The URL remains selectable if clipboard access is unavailable.
    }
    setCopiedAssessmentId(repoLink.assessmentId);
    window.setTimeout(() => setCopiedAssessmentId(null), 2000);
  };

  return (
    <section>
      <div className="mb-3 flex items-end justify-between">
        <div>
          <h2 className="text-xl font-semibold">Assessments</h2>
          <p className="mt-1 text-sm text-muted">
            Edit assessment details or remove an assessment and its nested content.
          </p>
        </div>
        <Link
          to="/create"
          className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell"
        >
          Create assessment
        </Link>
      </div>
      <div className="overflow-x-auto rounded-xl border border-line-hover bg-surface p-2">
        <table className="w-full min-w-[680px] border-separate border-spacing-y-2 text-left text-sm">
          <thead className="bg-panel text-xs uppercase text-muted">
            <tr>
              <th className="px-4 py-3">Assessment</th>
              <th className="px-4 py-3">Author</th>
              <th className="px-4 py-3">Difficulty</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {assessments.map((assessment) => (
              <tr key={assessment.id} className="bg-panel transition-colors hover:bg-raised">
                <td className="rounded-l-lg px-4 py-4">
                  <div className="font-medium">
                    {editingId === assessment.id ? (
                      <input
                        required
                        value={draft?.title ?? ''}
                        onChange={(event) =>
                          setDraft((current) =>
                            current ? { ...current, title: event.target.value } : current,
                          )
                        }
                        className="w-full rounded bg-panel px-2 py-1"
                        aria-label="Assessment title"
                      />
                    ) : (
                      assessment.title
                    )}
                  </div>
                  {editingId === assessment.id ? (
                    <div className="mt-2 flex flex-col gap-2">
                      <textarea
                        required
                        rows={3}
                        value={draft?.description ?? ''}
                        onChange={(event) =>
                          setDraft((current) =>
                            current
                              ? { ...current, description: event.target.value }
                              : current,
                          )
                        }
                        className="w-full rounded bg-panel px-2 py-1 text-sm"
                        aria-label="Assessment description"
                      />
                      <label className="flex items-center gap-2 text-xs text-muted">
                        Time limit (minutes)
                        <input
                          required
                          min={1}
                          type="number"
                          value={draft?.timeLimitMinutes ?? ''}
                          onChange={(event) =>
                            setDraft((current) =>
                              current
                                ? {
                                    ...current,
                                    timeLimitMinutes: Number(event.target.value),
                                  }
                                : current,
                            )
                          }
                          className="w-24 rounded bg-panel px-2 py-1"
                        />
                      </label>
                    </div>
                  ) : (
                    <p className="mt-1 line-clamp-2 text-xs text-muted">
                      {assessment.description}
                    </p>
                  )}
                  <div className="mt-1 text-xs text-muted">
                    #{assessment.id}
                    {editingId !== assessment.id && ` · ${assessment.timeLimitMinutes} min`}
                  </div>
                  {repoLink?.assessmentId === assessment.id && (
                    <div className="mt-3 flex items-center gap-2">
                      <input
                        readOnly
                        value={repoLink.url}
                        aria-label={`Clone URL for ${assessment.title}`}
                        onFocus={(event) => event.currentTarget.select()}
                        className="min-w-0 flex-1 rounded bg-panel px-2 py-1 font-mono text-xs text-ink"
                      />
                      <button
                        type="button"
                        onClick={() => void copyRepoUrl()}
                        className="shrink-0 rounded border border-line-hover bg-panel px-3 py-1 text-xs font-semibold text-muted hover:border-dim hover:text-ink"
                      >
                        {copiedAssessmentId === assessment.id ? 'Copied' : 'Copy'}
                      </button>
                    </div>
                  )}
                  {repoLoadingId === assessment.id && (
                    <p className="mt-2 text-xs text-muted">Loading Git repository link…</p>
                  )}
                  {repoError && editingId === assessment.id && (
                    <p role="alert" className="mt-2 text-xs text-danger">{repoError}</p>
                  )}
                </td>
                <td className="px-4 py-4 text-muted">
                  {assessment.author?.username ?? 'Deleted user'}
                </td>
                <td className="px-4 py-4 capitalize">
                  {editingId === assessment.id ? (
                    <select
                      value={draft?.difficulty ?? assessment.difficulty}
                      onChange={(event) =>
                        setDraft((current) =>
                          current
                            ? {
                                ...current,
                                difficulty: event.target.value as AdminAssessment['difficulty'],
                              }
                            : current,
                        )
                      }
                      className="rounded bg-panel px-2 py-1 capitalize"
                      aria-label="Assessment difficulty"
                    >
                      <option value="easy">Easy</option>
                      <option value="medium">Medium</option>
                      <option value="hard">Hard</option>
                    </select>
                  ) : (
                    assessment.difficulty
                  )}
                </td>
                <td className="rounded-r-lg px-4 py-4">
                  <div className="flex gap-2">
                    {editingId === assessment.id ? (
                      <>
                        <button
                          type="button"
                          disabled={busyId === assessment.id}
                          onClick={() => void saveAssessment(assessment)}
                          className="text-accent-light disabled:opacity-50"
                        >
                          {busyId === assessment.id ? 'Saving…' : 'Save'}
                        </button>
                        <button
                          type="button"
                          disabled={busyId === assessment.id}
                          onClick={() => {
                            setEditingId(null);
                            setDraft(null);
                            setRepoError('');
                          }}
                          className="text-muted"
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button
                        onClick={() => void editAssessment(assessment)}
                        disabled={repoLoadingId !== null}
                        className="text-accent-light"
                      >
                        Edit assessment
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
                <td colSpan={4} className="rounded-lg bg-panel px-4 py-10 text-center text-muted">
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
