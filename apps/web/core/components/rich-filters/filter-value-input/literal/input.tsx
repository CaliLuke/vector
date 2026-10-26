import { Popover, PopoverButton, PopoverPanel } from "@headlessui/react";
import { useForm } from "@tanstack/react-form";
import { useId } from "react";
import type { KeyboardEvent } from "react";
import type { TFilterConditionNodeForDisplay, TFilterValue, TLiteralFilterFieldConfig } from "@plane/types";
import { literalDraft, literalLabel, parseLiteralValue } from "./value";
import type { LiteralValue } from "./value";

type Props = {
  condition: TFilterConditionNodeForDisplay<string, TFilterValue>;
  config: TLiteralFilterFieldConfig;
  isDisabled?: boolean;
  onChange: (value: LiteralValue) => void;
};

export function LiteralFilterValueInput(props: Props) {
  const { condition, config, isDisabled } = props;
  return (
    <Popover className="relative flex shrink-0">
      <PopoverButton
        disabled={isDisabled}
        className="max-w-60 truncate px-2 text-13 text-primary hover:bg-layer-2 focus-visible:outline-accent-strong"
      >
        {literalLabel(condition.value)}
      </PopoverButton>
      {!isDisabled && (
        <PopoverPanel
          anchor="bottom start"
          portal
          focus
          className="z-50 mt-1 w-72 max-w-[calc(100vw-2rem)] rounded-md border border-subtle bg-surface-1 p-3 shadow-raised-200"
        >
          {({ close }) => (
            <LiteralEditor
              key={`${condition.id}:${condition.operator}:${config.type}:${JSON.stringify(condition.value)}`}
              {...props}
              onClose={close}
            />
          )}
        </PopoverPanel>
      )}
    </Popover>
  );
}

function LiteralEditor({ condition, config, onChange, onClose }: Props & { onClose: () => void }) {
  const id = useId();
  const range = config.type === "number_range";
  const numeric = config.type !== "text";
  const form = useForm({
    defaultValues: literalDraft(condition.value),
    validators: { onSubmit: ({ value }) => parseLiteralValue(config.type, value).error },
    onSubmit: ({ value }) => {
      const parsed = parseLiteralValue(config.type, value);
      if (parsed.error !== undefined) return;
      onChange(parsed.value);
      onClose();
    },
  });
  const submitOnEnter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !event.nativeEvent.isComposing) {
      event.preventDefault();
      event.stopPropagation();
      void form.handleSubmit();
    }
  };
  return (
    <div>
      <div className={range ? "grid grid-cols-2 gap-2" : ""}>
        {(range ? (["start", "end"] as const) : (["start"] as const)).map((name) => (
          <form.Field key={name} name={name}>
            {(field) => (
              <label className="block text-12 text-secondary">
                {range ? (name === "start" ? "From" : "To") : "Value"}
                <input
                  onKeyDown={submitOnEnter}
                  data-autofocus={name === "start" ? true : undefined}
                  aria-describedby={`${id}-error`}
                  value={field.state.value}
                  inputMode={numeric ? "decimal" : "text"}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  className="mt-1 w-full rounded border border-subtle bg-surface-1 px-2 py-1.5 text-13 text-primary focus:outline-accent-strong"
                />
              </label>
            )}
          </form.Field>
        ))}
      </div>
      <form.Subscribe selector={(state) => state.errors}>
        {(errors) => (
          <p id={`${id}-error`} aria-live="polite" className="mt-2 text-12 text-danger-primary">
            {errors.filter((error): error is string => typeof error === "string").join(" ")}
          </p>
        )}
      </form.Subscribe>
      <div className="mt-3 flex justify-end gap-2">
        <button type="button" onClick={onClose} className="rounded px-2 py-1 text-13 text-secondary hover:bg-layer-2">
          Cancel
        </button>
        <button
          type="button"
          onClick={() => void form.handleSubmit()}
          className="rounded bg-accent-primary px-3 py-1 text-13 text-on-color"
        >
          Apply
        </button>
      </div>
    </div>
  );
}
