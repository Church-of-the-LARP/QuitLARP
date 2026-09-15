import { useState } from "react";
import Editor from "@monaco-editor/react";

type Test = {
  id: string;
  title: string;
  code: string;
};

type Chapter = {
  id: string;
  title: string;
  tests: Test[];
};

let nextId = 0;
const makeId = () => `${Date.now()}-${nextId++}`;

const makeTest = (title: string): Test => ({
  id: makeId(),
  title,
  code: "// test code...\n",
});

const initialChapters: Chapter[] = [
  {
    id: makeId(),
    title: "Chapter 1",
    tests: [
      makeTest("LRU speedtest"),
      makeTest("correctness test"),
      makeTest("multiple threads"),
      makeTest("dangling pointers"),
      makeTest("data race"),
      makeTest("cache test"),
    ],
  },
  {
    id: makeId(),
    title: "Chapter 2",
    tests: [
      makeTest("test 1"),
      makeTest("test 2"),
      makeTest("test 3"),
      makeTest("test 4"),
      makeTest("test 5"),
    ],
  },
];

export default function Create() {
  const [title, setTitle] = useState("Assessment title");
  const [chapters, setChapters] = useState<Chapter[]>(initialChapters);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [selectedTestId, setSelectedTestId] = useState<string | null>(
    initialChapters[0]?.tests[1]?.id ?? null,
  );

  const selectedTest = chapters
    .flatMap((chapter) => chapter.tests)
    .find((test) => test.id === selectedTestId);

  const toggleChapter = (chapterId: string) => {
    setCollapsed((prev) => ({ ...prev, [chapterId]: !prev[chapterId] }));
  };

  const addChapter = () => {
    setChapters((prev) => [
      ...prev,
      { id: makeId(), title: `Chapter ${prev.length + 1}`, tests: [] },
    ]);
  };

  const renameChapter = (chapterId: string, newTitle: string) => {
    setChapters((prev) =>
      prev.map((chapter) =>
        chapter.id === chapterId ? { ...chapter, title: newTitle } : chapter,
      ),
    );
  };

  const addTest = (chapterId: string) => {
    const test = makeTest(`new test`);
    setChapters((prev) =>
      prev.map((chapter) =>
        chapter.id === chapterId
          ? { ...chapter, tests: [...chapter.tests, test] }
          : chapter,
      ),
    );
    setSelectedTestId(test.id);
  };

  const renameTest = (testId: string, newTitle: string) => {
    setChapters((prev) =>
      prev.map((chapter) => ({
        ...chapter,
        tests: chapter.tests.map((test) =>
          test.id === testId ? { ...test, title: newTitle } : test,
        ),
      })),
    );
  };

  const setTestCode = (testId: string, code: string) => {
    setChapters((prev) =>
      prev.map((chapter) => ({
        ...chapter,
        tests: chapter.tests.map((test) =>
          test.id === testId ? { ...test, code } : test,
        ),
      })),
    );
  };

  return (
    <main className="mx-auto flex h-[calc(100vh-65px)] w-[90vw] gap-5 py-6">
      <aside className="flex w-80 shrink-0 flex-col rounded-xl border border-zinc-700 bg-zinc-900">
        <div className="border-b border-zinc-700 px-4 py-4">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full bg-transparent text-xl font-bold text-zinc-100 outline-none focus:border-b focus:border-teal-500"
          />
        </div>

        <div className="flex-1 overflow-auto px-3 py-3">
          {chapters.map((chapter) => {
            const isCollapsed = collapsed[chapter.id];
            return (
              <div key={chapter.id} className="mb-4">
                <div className="flex items-center justify-between gap-2">
                  <button
                    onClick={() => toggleChapter(chapter.id)}
                    className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
                  >
                    <svg
                      viewBox="0 0 20 20"
                      fill="currentColor"
                      className={`h-3.5 w-3.5 shrink-0 text-zinc-500 transition-transform ${
                        isCollapsed ? "" : "rotate-90"
                      }`}
                    >
                      <path
                        fillRule="evenodd"
                        d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <input
                      value={chapter.title}
                      onChange={(e) =>
                        renameChapter(chapter.id, e.target.value)
                      }
                      onClick={(e) => e.stopPropagation()}
                      className="min-w-0 flex-1 truncate bg-transparent text-sm font-semibold text-zinc-100 outline-none focus:border-b focus:border-teal-500"
                    />
                    <span className="shrink-0 text-xs text-zinc-500">
                      ({chapter.tests.length})
                    </span>
                  </button>
                  <button
                    onClick={() => addTest(chapter.id)}
                    className="shrink-0 rounded-md border border-zinc-700 px-2 py-1 text-xs font-medium text-zinc-400 hover:bg-zinc-800"
                  >
                    new test
                  </button>
                </div>

                {!isCollapsed && (
                  <ul className="mt-1.5 ml-2 flex flex-col border-l border-zinc-700 pl-3">
                    {chapter.tests.map((test) => (
                      <li key={test.id}>
                        <button
                          onClick={() => setSelectedTestId(test.id)}
                          className={`w-full truncate rounded-md px-2 py-1.5 text-left text-sm transition ${
                            test.id === selectedTestId
                              ? "bg-teal-400/10 text-teal-400"
                              : "text-zinc-300 hover:bg-zinc-800"
                          }`}
                        >
                          {test.title}
                        </button>
                      </li>
                    ))}
                    {chapter.tests.length === 0 && (
                      <li className="px-2 py-1.5 text-sm text-zinc-500">
                        No tests yet
                      </li>
                    )}
                  </ul>
                )}
              </div>
            );
          })}
        </div>

        <div className="border-t border-zinc-700 p-3">
          <button
            onClick={addChapter}
            className="w-full rounded-lg border border-zinc-700 py-2 text-sm font-medium text-zinc-300 hover:bg-zinc-800"
          >
            new chapter
          </button>
        </div>
      </aside>

      <section className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-xl border border-zinc-700 bg-zinc-900">
        {selectedTest ? (
          <>
            <div className="border-b border-zinc-700 px-4 py-3">
              <input
                value={selectedTest.title}
                onChange={(e) => renameTest(selectedTest.id, e.target.value)}
                className="w-full bg-transparent text-sm font-semibold text-zinc-100 outline-none focus:border-b focus:border-teal-500"
              />
            </div>
            <div className="flex-1">
              <Editor
                height="100%"
                language="javascript"
                theme="vs-dark"
                value={selectedTest.code}
                onChange={(value) =>
                  setTestCode(selectedTest.id, value ?? "")
                }
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  scrollBeyondLastLine: false,
                }}
              />
            </div>
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center text-sm text-zinc-500">
            Select or create a test to start editing
          </div>
        )}
      </section>
    </main>
  );
}
