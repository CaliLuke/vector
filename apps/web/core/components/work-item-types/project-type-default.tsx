import { CustomSearchSelect } from "@plane/ui";
import { TypeIcon } from "./controls";
import type { ProjectWorkItemType } from "@/services/work-item-types.service";

export function ProjectTypeDefault({
  types,
  value,
  disabled,
  onChange,
}: {
  types: ProjectWorkItemType[];
  value: string;
  disabled: boolean;
  onChange: (id: string) => void;
}) {
  const selected = types.find((type) => type.id === value);
  return (
    <div className="mb-6 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-subtle p-4">
      <div>
        <h3 className="text-14 font-medium">Default work item type</h3>
        <p className="mt-1 text-12 text-tertiary">Suggested for new work when it fits the hierarchy.</p>
      </div>
      <CustomSearchSelect
        value={value}
        disabled={disabled}
        onChange={onChange}
        customButtonAsChild
        customButton={
          <button
            type="button"
            aria-label="Default work item type"
            className="flex items-center gap-2 rounded-md border border-subtle px-3 py-2 text-13"
            disabled={disabled}
          >
            {selected && <TypeIcon type={selected} />}
            {selected?.name ?? "Automatic"}
            {selected && !selected.available && <span className="text-tertiary">Unavailable</span>}
          </button>
        }
        options={[
          { value: "", query: "Automatic", content: "Automatic" },
          ...types
            .filter((type) => type.available || type.id === value)
            .map((type) => ({
              value: type.id,
              query: type.name,
              disabled: !type.available,
              content: (
                <span className="flex items-center gap-2">
                  <TypeIcon type={type} />
                  {type.name}
                </span>
              ),
            })),
        ]}
      />
    </div>
  );
}
