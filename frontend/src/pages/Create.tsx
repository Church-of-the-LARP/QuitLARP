import { useState, type FormEvent } from 'react';
import client from '../scripts/api';

type Difficulty = 'easy' | 'medium' | 'hard';

type ChapterDraft = {
  number: number;
  title: string;
  description: string;
  timeLimitMinutes: string;
};

export default function Create() {
  const [chapters, setChapters] = useState<ChapterDraft[]>([
    { number: 1, title: '', description: '', timeLimitMinutes: '' },
  ]);

  const addChapter = () => {
    setChapters((current) => {
      const nextNumber = Math.max(0, ...current.map((chapter) => chapter.number)) + 1;

      return [
        { number: nextNumber, title: '', description: '', timeLimitMinutes: '' },
        ...current,
      ];
    });
  };

  const removeChapter = (index: number) => {
    setChapters((current) => {
      if (current.length === 1) {
        return [{ number: 1, title: '', description: '', timeLimitMinutes: '' }];
      }

      return current.filter((_, chapterIndex) => chapterIndex !== index);
    });
  };

  const updateChapter = (
    index: number,
    field: keyof ChapterDraft,
    value: string,
  ) => {
    setChapters((current) =>
      current.map((chapter, chapterIndex) =>
        chapterIndex === index ? { ...chapter, [field]: value } : chapter,
      ),
    );
  };

  const handleCreateAssessment = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    const formData = new FormData(e.currentTarget);
    const title = String(formData.get('title') ?? '').trim();
    const description = String(formData.get('description') ?? '').trim();
    const difficulty = String(formData.get('difficulty') ?? 'easy') as Difficulty;
    const timeLimitMinutes = Number(formData.get('timeLimitMinutes'));
    const templateFileName = String(formData.get('templateFileName') ?? '').trim();
    const templateContent = String(formData.get('templateContent') ?? '');

    if (
      !title ||
      !description ||
      !difficulty ||
      !Number.isFinite(timeLimitMinutes) ||
      timeLimitMinutes <= 0 ||
      !templateFileName
    ) {
      return;
    }

    const tags = String(formData.get('tags') ?? '')
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean);

    const nonEmptyChapters = chapters.filter(
      (chapter) =>
        chapter.title.trim() || chapter.description.trim() || chapter.timeLimitMinutes.trim(),
    );

    for (const chapter of nonEmptyChapters) {
      if (
        !chapter.title.trim() ||
        !chapter.description.trim() ||
        !Number.isFinite(Number(chapter.timeLimitMinutes)) ||
        Number(chapter.timeLimitMinutes) <= 0
      ) {
        return;
      }
    }

    const chapterPayload = nonEmptyChapters.map((chapter) => ({
      title: chapter.title.trim(),
      description: chapter.description.trim(),
      timeLimitMinutes: Number(chapter.timeLimitMinutes),
    }));

    const testName = String(formData.get('testName') ?? '').trim();
    const testDescription = String(formData.get('testDescription') ?? '').trim();
    const testFileName = String(formData.get('testFileName') ?? '').trim();
    const testContent = String(formData.get('testContent') ?? '').trim();
    const hasTestValues = !!(
      testName ||
      testDescription ||
      testFileName ||
      testContent
    );

    if (hasTestValues) {
      if (!testName || !testDescription || !testFileName) {
        return;
      }
    }

    try {
      const res = await client.POST('/api/v1/assessments', {
        body: {
          title,
          description,
          difficulty,
          timeLimitMinutes,
          template: {
            fileName: templateFileName,
            content: templateContent,
          },
          ...(tags.length > 0 ? { tags } : {}),
          ...(chapterPayload.length > 0 ? { chapters: chapterPayload } : {}),
          ...(hasTestValues
            ? {
                tests: [
                  {
                    name: testName,
                    description: testDescription,
                    file: {
                      fileName: testFileName,
                      content: testContent,
                    },
                  },
                ],
              }
            : {}),
        },
      });

      console.log('Assessment created:', res.data?.assessment);
    } catch (err) {
      console.error('Failed to create assessment', err);
    }
  };

  return (
    <main className="mx-auto w-[60vw] flex flex-col gap-5 py-6">
      <h1 className="font-bold text-3xl mb-6 text-zinc-100">Create assessment</h1>

      <form onSubmit={handleCreateAssessment} className="flex flex-col gap-5">
        <div className="rounded-xl border border-zinc-700 bg-zinc-800 p-6 flex flex-col gap-8">
          <section>
            <h2 className="mb-4 text-lg font-semibold text-zinc-100">Assessment details</h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-zinc-200">Assessment title</label>
                <input
                  type="text"
                  name="title"
                  required
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="Title"
                />
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-zinc-200">Assessment description</label>
                <input
                  type="text"
                  name="description"
                  required
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="What is this assessment about?"
                />
              </div>

              <div>
                <label className="mb-1 block font-semibold text-zinc-200">Difficulty level</label>
                <select
                  name="difficulty"
                  defaultValue="easy"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                >
                  <option value="easy">Easy</option>
                  <option value="medium">Medium</option>
                  <option value="hard">Hard</option>
                </select>
              </div>

              <div>
                <label className="mb-1 block font-semibold text-zinc-200">
                  Time limit (minutes)
                </label>
                <input
                  type="number"
                  name="timeLimitMinutes"
                  required
                  min={1}
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="e.g. 60"
                />
              </div>

              <div>
                <label className="mb-1 block font-semibold text-zinc-200">
                  Starter template file name
                </label>
                <input
                  type="text"
                  name="templateFileName"
                  required
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="main.py"
                />
              </div>

              <div>
                <label className="mb-1 block font-semibold text-zinc-200">
                  Tags (comma separated)
                </label>
                <input
                  type="text"
                  name="tags"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="tag1, tag2, tag3"
                />
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-zinc-200">
                  Starter template content
                </label>
                <textarea
                  name="templateContent"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono text-sm text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="Paste the starting template code here"
                  rows={6}
                />
              </div>
            </div>
          </section>

          <section className="border-t border-zinc-700 pt-8">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-zinc-100">Chapters</h2>
              <button
                type="button"
                onClick={addChapter}
                className="text-zinc-950 px-3 py-1 bg-teal-500 hover:bg-teal-400 rounded-xl text-sm font-semibold"
              >
                + Add chapter
              </button>
            </div>

            <div className="flex flex-col gap-3">
              {chapters.map((chapter, index) => (
                <div
                  key={index}
                  className="rounded-xl border border-zinc-700 bg-zinc-900 p-4"
                >
                  <div className="mb-3 flex items-center justify-between">
                    <span className="font-semibold text-zinc-100">
                      Chapter {chapter.number}
                    </span>
                    {chapters.length > 1 && (
                      <button
                        type="button"
                        onClick={() => removeChapter(index)}
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
                      <input
                        type="text"
                        value={chapter.title}
                        onChange={(e) =>
                          updateChapter(index, 'title', e.target.value)
                        }
                        className="w-full rounded-xl border border-zinc-700 bg-zinc-800 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                        placeholder="Chapter title"
                      />
                    </div>

                    <div>
                      <label className="mb-1 block font-semibold text-zinc-200">
                        Chapter time limit (minutes)
                      </label>
                      <input
                        type="number"
                        min={1}
                        value={chapter.timeLimitMinutes}
                        onChange={(e) =>
                          updateChapter(index, 'timeLimitMinutes', e.target.value)
                        }
                        className="w-full rounded-xl border border-zinc-700 bg-zinc-800 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                        placeholder="Chapter time"
                      />
                    </div>

                    <div className="sm:col-span-2">
                      <label className="mb-1 block font-semibold text-zinc-200">
                        Chapter description
                      </label>
                      <input
                        type="text"
                        value={chapter.description}
                        onChange={(e) =>
                          updateChapter(index, 'description', e.target.value)
                        }
                        className="w-full rounded-xl border border-zinc-700 bg-zinc-800 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                        placeholder="Chapter description"
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </section>

          <section className="border-t border-zinc-700 pt-8">
            <h2 className="mb-4 text-lg font-semibold text-zinc-100">Optional hidden test</h2>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block font-semibold text-zinc-200">
                  Hidden test name
                </label>
                <input
                  type="text"
                  name="testName"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="e.g. Edge case checks"
                />
              </div>

              <div>
                <label className="mb-1 block font-semibold text-zinc-200">
                  Hidden test file name
                </label>
                <input
                  type="text"
                  name="testFileName"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="test.py"
                />
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-zinc-200">
                  Hidden test description
                </label>
                <input
                  type="text"
                  name="testDescription"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="What does this test verify?"
                />
              </div>

              <div className="sm:col-span-2">
                <label className="mb-1 block font-semibold text-zinc-200">
                  Hidden test file content
                </label>
                <textarea
                  name="testContent"
                  className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono text-sm text-zinc-100 outline-none placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
                  placeholder="Paste the hidden test file contents here"
                  rows={6}
                />
              </div>
            </div>
          </section>
        </div>

        <button
          type="submit"
          className="self-center text-zinc-950 font-semibold px-6 py-2 bg-teal-500 hover:bg-teal-400 rounded-xl"
        >
          Create assessment
        </button>
      </form>
    </main>
  );
}
