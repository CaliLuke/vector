import { CascadingValueInput } from "./cascading-value-input";
type PropertyInputProps = {
  property: PreviewProperty;
  value: Value | undefined;
  onChange: (value: Value) => void;
  disabled?: boolean;
  draft?: boolean;
};
import { lazy, Suspense, useState } from "react";
import { Button } from "@plane/propel/button";
import { CustomSearchSelect, ToggleSwitch } from "@plane/ui";
import { inputStyle, PreviewDialog } from "./controls";
import { previewPeople, previewReleases } from "./preview-content";
import type { PreviewItem, PreviewProperty } from "./preview-content";
const RichEditor = lazy(() =>
  import("@plane/editor").then((module) => ({
    default: module.RichTextEditorWithRef,
  }))
);
const localFileHandlers = {
  assetsUploadStatus: {},
  cancel: () => {},
  checkIfAssetExists: async () => false,
  delete: async () => {},
  restore: async () => {},
  getAssetDownloadSrc: async (path: string) => path,
  getAssetSrc: async (path: string) => path,
  upload: async () => {
    throw new Error("Uploads are unavailable in this sample preview.");
  },
  duplicate: async () => {
    throw new Error("Asset duplication is unavailable in this sample preview.");
  },
  validation: {
    maxFileSize: 0,
  },
};
type Value = PreviewItem["values"][string];
export function PropertyInput(props: PropertyInputProps) {
  const { property } = props;
  if (property.kind === "Boolean") return <BooleanInput {...props} />;
  if (property.kind === "Formula") return <FormulaInput {...props} />;
  if (property.kind === "Text" && property.format === "Read-only") return <FixedTextInput {...props} />;
  if (["Dropdown", "Member picker", "Release picker"].includes(property.kind)) return <SelectionInput {...props} />;
  if (property.kind === "Cascading") return <CascadingInput {...props} />;
  if (property.kind === "Rich text") return <RichTextInput {...props} />;
  return <PlainPropertyInput {...props} />;
}
function BooleanInput({ property, value, onChange, disabled }: PropertyInputProps) {
  return <ToggleSwitch label={property.name} value={value === true} disabled={disabled} onChange={onChange} />;
}
function FormulaInput({ property }: PropertyInputProps) {
  const [formulaState, setFormulaState] = useState("Value");
  return (
    <div className="flex items-center gap-2">
      <output
        aria-label={`${property.name} sample result`}
        className={`flex-1 text-13 ${formulaState === "Error" ? "text-danger-primary" : "text-secondary"}`}
      >
        {formulaState === "Value" ? "6" : formulaState === "Empty" ? "—" : "Cannot divide by zero"}
      </output>
      <select
        aria-label="Formula sample state"
        className="max-w-24 rounded border border-subtle bg-surface-1 px-1 py-0.5 text-11 text-tertiary"
        value={formulaState}
        onChange={(event) => setFormulaState(event.target.value)}
      >
        <option>Value</option>
        <option>Empty</option>
        <option>Error</option>
      </select>
    </div>
  );
}
function FixedTextInput({ property }: PropertyInputProps) {
  return <p className="text-13 text-secondary">{property.defaultValue}</p>;
}
function SelectionInput({ property, value, onChange, disabled }: PropertyInputProps) {
  const options =
    property.kind === "Member picker"
      ? previewPeople.map((name) => ({
          id: name,
          name,
        }))
      : property.kind === "Release picker"
        ? previewReleases.map((name) => ({
            id: name,
            name,
          }))
        : property.options;
  const selected = Array.isArray(value) ? value : value ? [String(value)] : [];
  const label =
    selected.map((id) => options.find((option) => option.id === id)?.name ?? id).join(", ") ||
    `Select ${property.name.toLowerCase()}`;
  const props = {
    disabled,
    input: true,
    label,
    className: "w-full",
    buttonClassName: "w-full text-left",
    options: options.map((option) => ({
      value: option.id,
      query: option.name,
      content: option.name,
    })),
    onChange,
  };
  return property.multiple ? (
    <CustomSearchSelect {...props} multiple value={selected} />
  ) : (
    <CustomSearchSelect {...props} value={selected[0] ?? ""} />
  );
}
function CascadingInput({ property, value, onChange, disabled, draft }: PropertyInputProps) {
  if (draft) return <p className="text-12 text-tertiary">Available when you publish this draft.</p>;
  return (
    <CascadingValueInput
      configuration={{
        levels: property.levels,
        options: property.options.map((option) => ({ ...option, next_label: option.nextLabel ?? "" })),
      }}
      value={value}
      onChange={onChange}
      disabled={disabled}
    />
  );
}

function RichTextInput({ property, value, onChange, disabled }: PropertyInputProps) {
  const [history, setHistory] = useState(false);
  return (
    <div className="space-y-2">
      <div className="rounded-md border border-subtle">
        <Suspense fallback={<div className="h-28 animate-pulse bg-layer-1" />}>
          <RichEditor
            id={`preview-${property.id}`}
            initialValue={String(value || "<p></p>")}
            value={String(value || "<p></p>")}
            editable={!disabled}
            onChange={(_, html) => onChange(html)}
            disabledExtensions={["image"]}
            flaggedExtensions={[]}
            fileHandler={localFileHandlers}
            mentionHandler={{
              renderComponent: () => null,
            }}
            getEditorMetaData={() => ({
              file_assets: [],
              user_mentions: [],
            })}
            extendedEditorProps={{}}
            containerClassName="min-h-28 p-3 text-13"
            placeholder="Write a description…"
          />
        </Suspense>
      </div>
      <Button variant="ghost" onClick={() => setHistory(true)}>
        Version history
      </Button>
      {history && (
        <PreviewDialog
          title={`${property.name} · Version history`}
          description="Sample history for comparing the field editor."
          onClose={() => setHistory(false)}
          footer={
            <Button variant="secondary" onClick={() => setHistory(false)}>
              Close
            </Button>
          }
        >
          <div className="space-y-4">
            <p className="text-12 text-tertiary">Alex Morgan · Today, 10:42 AM</p>
            <p className="text-13 whitespace-pre-wrap">
              Sign in on two devices. The second device should restore the existing session.
            </p>
            <Button
              disabled={disabled}
              onClick={() => {
                onChange("<p>Sign in on two devices. The second device should restore the existing session.</p>");
                setHistory(false);
              }}
            >
              Restore this sample version
            </Button>
          </div>
        </PreviewDialog>
      )}
    </div>
  );
}
function PlainPropertyInput({ property, value, onChange, disabled }: PropertyInputProps) {
  const text = typeof value === "string" ? value : "";
  if (property.kind === "Text" && property.format === "Paragraph")
    return (
      <textarea
        aria-label={property.name}
        className={inputStyle}
        rows={3}
        value={text}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  return (
    <input
      aria-label={property.name}
      className={inputStyle}
      type={
        property.kind === "Number"
          ? "number"
          : property.kind === "Date"
            ? "date"
            : property.kind === "URL"
              ? "url"
              : "text"
      }
      step={property.kind === "Number" ? "any" : undefined}
      value={text}
      disabled={disabled}
      onChange={(event) => onChange(event.target.value)}
      placeholder={property.kind === "URL" ? "https://" : "Empty"}
    />
  );
}
