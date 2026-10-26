import type { IIssueDisplayProperties, IIssueDisplayFilterOptions } from "@plane/types";
import { useWorkItemTableCatalog } from "@/hooks/use-work-item-table-catalog";
import { OptionIcon } from "@/components/work-item-types/property-option-label";

export function CustomPropertyColumns({
  displayProperties,
  displayFilters,
  isEpic,
  onChange,
}: {
  displayProperties: IIssueDisplayProperties;
  displayFilters?: IIssueDisplayFilterOptions;
  isEpic: boolean;
  onChange: (value: Partial<IIssueDisplayProperties>) => void;
}) {
  const enabled = displayFilters?.layout === "spreadsheet" && !isEpic;
  const catalog = useWorkItemTableCatalog(enabled);
  if (!enabled || !catalog.supported) return null;
  if (catalog.isError)
    return (
      <p role="alert" className="text-12 text-danger-primary">
        Could not load custom columns.{" "}
        <button type="button" onClick={() => void catalog.retry()}>
          Retry
        </button>
      </p>
    );
  if (!catalog.identities) return <output className="text-12 text-tertiary">Loading custom columns…</output>;
  if (!catalog.fields.length) return null;
  return (
    <fieldset className="my-3">
      <legend className="mb-2 text-12 text-secondary">Custom properties</legend>
      <div className="flex flex-wrap gap-2">
        {catalog.fields.map((field) => (
          <button
            key={field.key}
            type="button"
            aria-pressed={!!displayProperties[field.key]}
            className={`rounded-sm border px-2 py-0.5 text-11 ${displayProperties[field.key] ? "border-accent-strong bg-accent-primary text-on-color" : "border-subtle hover:bg-layer-1"}`}
            onClick={() => onChange({ [field.key]: !displayProperties[field.key] })}
          >
            <OptionIcon icon={field.icon} />
            {field.name}
          </button>
        ))}
      </div>
    </fieldset>
  );
}
