import type { AnyFieldApi } from '@tanstack/react-form';
import FieldErrors from './FieldErrors.tsx';

export default function TextField({
  field,
  label,
  type = 'text',
  autoComplete,
  placeholder,
}: {
  field: AnyFieldApi;
  label: string;
  type?: string;
  autoComplete?: string;
  placeholder?: string;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-muted">
        {label}
      </span>
      <input
        id={field.name}
        name={field.name}
        type={type}
        autoComplete={autoComplete}
        placeholder={placeholder}
        value={field.state.value}
        onBlur={field.handleBlur}
        onChange={(e) => field.handleChange(e.target.value)}
        className="w-full rounded-md border border-line-hover bg-surface px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-2 focus:ring-accent/30"
      />
      <FieldErrors field={field} />
    </label>
  );
}
