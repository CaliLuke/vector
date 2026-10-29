import { PropertyIcon } from "./property-icon";
import { CascadingValueInput } from "./cascading-value-input";
import { useId } from "react";
import type { ReactNode } from "react";
import { OptionIcon, dropdownPath } from "./property-option-label";
import { CustomSearchSelect } from "@plane/ui/dropdowns";
import { ToggleSwitch } from "@plane/ui/button";
import type { ItemSchema, ResolvedProperty } from "@/services/work-item-schema.service";

const control =
  "w-full rounded-md border border-subtle bg-surface-1 px-3 py-2 text-13 text-primary outline-none focus:border-accent-strong";
type FormulaResult = { status: "value" | "empty" | "error"; value: unknown; error: string };
type Props = {
  property: ResolvedProperty;
  value: unknown;
  onChange: (value: unknown) => void;
  schema: ItemSchema;
  error?: string;
  onControlMount?: (element: HTMLElement | null) => void;
  disabled?: boolean;
  formulaResult?: FormulaResult;
  renderRichText: (property: ResolvedProperty, attributes: Record<string, string>) => ReactNode;
};

export function ItemPropertyInput(props: Props) {
  const { property, error } = props;
  const id = `property-${useId()}`;
  const required = property.configuration.required && property.kind !== "Boolean";
  return (
    <fieldset className="min-w-0 space-y-1.5">
      <legend id={`${id}-label`} className="flex items-center gap-2 text-13 font-medium text-secondary">
        <PropertyIcon kind={property.kind} icon={property.icon} />
        {property.title}
        {required && <span aria-label="required"> *</span>}
      </legend>
      {property.description && (
        <p id={`${id}-description`} className="text-12 text-tertiary">
          {property.description}
        </p>
      )}
      <PropertyControl {...props} controlId={id} />
      {error && (
        <p id={`${id}-error`} role="alert" className="text-12 text-danger-primary">
          {error}
        </p>
      )}
    </fieldset>
  );
}

function PropertyControl(props: Props & { controlId: string }) {
  const {
    property,
    value,
    onChange,
    disabled,
    formulaResult,
    renderRichText,
    error,
    controlId: id,
    onControlMount,
  } = props;
  const config = property.configuration;
  const accessible = controlProps(props);
  if (property.kind === "Formula") {
    return (
      <output
        className={formulaResult?.status === "error" ? "text-13 text-danger-primary" : "text-13 text-secondary"}
        aria-live="polite"
      >
        {!formulaResult
          ? "Calculating…"
          : formulaResult.status === "error"
            ? formulaResult.error
            : formulaResult.status === "empty"
              ? "—"
              : String(formulaResult.value)}
      </output>
    );
  }
  if (property.kind === "Text" && config.format === "Read-only")
    return <p className="text-13 whitespace-pre-wrap text-secondary">{config.content}</p>;
  if (property.kind === "Boolean")
    return (
      <ToggleSwitch
        {...accessible}
        label={property.title}
        value={value === true}
        disabled={disabled}
        onChange={onChange}
      />
    );
  if (property.kind === "Rich text")
    return renderRichText(property, {
      "aria-labelledby": accessible["aria-labelledby"],
      "aria-describedby": accessible["aria-describedby"] ?? "",
      "aria-invalid": String(accessible["aria-invalid"]),
    });
  if (property.kind === "Cascading")
    return (
      <CascadingValueInput
        configuration={config}
        value={value}
        onChange={onChange}
        disabled={disabled}
        error={error}
        controlId={id}
        onControlMount={onControlMount}
      />
    );
  if (["Dropdown", "Member picker", "Release picker"].includes(property.kind)) return <SelectionControl {...props} />;
  return <PlainControl {...props} />;
}

function SelectionControl(props: Props & { controlId: string }) {
  const { property, value, onChange, disabled, schema } = props;
  const config = property.configuration;
  const accessible = controlProps(props);
  const choices =
    property.kind === "Member picker"
      ? schema.members.map((member) => ({
          id: member.id,
          name: member.name,
          reason: member.reason,
          disabled: !member.eligible,
        }))
      : property.kind === "Release picker"
        ? schema.releases.map((option) => ({
            id: option.id,
            name: option.name,
            reason: option.reason,
            disabled: !option.eligible,
          }))
        : config.options.map((option) => ({
            id: option.id,
            icon: option.icon,
            name: dropdownPath(option.id, config.options),
            disabled: false,
            reason: "",
          }));
  const selected = Array.isArray(value)
    ? value.filter((id): id is string => typeof id === "string")
    : typeof value === "string" && value
      ? [value]
      : [];
  const label =
    selected.map((id) => choices.find((choice) => choice.id === id)?.name ?? "Unavailable selection").join(", ") ||
    `Select ${property.title.toLowerCase()}`;
  const options = choices.map((choice) => ({
    value: choice.id,
    query: choice.name,
    disabled: choice.disabled,
    content: (
      <span title={choice.reason || undefined}>
        {"icon" in choice && typeof choice.icon === "string" && <OptionIcon icon={choice.icon} />}
        {choice.name}
        {choice.reason && <span className="block text-11 text-tertiary">{choice.reason}</span>}
      </span>
    ),
  }));
  const shared = {
    disabled,
    input: true,
    label,
    className: "min-w-0 flex-1",
    options,
    onChange,
    customButtonAsChild: true,
    customButton: (
      <button type="button" {...accessible} className={`${control} text-left`}>
        {label}
      </button>
    ),
  };
  return (
    <div className="flex min-w-0 items-center gap-2">
      {config.multiple ? (
        <CustomSearchSelect {...shared} multiple value={selected} />
      ) : (
        <CustomSearchSelect {...shared} value={selected[0] ?? ""} />
      )}
      {selected.length > 0 && (
        <button
          type="button"
          disabled={disabled}
          className="shrink-0 text-12 text-secondary hover:text-primary"
          aria-label={`Clear ${property.title}`}
          onClick={() => onChange(config.multiple ? [] : null)}
        >
          Clear
        </button>
      )}
    </div>
  );
}

function PlainControl(props: Props & { controlId: string }) {
  const { property, value, onChange } = props;
  const config = property.configuration;
  const accessible = controlProps(props);
  const text = typeof value === "string" || typeof value === "number" ? String(value) : "";
  if (property.kind === "Text" && config.format === "Paragraph")
    return (
      <textarea
        {...accessible}
        className={`${control} min-h-20`}
        value={text}
        onChange={(event) => onChange(event.target.value || null)}
      />
    );
  return (
    <input
      {...accessible}
      className={control}
      type={property.kind === "Date" ? "date" : property.kind === "URL" ? "url" : "text"}
      inputMode={property.kind === "Number" ? "decimal" : undefined}
      value={text}
      onChange={(event) => {
        const next = event.target.value;
        if (next === "") {
          onChange(null);
          return;
        }
        onChange(next);
      }}
    />
  );
}

function controlProps({ property, disabled, error, controlId: id, onControlMount }: Props & { controlId: string }) {
  return {
    id,
    ref: onControlMount,
    "aria-labelledby": `${id}-label`,
    "aria-describedby":
      [property.description && `${id}-description`, error && `${id}-error`].filter(Boolean).join(" ") || undefined,
    "aria-invalid": Boolean(error),
    disabled,
  };
}
