import { useEffect, useState, type FormEvent } from "react";
import createClient from "openapi-fetch";
import type { components, paths } from "./api/schema";

const client = createClient<paths>({ baseUrl: "http://localhost:8888" });

type User = components["schemas"]["User"];

function describeError(err: { detail?: unknown; status?: unknown }) {
  return String(err.detail ?? err.status ?? "unknown error");
}

function App() {
  const [users, setUsers] = useState<User[]>([]);
  const [name, setName] = useState("");
  const [message, setMessage] = useState<string | null>(null);

  async function loadUsers() {
    const res = await client.GET("/api/v1/users");
    if (res.error) {
      setMessage(`failed to load users: ${describeError(res.error)}`);
      return;
    }
    setMessage(null);
    setUsers(res.data?.users ?? []);
  }

  useEffect(() => {
    let cancelled = false;
    client.GET("/api/v1/users").then((res) => {
      if (cancelled) {
        return;
      }
      if (res.error) {
        setMessage(`failed to load users: ${describeError(res.error)}`);
        return;
      }
      setMessage(null);
      setUsers(res.data?.users ?? []);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      return;
    }
    const res = await client.POST("/api/v1/users", { body: { name: trimmed } });
    if (res.error) {
      setMessage(`failed to create user: ${describeError(res.error)}`);
      return;
    }
    setName("");
    await loadUsers();
  }

  return (
    <main className="mx-auto max-w-xl px-6 py-12">
      <h1 className="text-2xl font-semibold">Users</h1>
      <p className="mt-1 text-sm text-gray-500">
        Typed client generated from the backend OpenAPI spec
      </p>
      {message && <p className="mt-3 text-sm text-red-600">{message}</p>}
      <form className="mt-6 flex gap-2" onSubmit={submit}>
        <input
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="Name"
          className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm"
        />
        <button
          type="submit"
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white"
        >
          Add
        </button>
      </form>
      <ul className="mt-6 space-y-2">
        {users.map((user) => (
          <li
            key={user.id}
            className="rounded-md border border-gray-200 px-3 py-2 text-sm"
          >
            {user.name}
          </li>
        ))}
      </ul>
    </main>
  );
}

export default App;
