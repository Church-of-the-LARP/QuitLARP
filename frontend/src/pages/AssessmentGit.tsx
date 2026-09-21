import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api';

type Entry = {
  name: string;
  type: string;
  size: number;
  hash: string;
};

type FileView = {
  path: string;
  size: number;
  binary: boolean;
  truncated: boolean;
  content: string;
};

const joinPath = (base: string, name: string) => (base ? `${base}/${name}` : name);

const formatSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

export default function AssessmentGit() {
  const params = useParams();
  const id = Number(params.id);

  const [path, setPath] = useState('');
  const [ref, setRef] = useState('');
  const [entries, setEntries] = useState<Entry[]>([]);
  const [empty, setEmpty] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [forbidden, setForbidden] = useState(false);

  const [fileTarget, setFileTarget] = useState<string | null>(null);
  const [file, setFile] = useState<FileView | null>(null);
  const [fileLoading, setFileLoading] = useState(false);
  const [fileError, setFileError] = useState<string | null>(null);

  useEffect(() => {
    if (!Number.isInteger(id) || id <= 0) {
      setError('This assessment id is not valid');
      setLoading(false);
      return;
    }
    let active = true;
    setLoading(true);
    setError(null);
    setForbidden(false);
    client
      .GET('/api/v1/assessments/{id}/repo/tree', {
        params: { path: { id }, query: { path } },
      })
      .then((res) => {
        if (!active) return;
        if (res.error) {
          if (res.response.status === 403) setForbidden(true);
          else setError(getErrorText(res.error, 'Could not load the repository'));
          return;
        }
        setRef(res.data?.ref ?? '');
        setEntries((res.data?.entries ?? []) as Entry[]);
        setEmpty(res.data?.empty ?? false);
      })
      .catch(() => {
        if (active) setError('Could not load the repository');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id, path]);

  useEffect(() => {
    if (fileTarget === null) return;
    let active = true;
    setFileLoading(true);
    setFileError(null);
    setFile(null);
    client
      .GET('/api/v1/assessments/{id}/repo/file', {
        params: { path: { id }, query: { path: fileTarget } },
      })
      .then((res) => {
        if (!active) return;
        if (res.error) {
          setFileError(getErrorText(res.error, 'Could not load the file'));
          return;
        }
        setFile({
          path: res.data?.path ?? fileTarget,
          size: res.data?.size ?? 0,
          binary: res.data?.binary ?? false,
          truncated: res.data?.truncated ?? false,
          content: res.data?.content ?? '',
        });
      })
      .catch(() => {
        if (active) setFileError('Could not load the file');
      })
      .finally(() => {
        if (active) setFileLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id, fileTarget]);

  const segments = path ? path.split('/').filter(Boolean) : [];

  const openFile = (name: string) => {
    setFileTarget(joinPath(path, name));
  };

  const backToTree = () => {
    setFileTarget(null);
    setFile(null);
    setFileError(null);
  };

  return (
    <main className="mx-auto flex w-[60vw] flex-col gap-5 py-6">
      <div>
        <p className="mb-2 text-xs font-semibold uppercase text-teal-400">Repository</p>
        <h1 className="text-3xl font-bold text-zinc-100">Files</h1>
      </div>

      {forbidden ? (
        <div className="rounded-xl border border-zinc-700 bg-zinc-800 p-6 text-zinc-300">
          Only the author or an admin can browse this repository.
        </div>
      ) : error ? (
        <div className="rounded-xl border border-zinc-700 bg-zinc-800 p-6 text-rose-400">{error}</div>
      ) : (
        <div className="flex flex-col gap-4 rounded-xl border border-zinc-700 bg-zinc-800 p-6">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <button
              type="button"
              onClick={() => {
                backToTree();
                setPath('');
              }}
              className="font-mono font-semibold text-teal-400 transition hover:text-teal-300"
            >
              {ref || 'repository'}
            </button>
            {segments.map((segment, index) => (
              <span key={`${segment}-${index}`} className="flex items-center gap-2">
                <span className="text-zinc-600">/</span>
                <button
                  type="button"
                  onClick={() => {
                    backToTree();
                    setPath(segments.slice(0, index + 1).join('/'));
                  }}
                  className="font-mono text-zinc-300 transition hover:text-teal-300"
                >
                  {segment}
                </button>
              </span>
            ))}
          </div>

          {loading ? (
            <p className="text-zinc-400">Loading...</p>
          ) : empty ? (
            <p className="text-zinc-400">No commits yet. Push to main to see files here.</p>
          ) : fileTarget !== null ? (
            <div className="flex flex-col gap-3">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <button
                  type="button"
                  onClick={backToTree}
                  className="text-sm font-semibold text-teal-400 transition hover:text-teal-300"
                >
                  &larr; Back to files
                </button>
                <span className="font-mono text-xs text-zinc-500">
                  {file ? `${formatSize(file.size)} - ${file.path}` : fileTarget}
                </span>
              </div>

              {fileLoading ? (
                <p className="text-zinc-400">Loading...</p>
              ) : fileError ? (
                <p className="text-rose-400">{fileError}</p>
              ) : file?.binary ? (
                <p className="text-zinc-400">
                  This file looks binary and cannot be displayed.
                </p>
              ) : file?.truncated ? (
                <p className="text-zinc-400">
                  This file is too large to display ({formatSize(file.size)}).
                </p>
              ) : (
                <pre className="max-h-[70vh] overflow-auto rounded-lg border border-zinc-700 bg-zinc-900 p-4 font-mono text-sm leading-relaxed text-zinc-200">
                  {file?.content ?? ''}
                </pre>
              )}
            </div>
          ) : entries.length === 0 ? (
            <p className="text-zinc-400">This directory is empty.</p>
          ) : (
            <ul className="flex flex-col divide-y divide-zinc-700">
              {entries.map((entry) => (
                <li key={entry.name}>
                  {entry.type === 'tree' ? (
                    <button
                      type="button"
                      onClick={() => {
                        setPath(joinPath(path, entry.name));
                      }}
                      className="flex w-full items-center gap-3 px-1 py-2 text-left transition hover:bg-zinc-700/40"
                    >
                      <span className="font-mono text-sm text-teal-400">{entry.name}/</span>
                      <span className="ml-auto text-xs text-zinc-500">directory</span>
                    </button>
                  ) : (
                    <button
                      type="button"
                      onClick={() => openFile(entry.name)}
                      className="flex w-full items-center gap-3 px-1 py-2 text-left transition hover:bg-zinc-700/40"
                    >
                      <span className="font-mono text-sm text-zinc-200">{entry.name}</span>
                      <span className="ml-auto font-mono text-xs text-zinc-500">
                        {formatSize(entry.size)}
                      </span>
                    </button>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <Link to="/create" className="text-sm font-semibold text-teal-400 hover:text-teal-300">
        Create another assessment
      </Link>
    </main>
  );
}
