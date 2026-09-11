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
    <form onSubmit={handleCreateAssessment} className="space-y-4">
      <div>
        <label className="mb-1 block font-medium">Title</label>
        <input
          type="text"
          name="title"
          required
          className="w-90 rounded-xl border px-3 py-2"
          placeholder="Title"
        />
      </div>

      <div>
        <label className="mb-1 block font-medium">Description</label>
        <input
          type="text"
          name="description"
          required
          className="w-90 rounded-xl border px-3 py-2"
          placeholder="Description"
        />
      </div>

      <div>
        <label className="mb-1 block font-medium">Difficulty</label>
        <select
          name="difficulty"
          defaultValue="easy"
          className="w-90 rounded-xl border px-3 py-2"
        >
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </select>
      </div>

      <div>
        <label className="mb-1 block font-medium">Time limit in minutes</label>
        <input
          type="number"
          name="timeLimitMinutes"
          required
          min={1}
          className="w-90 rounded-xl border px-3 py-2"
          placeholder="Time in minutes"
        />
      </div>

      <div>
        <label className="mb-1 block font-medium">Template file name</label>
        <input
          type="text"
          name="templateFileName"
          required
          className="w-90 rounded-xl border px-3 py-2"
          placeholder="main.py"
        />
      </div>

      <div>
        <label className="mb-1 block font-medium">Template content</label>
        <textarea
          name="templateContent"
          className="w-90 rounded-xl border px-3 py-2"
          placeholder="Paste the starting template code here"
          rows={6}
        />
      </div>

      <div>
        <label className="mb-1 block font-medium">Tags</label>
        <input
          type="text"
          name="tags"
          className="w-90 rounded-xl border px-3 py-2"
          placeholder="tag1, tag2, tag3"
        />
      </div>

      <div className="rounded-xl border p-4">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-lg font-semibold">Chapters</h3>
          <button
            type="button"
            onClick={addChapter}
            className="rounded-xl border border-blue-600 px-3 py-1 text-sm text-blue-600"
          >
            + Add chapter
          </button>
        </div>

        {chapters.map((chapter, index) => (
          <div key={index} className="mb-4 rounded-xl border p-3 last:mb-0">
            <div className="mb-3 flex items-center justify-between">
              <span className="font-medium">Chapter {chapter.number}</span>
              {chapters.length > 1 && (
                <button
                  type="button"
                  onClick={() => removeChapter(index)}
                  className="text-sm text-red-600"
                >
                  Remove
                </button>
              )}
            </div>

            <div>
              <label className="mb-1 block font-medium">Chapter title</label>
              <input
                type="text"
                value={chapter.title}
                onChange={(e) => updateChapter(index, 'title', e.target.value)}
                className="w-90 rounded-xl border px-3 py-2"
                placeholder="Chapter title"
              />
            </div>

            <div className="mt-3">
              <label className="mb-1 block font-medium">Chapter description</label>
              <input
                type="text"
                value={chapter.description}
                onChange={(e) => updateChapter(index, 'description', e.target.value)}
                className="w-90 rounded-xl border px-3 py-2"
                placeholder="Chapter description"
              />
            </div>

            <div className="mt-3">
              <label className="mb-1 block font-medium">Chapter time limit in minutes</label>
              <input
                type="number"
                min={1}
                value={chapter.timeLimitMinutes}
                onChange={(e) => updateChapter(index, 'timeLimitMinutes', e.target.value)}
                className="w-90 rounded-xl border px-3 py-2"
                placeholder="Chapter time"
              />
            </div>
          </div>
        ))}
      </div>

      <div className="rounded-xl border p-4">
        <h3 className="mb-3 text-lg font-semibold">Optional hidden test</h3>

        <div>
          <label className="mb-1 block font-medium">Test name</label>
          <input
            type="text"
            name="testName"
            className="w-90 rounded-xl border px-3 py-2"
            placeholder="Test name"
          />
        </div>

        <div className="mt-3">
          <label className="mb-1 block font-medium">Test description</label>
          <input
            type="text"
            name="testDescription"
            className="w-90 rounded-xl border px-3 py-2"
            placeholder="Test description"
          />
        </div>

        <div className="mt-3">
          <label className="mb-1 block font-medium">Test file name</label>
          <input
            type="text"
            name="testFileName"
            className="w-90 rounded-xl border px-3 py-2"
            placeholder="test.py"
          />
        </div>

        <div className="mt-3">
          <label className="mb-1 block font-medium">Test file content</label>
          <textarea
            name="testContent"
            className="w-90 rounded-xl border px-3 py-2"
            placeholder="Test file content"
            rows={6}
          />
        </div>
      </div>

      <button type="submit" className="rounded-xl bg-blue-600 px-4 py-2 text-white">
        Create assessment
      </button>
    </form>
  );
}
