import type { ReactNode } from "react";
import { Link, useParams } from "@tanstack/react-router";
import { ArrowDown, GripVertical } from "lucide-react";
import { Button } from "@plane/propel/button";
import type { WorkItemType } from "@/services/work-item-types.service";
import { TypeIcon } from "./controls";

type Props = {
  types: Pick<WorkItemType, "id" | "name" | "icon" | "color">[];
  levels: Record<string, number>;
  defaultLevel: number;
  readOnly: boolean;
  onMove: (id: string, level: number) => void;
  onDefault: (level: number) => void;
  children: ReactNode;
};
export function HierarchyLayout({ types, levels, defaultLevel, readOnly, onMove, onDefault, children }: Props) {
  const { workspaceSlug } = useParams({ strict: false });
  const top = Math.max(...Object.values(levels), 0);
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-16 font-medium">Work item hierarchy</h2>
          <p className="mt-1 max-w-xl text-13 text-tertiary">
            Place broader work above more detailed work. Each level can contain work from the level directly below it.
          </p>
        </div>
        {workspaceSlug && (
          <Link
            to="/$workspaceSlug/settings/work-item-types"
            params={{ workspaceSlug }}
            search={{ screen: undefined, state: undefined }}
            className="text-13 text-accent-primary hover:underline"
          >
            Manage work item types
          </Link>
        )}
      </div>
      <p className="text-12 text-tertiary">
        Create types in the work item type library. Empty levels are removed when you move their last type; Level 0
        stays available.
      </p>
      {top < 20 && (
        <div
          aria-label="New hierarchy level"
          onDragOver={(event) => {
            if (!readOnly) event.preventDefault();
          }}
          onDrop={(event) => {
            event.preventDefault();
            onMove(event.dataTransfer.getData("text/plain"), top + 1);
          }}
          className="rounded-lg border border-dashed border-strong p-5 text-center text-13 text-tertiary"
        >
          Drag a type here to create a new level, or choose New level from its menu.
        </div>
      )}
      <div className="space-y-3">
        {Array.from({ length: top + 1 }, (_, index) => top - index).map((level) => (
          <div
            key={level}
            onDragOver={(event) => {
              if (!readOnly) event.preventDefault();
            }}
            onDrop={(event) => {
              event.preventDefault();
              onMove(event.dataTransfer.getData("text/plain"), level);
            }}
          >
            <section aria-label={`Level ${level}`} className="rounded-lg border border-subtle bg-surface-1">
              <div className="flex items-center gap-3 border-b border-subtle bg-layer-1 px-5 py-3">
                <h3 className="text-13 font-medium">Level {level}</h3>
                {defaultLevel === level && (
                  <span className="rounded bg-layer-2 px-2 py-0.5 text-11 text-secondary">Default</span>
                )}
                <span className="ml-auto text-12 text-tertiary">
                  {level === 0 ? "Leaf work items" : `Parent of Level ${level - 1}`}
                </span>
                {defaultLevel !== level && (
                  <Button
                    variant="ghost"
                    disabled={readOnly}
                    onClick={() => {
                      onDefault(level);
                    }}
                  >
                    Set as default
                  </Button>
                )}
              </div>
              <div className="flex min-h-24 flex-wrap items-center gap-3 p-5">
                {types
                  .filter((type) => levels[type.id] === level)
                  .map((type) => (
                    <div
                      key={type.id}
                      draggable={!readOnly}
                      onDragStart={(event) => event.dataTransfer.setData("text/plain", type.id)}
                      className="shadow-sm flex items-center gap-2 rounded-md border border-strong bg-surface-1 px-3 py-2"
                    >
                      <GripVertical className="size-3.5 text-tertiary" />
                      <TypeIcon type={type} />
                      <span className="text-13">{type.name}</span>
                      <select
                        aria-label={`Move ${type.name} to level`}
                        className="ml-2 rounded border border-subtle bg-surface-1 p-1 text-11"
                        value={levels[type.id]}
                        disabled={readOnly}
                        onChange={(event) => onMove(type.id, Number(event.target.value))}
                      >
                        {Array.from({ length: top + 1 }, (_, i) => (
                          <option key={i} value={i}>
                            Level {i}
                          </option>
                        ))}
                        {top < 20 && <option value={top + 1}>New level</option>}
                      </select>
                    </div>
                  ))}
                {!types.some((type) => levels[type.id] === level) && (
                  <p className="text-13 text-tertiary">Drop a type here, or use its level menu.</p>
                )}
              </div>
            </section>
            {level > 0 && <ArrowDown className="mx-auto mt-3 size-5 text-tertiary" />}
          </div>
        ))}
      </div>
      {children}
    </div>
  );
}
