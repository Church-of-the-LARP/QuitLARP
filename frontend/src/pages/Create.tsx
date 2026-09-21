import { useState } from 'react';
import { useForm } from '@tanstack/react-form';
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
  'w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30';
const chapterInputClass =
  'w-full rounded-xl border border-zinc-700 bg-zinc-800 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30';

export default function Create() {
  const [alertMsg, setAlertMsg] = useState<string | null>(null);

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

        console.log('Assessment created:', res.data?.assessment);
      } catch (err) {
        console.error('Failed to create assessment', err);
        setAlertMsg('Could not create the assessment');
      }
    },
  });

  return (
    <main className="mx-auto w-[60vw] flex flex-col gap-5 py-6">
      <h1 className="font-bold text-3xl mb-6 text-zinc-100">Create assessment</h1>

      {alertMsg && <Alert kind="error" text={alertMsg} />}

      <form
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          form.handleSubmit();
        }}
        className="flex flex-col gap-5"
      >
        <div className="rounded-xl border border-zinc-700 bg-zinc-800 p-6 flex flex-col gap-8">
          <section>
            <h2 className="mb-4 text-lg font-semibold text-zinc-100">Assessment details</h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-zinc-200">Assessment title</label>
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
                <label className="mb-1 block font-semibold text-zinc-200">Assessment description</label>
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
                <label className="mb-1 block font-semibold text-zinc-200">Difficulty level</label>
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
                <label className="mb-1 block font-semibold text-zinc-200">
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
                <label className="mb-1 block font-semibold text-zinc-200">
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
                <label className="mb-1 block font-semibold text-zinc-200">
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
                <label className="mb-1 block font-semibold text-zinc-200">
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

          <section className="border-t border-zinc-700 pt-8">
            <form.Field name="chapters" mode="array">
              {(chaptersField) => (
                <>
                  <div className="mb-4 flex items-center justify-between">
                    <h2 className="text-lg font-semibold text-zinc-100">Chapters</h2>
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
                      className="text-zinc-950 px-3 py-1 bg-teal-500 hover:bg-teal-400 rounded-xl text-sm font-semibold"
                    >
                      + Add chapter
                    </button>
                  </div>

                  <div className="flex flex-col gap-3">
                    {chaptersField.state.value.map((chapter, index) => (
                      <div
                        key={index}
                        className="rounded-xl border border-zinc-700 bg-zinc-900 p-4"
                      >
                        <div className="mb-3 flex items-center justify-between">
                          <span className="font-semibold text-zinc-100">
                            Chapter {chapter.number}
                          </span>
                          {chaptersField.state.value.length > 1 && (
                            <button
                              type="button"
                              onClick={() => chaptersField.removeValue(index)}
                              className="text-sm text-rose-400 hover:text-rose-300"
                            >
                              Remove
                            </button>
                          )}
                        </div>

                        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                          <div>
                            <label className="mb-1 block font-semibold text-zinc-200">
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
                            <label className="mb-1 block font-semibold text-zinc-200">
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
                            <label className="mb-1 block font-semibold text-zinc-200">
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

          <section className="border-t border-zinc-700 pt-8">
            <h2 className="mb-4 text-lg font-semibold text-zinc-100">Optional hidden test</h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block font-semibold text-zinc-200">
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
                <label className="mb-1 block font-semibold text-zinc-200">
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
                <label className="mb-1 block font-semibold text-zinc-200">
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
                <label className="mb-1 block font-semibold text-zinc-200">
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
              className="self-center text-zinc-950 font-semibold px-6 py-2 bg-teal-500 hover:bg-teal-400 rounded-xl disabled:opacity-60"
            >
              {isSubmitting ? 'Creating…' : 'Create assessment'}
            </button>
          )}
        </form.Subscribe>
      </form>
    </main>
  );
}
