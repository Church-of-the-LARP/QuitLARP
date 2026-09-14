import type { AnyFieldApi } from '@tanstack/react-form';

export default function FieldErrors({ field }: { field: AnyFieldApi }) {
  if (field.state.meta.isValid) return null;
  if (!field.state.meta.isTouched) return null;

  return (
    <p role="alert" className="mt-1 text-xs text-rose-400">
      {field.state.meta.errors
        .map((e) => (typeof e === 'string' ? e : e?.message))
        .filter(Boolean)
        .join(', ')}
    </p>
  );
}
