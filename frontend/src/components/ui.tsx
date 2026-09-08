import type { InputHTMLAttributes, ReactNode } from "react";
import type { Role } from "../auth/AuthContext";

/** Shared visual bits so the pages stay lean. */

export function Notice({ kind, children }: { kind: "error" | "notice"; children: ReactNode }) {
  const tone =
    kind === "error"
      ? "border-red-200 bg-red-50 text-red-700"
      : "border-blue-200 bg-blue-50 text-blue-700";
  return <div className={`rounded-md border px-3 py-2 text-sm ${tone}`}>{children}</div>;
}

export function Field({
  label,
  ...inputProps
}: { label: string } & InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
        {label}
      </span>
      <input
        {...inputProps}
        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
      />
    </label>
  );
}

export const roleStyles: Record<Role, string> = {
  user: "bg-gray-100 text-gray-700",
  admin: "bg-violet-100 text-violet-700",
  superadmin: "bg-amber-100 text-amber-800",
};

export function RoleBadge({ role }: { role: Role }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[role]}`}
    >
      {role}
    </span>
  );
}
