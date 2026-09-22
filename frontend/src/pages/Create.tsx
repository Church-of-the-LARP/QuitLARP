import { useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { Link } from 'react-router-dom';
import { z } from 'zod';
import client, { getErrorText } from '../scripts/api';
import Alert from '../components/Alert.tsx';
import FieldErrors from '../components/FieldErrors.tsx';

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
const chapterInputClass =
  'w-full rounded-xl border border-line-hover bg-surface px-3 py-2 text-ink outline-none placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30';

export default function Create() {
  const [alertMsg, setAlertMsg] = useState<string | null>(null);
  const [repoUrl, setRepoUrl] = useState<string | null>(null);
  const [repoForId, setRepoForId] = useState<number | null>(null);
  const [copied, setCopied] = useState(false);

  const copyRepoUrl = async () => {
    if (!repoUrl) return;
    await navigator.clipboard.writeText(repoUrl);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  const form = useForm({
    defaultValues: {
      title: '',
      description: '',
      difficulty: 'easy' as Difficulty,
      timeLimitMinutes: '',
      templateFileName: '',
      tags: '',
      templateContent: '',
      chapters: [{ number: 1, title: '', description: '', timeLimitMinutes: '' }],
      testName: '',
      testFileName: '',
      testDescription: '',
      testContent: '',
    },
    validators: {
      onChange: schema,
    },
    onSubmit: async ({ value }) => {
      setAlertMsg(null);

      const tags = value.tags
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean);

      const nonEmptyChapters = value.chapters.filter(
        (chapter) =>
          chapter.title.trim() || chapter.description.trim() || chapter.timeLimitMinutes.trim(),
      );
      const chapterPayload = nonEmptyChapters.map((chapter) => ({
        title: chapter.title.trim(),
        description: chapter.description.trim(),
        timeLimitMinutes: Number(chapter.timeLimitMinutes),
      }));

      const hasTestValues = !!(
        value.testName.trim() ||
        value.testDescription.trim() ||
        value.testFileName.trim() ||
        value.testContent.trim()
      );

      try {
        const res = await client.POST('/api/v1/assessments', {
          body: {
            title: value.title.trim(),
            description: value.description.trim(),
            difficulty: value.difficulty,
            timeLimitMinutes: Number(value.timeLimitMinutes),
            template: {
              fileName: value.templateFileName.trim(),
              content: value.templateContent,
            },
            ...(tags.length > 0 ? { tags } : {}),
            ...(chapterPayload.length > 0 ? { chapters: chapterPayload } : {}),
            ...(hasTestValues
              ? {
                  tests: [
                    {
                      name: value.testName.trim(),
                      description: value.testDescription.trim(),
                      file: {
                        fileName: value.testFileName.trim(),
                        content: value.testContent.trim(),
                      },
                    },
                  ],
                }
              : {}),
          },
        });

        if (res.error) {
          setAlertMsg(getErrorText(res.error, 'Could not create the assessment'));
          return;
        }

        const assessmentId = res.data?.assessment.id;
        if (assessmentId === undefined) return;

        const repo = await client.GET('/api/v1/assessments/{id}/repo', {
          params: { path: { id: assessmentId } },
        });
        if (repo.error) {
          setAlertMsg(getErrorText(repo.error, 'Could not load the repository link'));
          return;
        }
        setCopied(false);
        setRepoUrl(repo.data?.url ?? null);
        setRepoForId(assessmentId);
      } catch (err) {
        console.error('Failed to create assessment', err);
        setAlertMsg('Could not create the assessment');
      }
    },
  });

  return (
    <main className="mx-auto w-[60vw] flex flex-col gap-5 py-6">
      <button
        type="button"
        onClick={copyRepoUrl}
        disabled={!repoUrl}
        className={`flex w-full items-center gap-3 rounded-full border border-line-hover bg-panel px-4 py-2 text-left transition ${
          repoUrl ? 'cursor-pointer hover:border-dim' : 'cursor-default'
        }`}
      >
        <span
          className={`h-3 w-3 shrink-0 rounded-full ${repoUrl ? 'bg-accent' : 'bg-raised'}`}
        />
        <span
          className={`flex-1 truncate font-mono text-sm ${
            repoUrl ? 'text-muted' : 'text-dim'
          }`}
        >
          {repoUrl ?? 'Your git link appears here after you create the assessment'}
        </span>
        {repoUrl && (
          <span className="shrink-0 text-xs font-semibold text-muted">
            {copied ? 'Copied' : 'Copy'}
          </span>
        )}
      </button>

      {repoForId !== null && repoUrl && (
        <Link
          to={`/assessments/${repoForId}/git`}
          className="self-start text-sm font-semibold text-accent-light transition hover:text-accent-lighter"
        >
          Browse the files of this assessment repository
        </Link>
      )}

      <h1 className="font-bold text-3xl mb-6 text-ink">Create assessment</h1>

      {alertMsg && <Alert kind="error" text={alertMsg} />}

      <form
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          form.handleSubmit();
        }}
        className="flex flex-col gap-5"
      >
        <div className="rounded-xl border border-line-hover bg-surface p-6 flex flex-col gap-8">
          <section>
            <h2 className="mb-4 text-lg font-semibold text-ink">Assessment details</h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-muted">Assessment title</label>
                <form.Field name="title">
                  {(field) => (
                    <>
                      <input
                        type="text"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="Title"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-muted">Assessment description</label>
                <form.Field name="description">
                  {(field) => (
                    <>
                      <input
                        type="text"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="What is this assessment about?"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div>
                <label className="mb-1 block font-semibold text-muted">Difficulty level</label>
                <form.Field name="difficulty">
                  {(field) => (
                    <select
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value as Difficulty)}
                      className={inputClass}
                    >
                      <option value="easy">Easy</option>
                      <option value="medium">Medium</option>
                      <option value="hard">Hard</option>
                    </select>
                  )}
                </form.Field>
              </div>

              <div>
                <label className="mb-1 block font-semibold text-muted">
                  Time limit (minutes)
                </label>
                <form.Field name="timeLimitMinutes">
                  {(field) => (
                    <>
                      <input
                        type="number"
                        min={1}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="e.g. 60"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div>
                <label className="mb-1 block font-semibold text-muted">
                  Starter template file name
                </label>
                <form.Field name="templateFileName">
                  {(field) => (
                    <>
                      <input
                        type="text"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="main.py"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div>
                <label className="mb-1 block font-semibold text-muted">
                  Tags (comma separated)
                </label>
                <form.Field name="tags">
                  {(field) => (
                    <input
                      type="text"
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value)}
                      className={inputClass}
                      placeholder="tag1, tag2, tag3"
                    />
                  )}
                </form.Field>
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-muted">
                  Starter template content
                </label>
                <form.Field name="templateContent">
                  {(field) => (
                    <textarea
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value)}
                      className={`${inputClass} font-mono text-sm`}
                      placeholder="Paste the starting template code here"
                      rows={6}
                    />
                  )}
                </form.Field>
              </div>
            </div>
          </section>

          <section className="border-t border-line-hover pt-8">
            <form.Field name="chapters" mode="array">
              {(chaptersField) => (
                <>
                  <div className="mb-4 flex items-center justify-between">
                    <h2 className="text-lg font-semibold text-ink">Chapters</h2>
                    <button
                      type="button"
                      onClick={() => {
                        const nextNumber =
                          Math.max(
                            0,
                            ...chaptersField.state.value.map((chapter) => chapter.number),
                          ) + 1;
                        chaptersField.insertValue(0, {
                          number: nextNumber,
                          title: '',
                          description: '',
                          timeLimitMinutes: '',
                        });
                      }}
                      className="text-shell px-3 py-1 bg-accent hover:bg-accent-light rounded-xl text-sm font-semibold"
                    >
                      + Add chapter
                    </button>
                  </div>

                  <div className="flex flex-col gap-3">
                    {chaptersField.state.value.map((chapter, index) => (
                      <div
                        key={index}
                        className="rounded-xl border border-line-hover bg-panel p-4"
                      >
                        <div className="mb-3 flex items-center justify-between">
                          <span className="font-semibold text-ink">
                            Chapter {chapter.number}
                          </span>
                          {chaptersField.state.value.length > 1 && (
                            <button
                              type="button"
                              onClick={() => chaptersField.removeValue(index)}
                              className="text-sm text-danger hover:text-danger-light"
                            >
                              Remove
                            </button>
                          )}
                        </div>

                        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                          <div>
                            <label className="mb-1 block font-semibold text-muted">
                              Chapter title
                            </label>
                            <form.Field name={`chapters[${index}].title`}>
                              {(field) => (
                                <>
                                  <input
                                    type="text"
                                    value={field.state.value}
                                    onBlur={field.handleBlur}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className={chapterInputClass}
                                    placeholder="Chapter title"
                                  />
                                  <FieldErrors field={field} />
                                </>
                              )}
                            </form.Field>
                          </div>

                          <div>
                            <label className="mb-1 block font-semibold text-muted">
                              Chapter time limit (minutes)
                            </label>
                            <form.Field name={`chapters[${index}].timeLimitMinutes`}>
                              {(field) => (
                                <>
                                  <input
                                    type="number"
                                    min={1}
                                    value={field.state.value}
                                    onBlur={field.handleBlur}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className={chapterInputClass}
                                    placeholder="Chapter time"
                                  />
                                  <FieldErrors field={field} />
                                </>
                              )}
                            </form.Field>
                          </div>

                          <div className="sm:col-span-2">
                            <label className="mb-1 block font-semibold text-muted">
                              Chapter description
                            </label>
                            <form.Field name={`chapters[${index}].description`}>
                              {(field) => (
                                <>
                                  <input
                                    type="text"
                                    value={field.state.value}
                                    onBlur={field.handleBlur}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className={chapterInputClass}
                                    placeholder="Chapter description"
                                  />
                                  <FieldErrors field={field} />
                                </>
                              )}
                            </form.Field>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </form.Field>
          </section>

          <section className="border-t border-line-hover pt-8">
            <h2 className="mb-4 text-lg font-semibold text-ink">Optional hidden test</h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block font-semibold text-muted">
                  Hidden test name
                </label>
                <form.Field name="testName">
                  {(field) => (
                    <>
                      <input
                        type="text"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="e.g. Edge case checks"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div>
                <label className="mb-1 block font-semibold text-muted">
                  Hidden test file name
                </label>
                <form.Field name="testFileName">
                  {(field) => (
                    <>
                      <input
                        type="text"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="test.py"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-muted">
                  Hidden test description
                </label>
                <form.Field name="testDescription">
                  {(field) => (
                    <>
                      <input
                        type="text"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className={inputClass}
                        placeholder="What does this test verify?"
                      />
                      <FieldErrors field={field} />
                    </>
                  )}
                </form.Field>
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-muted">
                  Hidden test file content
                </label>
                <form.Field name="testContent">
                  {(field) => (
                    <textarea
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value)}
                      className={`${inputClass} font-mono text-sm`}
                      placeholder="Paste the hidden test file contents here"
                      rows={6}
                    />
                  )}
                </form.Field>
              </div>
            </div>
          </section>
        </div>

        <form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting]}>
          {([canSubmit, isSubmitting]) => (
            <button
              type="submit"
              disabled={!canSubmit}
              className="self-center text-shell font-semibold px-6 py-2 bg-accent hover:bg-accent-light rounded-xl disabled:opacity-60"
            >
              {isSubmitting ? 'Creating…' : 'Create assessment'}
            </button>
          )}
        </form.Subscribe>
      </form>
    </main>
  );
}
