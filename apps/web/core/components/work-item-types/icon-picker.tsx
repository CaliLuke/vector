import { Popover, PopoverButton, PopoverPanel } from "@headlessui/react";
import { Check, Search, X } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import { DEFAULT_COLORS } from "@plane/propel/emoji-icon-picker";
import { typeIcons } from "./type-icons";

export function IconPicker({
  value,
  onChange,
  disabled,
  trigger,
  label,
  color,
  allowClear = false,
}: {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  trigger: ReactNode;
  label: string;
  color?: { value: string; onChange: (value: string) => void };
  allowClear?: boolean;
}) {
  const [initialColor] = useState(color?.value.toLowerCase());
  return (
    <Popover>
      <PopoverButton
        aria-label={label}
        disabled={disabled}
        className="flex size-9 shrink-0 items-center justify-center rounded-md outline-none hover:bg-layer-2 focus-visible:ring-2 focus-visible:ring-accent-strong"
      >
        {trigger}
      </PopoverButton>
      <PopoverPanel
        anchor="bottom start"
        portal
        focus
        aria-label={color ? "Icon and color" : "Icon"}
        className="z-50 mt-2 flex max-h-[var(--anchor-max-height)] w-90 max-w-[calc(100vw-2rem)] flex-col overflow-hidden rounded-lg border border-subtle bg-surface-1 shadow-raised-200 outline-none"
      >
        {({ close }) => (
          <IconSheet
            value={value}
            initialColor={initialColor}
            color={color}
            allowClear={allowClear}
            disabled={disabled}
            onChange={onChange}
            onClose={() => close()}
          />
        )}
      </PopoverPanel>
    </Popover>
  );
}

function IconSheet({
  value,
  initialColor,
  onChange,
  onClose,
  color,
  allowClear,
  disabled,
}: {
  value: string;
  initialColor?: string;
  onChange: (value: string) => void;
  onClose: () => void;
  color?: { value: string; onChange: (value: string) => void };
  allowClear: boolean;
  disabled?: boolean;
}) {
  const [query, setQuery] = useState("");
  const id = useId();
  const icons = typeIcons.filter((icon) =>
    `${icon.name} ${icon.label}`.toLowerCase().includes(query.trim().toLowerCase())
  );
  const colors = [...new Set([...DEFAULT_COLORS, "#8b5cf6", ...(initialColor ? [initialColor] : [])])];
  return (
    <>
      <div className="flex shrink-0 items-center gap-2 border-b border-subtle px-3 py-2">
        <Search className="size-4 shrink-0 text-tertiary" aria-hidden />
        <input
          data-autofocus
          aria-label="Search icons"
          placeholder="Search icons"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          className="min-w-0 flex-1 bg-transparent py-1 text-13 text-primary outline-none"
        />
        <button
          type="button"
          aria-label="Close icon picker"
          onClick={onClose}
          className="rounded p-1 text-tertiary hover:bg-layer-2 focus-visible:outline-accent-strong"
        >
          <X className="size-4" aria-hidden />
        </button>
      </div>
      <div
        role="radiogroup"
        aria-label="Icons"
        className="grid max-h-56 min-h-0 grid-cols-8 gap-1 overflow-y-auto overscroll-contain p-3"
      >
        {allowClear && !query && (
          <label className="relative flex aspect-square cursor-pointer items-center justify-center rounded-md text-11 focus-within:ring-2 focus-within:ring-accent-strong hover:bg-layer-2">
            <input
              type="radio"
              name={`${id}-icon`}
              aria-label="Use default icon"
              checked={!value}
              disabled={disabled}
              onChange={() => onChange("")}
              className="peer sr-only"
            />
            <span className="absolute inset-0 rounded-md peer-checked:bg-layer-2 peer-checked:ring-1 peer-checked:ring-accent-strong" />
            <span className="relative">Default</span>
          </label>
        )}
        {icons.map(({ name, label, element: Icon }) => (
          <label
            key={name}
            title={label}
            className="relative flex aspect-square cursor-pointer items-center justify-center rounded-md focus-within:ring-2 focus-within:ring-accent-strong hover:bg-layer-2"
          >
            <input
              type="radio"
              name={`${id}-icon`}
              aria-label={label}
              value={name}
              checked={value === name}
              disabled={disabled}
              onChange={() => onChange(name)}
              className="peer sr-only"
            />
            <span className="absolute inset-0 rounded-md peer-checked:bg-layer-2 peer-checked:ring-1 peer-checked:ring-accent-strong" />
            <Icon className="relative size-4" style={{ color: color?.value }} aria-hidden />
            {value === name && (
              <Check
                className="absolute top-0 right-0 size-2.5 rounded-sm bg-surface-1 text-accent-primary"
                aria-hidden
              />
            )}
          </label>
        ))}
        {icons.length === 0 && (
          <output className="col-span-8 py-8 text-center text-13 text-tertiary">No icons found.</output>
        )}
      </div>
      {color && (
        <div className="shrink-0 border-t border-subtle p-3">
          <p id={`${id}-colors`} className="mb-2 text-12 font-medium text-secondary">
            Color
          </p>
          <div role="radiogroup" aria-labelledby={`${id}-colors`} className="grid auto-cols-fr grid-flow-col gap-2">
            {colors.map((swatch) => (
              <label
                key={swatch}
                title={swatch}
                className="relative flex aspect-square cursor-pointer items-center justify-center rounded-md focus-within:ring-2 focus-within:ring-accent-strong focus-within:ring-offset-2"
                style={{ backgroundColor: swatch }}
              >
                <input
                  type="radio"
                  name={`${id}-color`}
                  aria-label={`Color ${swatch}`}
                  value={swatch}
                  checked={color.value.toLowerCase() === swatch}
                  disabled={disabled}
                  onChange={() => color.onChange(swatch)}
                  className="peer sr-only"
                />
                <span className="absolute -inset-0.5 rounded-lg peer-checked:ring-1 peer-checked:ring-strong" />
                {color.value.toLowerCase() === swatch && (
                  <Check
                    className="size-4 text-white drop-shadow-[0_1px_1px_rgba(0,0,0,0.8)]"
                    strokeWidth={3}
                    aria-hidden
                  />
                )}
              </label>
            ))}
          </div>
        </div>
      )}
    </>
  );
}
