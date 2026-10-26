import type { PropertyConfiguration, PropertyOption } from "@/services/work-item-properties.service";
import { inputStyle } from "./controls";

type Props = {
  configuration: Pick<PropertyConfiguration, "levels"> & {
    options: Pick<PropertyOption, "id" | "name" | "parent" | "level" | "next_label">[];
  };
  value: unknown;
  onChange: (value: string[]) => void;
  disabled?: boolean;
  error?: string;
  controlId?: string;
  onControlMount?: (element: HTMLElement | null) => void;
};
export function CascadingValueInput({
  configuration,
  value,
  onChange,
  disabled,
  error,
  controlId,
  onControlMount,
}: Props) {
  const path = Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : [];
  return (
    <div className="space-y-2">
      {configuration.levels.map((level, index) => {
        const options = configuration.options.filter(
          (option) => option.level === index && option.parent === (path[index - 1] ?? "")
        );
        if (index > 0 && (!path[index - 1] || (options.length === 0 && !path[index]))) return null;
        const unavailable = Boolean(path[index] && !options.some((option) => option.id === path[index]));
        const label = configuration.options.find((option) => option.id === path[index - 1])?.next_label || level.name;
        return (
          <label key={level.id} className="block text-12 text-secondary">
            {label}
            {level.required && " *"}
            <select
              ref={index === 0 ? onControlMount : undefined}
              aria-describedby={error && controlId ? `${controlId}-error` : undefined}
              aria-invalid={Boolean(error) || unavailable}
              className={`${inputStyle} mt-1`}
              disabled={disabled}
              value={path[index] ?? ""}
              onChange={(event) =>
                onChange(event.target.value ? [...path.slice(0, index), event.target.value] : path.slice(0, index))
              }
            >
              <option value="">Select {label.toLowerCase()}</option>
              {unavailable && (
                <option value={path[index]} disabled>
                  Unavailable selection — choose another option
                </option>
              )}
              {options.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.name}
                </option>
              ))}
            </select>
          </label>
        );
      })}
    </div>
  );
}
