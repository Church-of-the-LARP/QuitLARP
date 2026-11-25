import { useEffect, useState } from "react";
import { searchUsers } from "../scripts/eventsApi.ts";

type Suggestion = { id: number; username: string };

type UsernameInputProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  id?: string;
  disabled?: boolean;
};

// UsernameInput is a text input that suggests usernames once the current token
// (the text after the last comma or space) reaches three characters. It is used
// wherever a host assigns people by username.
export default function UsernameInput({
  value,
  onChange,
  placeholder,
  className,
  id,
  disabled,
}: UsernameInputProps) {
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const current = /([^,\s]*)$/.exec(value)?.[1] ?? "";
  const query = current.trim();
  const prefix = value.slice(0, value.length - current.length);
  const selectedKey = prefix
    .split(/[,\s]+/)
    .map((name) => name.trim().toLowerCase())
    .filter(Boolean)
    .sort()
    .join(",");
  const selected = new Set(selectedKey ? selectedKey.split(",") : []);
  const visible = query.length >= 3 ? suggestions : [];

  useEffect(() => {
    if (query.length < 3) return;
    let live = true;
    const timer = setTimeout(() => {
      searchUsers(query)
        .then((users) => {
          if (!live) return;
          const alreadySelected = new Set(selectedKey.split(","));
          const remaining = users.filter(
            (user) => !alreadySelected.has(user.username.toLowerCase()),
          );
          setSuggestions(remaining);
          setActive(-1);
          setOpen(remaining.length > 0);
        })
        .catch(() => {
          if (!live) return;
          setSuggestions([]);
          setOpen(false);
        });
    }, 200);
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [query, selectedKey]);

  const choose = (username: string) => {
    if (selected.has(username.toLowerCase())) {
      setOpen(false);
      return;
    }
    onChange(`${prefix}${username}, `);
    setOpen(false);
    setActive(-1);
  };

  return (
    <div className="relative">
      <input
        id={id}
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        autoComplete="off"
        spellCheck={false}
        onChange={(event) => onChange(event.target.value)}
        onBlur={() => setOpen(false)}
        onKeyDown={(event) => {
          if (!open || visible.length === 0) return;
          if (event.key === "ArrowDown") {
            event.preventDefault();
            setActive((index) => (index + 1) % visible.length);
          } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setActive((index) =>
              index <= 0 ? visible.length - 1 : index - 1,
            );
          } else if (event.key === "Enter") {
            event.preventDefault();
            choose(visible[active >= 0 ? active : 0].username);
          } else if (event.key === "Escape") {
            setOpen(false);
          }
        }}
        className={className}
      />
      {open && visible.length > 0 && (
        <ul className="absolute left-0 right-0 top-full z-30 mt-1 overflow-hidden rounded-lg border border-line-hover bg-panel shadow-xl">
          {visible.map((user, index) => (
            <li key={user.id}>
              <button
                type="button"
                onMouseDown={(event) => {
                  event.preventDefault();
                  choose(user.username);
                }}
                onMouseEnter={() => setActive(index)}
                className={`block w-full px-3 py-1.5 text-left text-sm ${
                  index === active
                    ? "bg-raised text-ink"
                    : "text-muted hover:bg-raised/60 hover:text-ink"
                }`}
              >
                {user.username}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
