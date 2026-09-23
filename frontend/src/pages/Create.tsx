import { useState } from 'react';
import type { FormEvent } from 'react';
import { Link } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api';
import Alert from '../components/Alert.tsx';
import FieldErrors from '../components/FieldErrors.tsx';
import usePageTitle from '../scripts/usePageTitle.ts';

type Difficulty = 'easy' | 'medium' | 'hard';

const chapterSchema = z.object({
  number: z.number(),
  title: z.string(),
  description: z.string(),
  timeLimitMinutes: z.string(),
});

const schema = z
  .object({
    title: z.string(),
    description: z.string(),
    difficulty: z.enum(['easy', 'medium', 'hard']),
    timeLimitMinutes: z.string(),
    templateFileName: z.string(),
    tags: z.string(),
    templateContent: z.string(),
    chapters: z.array(chapterSchema),
    testName: z.string(),
    testFileName: z.string(),
    testDescription: z.string(),
    testContent: z.string(),
  })
  .superRefine((value, ctx) => {
    if (!value.title.trim()) {
      ctx.addIssue({ code: 'custom', message: 'Required', path: ['title'] });
    }
    if (!value.description.trim()) {
      ctx.addIssue({ code: 'custom', message: 'Required', path: ['description'] });
    }
    if (!value.templateFileName.trim()) {
      ctx.addIssue({ code: 'custom', message: 'Required', path: ['templateFileName'] });
    }
    const time = Number(value.timeLimitMinutes);
    if (!Number.isFinite(time) || time <= 0) {
      ctx.addIssue({
        code: 'custom',
        message: 'Must be greater than 0',
        path: ['timeLimitMinutes'],
      });
    }

    value.chapters.forEach((chapter, index) => {
      const hasAny =
        chapter.title.trim() || chapter.description.trim() || chapter.timeLimitMinutes.trim();
      if (!hasAny) return;

      if (!chapter.title.trim()) {
        ctx.addIssue({ code: 'custom', message: 'Required', path: ['chapters', index, 'title'] });
      }
      if (!chapter.description.trim()) {
        ctx.addIssue({
          code: 'custom',
          message: 'Required',
          path: ['chapters', index, 'description'],
        });
      }
      const chapterTime = Number(chapter.timeLimitMinutes);
      if (!Number.isFinite(chapterTime) || chapterTime <= 0) {
        ctx.addIssue({
          code: 'custom',
          message: 'Must be greater than 0',
          path: ['chapters', index, 'timeLimitMinutes'],
        });
      }
    });

    const hasTestValues = !!(
      value.testName.trim() ||
      value.testDescription.trim() ||
      value.testFileName.trim() ||
      value.testContent.trim()
    );
    if (hasTestValues) {
      if (!value.testName.trim()) {
        ctx.addIssue({ code: 'custom', message: 'Required', path: ['testName'] });
      }
      if (!value.testDescription.trim()) {
        ctx.addIssue({ code: 'custom', message: 'Required', path: ['testDescription'] });
      }
      if (!value.testFileName.trim()) {
        ctx.addIssue({ code: 'custom', message: 'Required', path: ['testFileName'] });
      }
    }
  });

const inputClass =
  'w-full rounded-xl border border-line-hover bg-panel px-3 py-2 text-ink outline-none placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30';

type RepoState = {
  id: number;
  url: string | null;
  error: string | null;
};

export default function Create() {
  usePageTitle('Create assessment');
  const [title, setTitle] = useState('');
  const [titleError, setTitleError] = useState<string | null>(null);
  const [alertMsg, setAlertMsg] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [repo, setRepo] = useState<RepoState | null>(null);
  const [copied, setCopied] = useState(false);

  const copyRepoUrl = async () => {
    const url = repo?.url;
    if (!url) return;

    try {
      await navigator.clipboard?.writeText(url);
    } catch {
      // Clipboard access can be denied by the browser; the field stays selectable by hand.
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  const reset = () => {
    setTitle('');
    setTitleError(null);
    setAlertMsg(null);
    setRepo(null);
    setCopied(false);
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const name = title.trim();
    if (!name) {
      setTitleError('Required');
      return;
    }

    setTitleError(null);
    setAlertMsg(null);
    setSubmitting(true);

    try {
      const res = await client.POST('/api/v1/assessments', {
        body: {
          title: name,
          description: 'Maintained from the assessment repository.',
          difficulty: 'easy',
          timeLimitMinutes: 120,
          template: { fileName: 'main.py', content: '' },
        },
      });

      if (res.error) {
        setAlertMsg(getErrorText(res.error, 'Could not create the assessment'));
        return;
      }

      const assessmentId = res.data?.assessment.id;
      if (assessmentId === undefined) {
        setAlertMsg('Could not create the assessment');
        return;
      }

      setCopied(false);

      const repoRes = await client.GET('/api/v1/assessments/{id}/repo', {
        params: { path: { id: assessmentId } },
      });

      if (repoRes.error) {
        setRepo({
          id: assessmentId,
          url: null,
          error: getErrorText(repoRes.error, 'Could not load the repository link.'),
        });
        return;
      }

      setRepo({ id: assessmentId, url: repoRes.data?.url ?? null, error: null });
    } catch (err) {
      console.error('Failed to create assessment', err);
      setAlertMsg('Could not create the assessment');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="mx-auto w-[60vw] flex flex-col gap-5 py-6">
      <h1 className="font-bold text-3xl mb-6 text-ink">Create assessment</h1>

      {repo ? (
        <div className="rounded-xl border border-line-hover bg-surface p-6 flex flex-col gap-5">
          <div>
            <h2 className="text-lg font-semibold text-ink">Assessment created</h2>
            <p className="mt-1 text-sm text-muted">Assessment id {repo.id}</p>
          </div>

          {repo.error && <Alert kind="error" text={repo.error} />}

          {repo.url && (
            <div className="flex items-center gap-2">
              <input
                readOnly
                value={repo.url}
                aria-label="Clone URL"
                onFocus={(event) => event.currentTarget.select()}
                className={`${inputClass} font-mono text-sm`}
              />
              <button
                type="button"
                onClick={copyRepoUrl}
                className="shrink-0 rounded-xl border border-line-hover bg-panel px-3 py-2 text-sm font-semibold text-muted hover:border-dim hover:text-ink"
              >
                {copied ? 'Copied' : 'Copy'}
              </button>
            </div>
          )}

          <ul className="flex flex-col gap-1 text-sm text-muted">
            <li>Push the assessment content to the main branch of this repository.</li>
            <li>The README.md at the repository root becomes the assessment description.</li>
            <li>
              Each <code className="font-mono text-ink">chapters/&lt;index&gt;_&lt;name&gt;/</code>{' '}
              directory becomes a chapter.
            </li>
          </ul>

          <Link
            to={`/assessments/${repo.id}/git`}
            className="self-start text-sm font-semibold text-accent-light transition hover:text-accent-lighter"
          >
            Browse the repository
          </Link>

          <button
            type="button"
            onClick={reset}
            className="self-start rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted hover:border-dim hover:text-ink"
          >
            Create another assessment
          </button>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          <div className="rounded-xl border border-line-hover bg-surface p-6 flex flex-col gap-4">
            <div>
              <label className="mb-1 block font-semibold text-muted" htmlFor="assessment-name">
                Assessment name
              </label>
              <input
                id="assessment-name"
                name="title"
                type="text"
                autoComplete="off"
                value={title}
                placeholder="e.g. Binary search warmup"
                aria-invalid={titleError ? true : undefined}
                aria-describedby={titleError ? 'assessment-name-error' : undefined}
                onChange={(event) => {
                  setTitle(event.target.value);
                  if (titleError) setTitleError(null);
                }}
                className={inputClass}
              />
              {titleError && (
                <p id="assessment-name-error" role="alert" className="mt-1 text-xs text-danger">
                  {titleError}
                </p>
              )}
            </div>

            <p className="text-sm text-muted">
              The description and chapters come from the repository that is created together with the
              assessment.
            </p>
          </div>

          {alertMsg && <Alert kind="error" text={alertMsg} />}

          <button
            type="submit"
            disabled={submitting}
            className="self-center text-shell font-semibold px-6 py-2 bg-accent hover:bg-accent-light rounded-xl disabled:opacity-60"
          >
            {submitting ? 'Creating…' : 'Create assessment'}
          </button>
        </form>
      )}
    </main>
  );
}
