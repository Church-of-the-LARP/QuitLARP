import type { ReactNode } from "react";

export function Notice({
  kind,
  children,
}: {
  kind: "error" | "notice";
  children: ReactNode;
}) {
  const tone =
    kind === "error"
      ? "border-red-200 bg-red-50 text-red-700"
      : "border-blue-200 bg-blue-50 text-blue-700";
  return (
    <div className={`rounded-md border px-3 py-2 text-sm ${tone}`}>
      {children}
    </div>
  );
}
