import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { Confirmation, EmptyState } from "./controls";
import { useTypePreview } from "./preview-provider";
import { hierarchyConflicts, normalizeHierarchy } from "./preview-actions";
import { HierarchyLayout } from "./hierarchy-layout";

export function HierarchyPreview({ readOnly = false }: { readOnly?: boolean }) {
  const { content, setContent, announce } = useTypePreview();
  const savedHierarchy = normalizeHierarchy(
    Object.fromEntries(content.types.map((type) => [type.id, type.level])),
    content.hierarchyDefaultLevel
  );
  const form = useForm({
    defaultValues: { hierarchy: savedHierarchy },
    onSubmit: ({ value }) => save(value.hierarchy),
  });
  const { levels, defaultLevel } = useStore(form.store, (state) => state.values.hierarchy);
  const [confirm, setConfirm] = useState(false);
  const submitError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const conflicts = hierarchyConflicts(content, levels);
  const changed =
    defaultLevel !== content.hierarchyDefaultLevel || content.types.some((type) => type.level !== levels[type.id]);
  const moveType = (id: string, level: number) => {
    if (readOnly || !content.types.some((type) => type.id === id)) return;
    form.setFieldValue("hierarchy", normalizeHierarchy({ ...levels, [id]: level }, defaultLevel));
  };
  function save(hierarchy: typeof savedHierarchy) {
    if (readOnly) return;
    const submittedConflicts = hierarchyConflicts(content, hierarchy.levels);
    setContent((current) => ({
      ...current,
      hierarchyDefaultLevel: hierarchy.defaultLevel,
      types: current.types.map((type) => ({ ...type, level: hierarchy.levels[type.id] })),
      items: current.items.map((item) =>
        submittedConflicts.some((conflict) => conflict.id === item.id)
          ? { ...item, parent: "", activity: ["Parent removed after hierarchy changed.", ...item.activity] }
          : item
      ),
    }));
    form.reset({ hierarchy });
    announce("Hierarchy saved.");
    setConfirm(false);
  }
  if (!content.hierarchyEnabled)
    return (
      <EmptyState
        title="Give your work a structure"
        description="Choose which types can contain other types. Once enabled, hierarchy cannot be turned off."
      >
        <Button disabled={readOnly} onClick={() => setContent((current) => ({ ...current, hierarchyEnabled: true }))}>
          Enable hierarchy
        </Button>
      </EmptyState>
    );
  return (
    <HierarchyLayout
      types={content.types}
      levels={levels}
      defaultLevel={defaultLevel}
      readOnly={readOnly}
      onMove={moveType}
      onDefault={(level) => form.setFieldValue("hierarchy.defaultLevel", level)}
    >
      <div className="flex items-center justify-between gap-4 border-t border-subtle pt-5">
        <p className="text-12 text-tertiary">
          {changed ? "You have unsaved changes." : "Drag types between levels, or use the level selector."}
        </p>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            disabled={readOnly || !changed}
            onClick={() => {
              form.reset({ hierarchy: savedHierarchy });
            }}
          >
            Discard
          </Button>
          {conflicts.length ? (
            <Button disabled={readOnly || !changed} onClick={() => setConfirm(true)}>
              Save changes
            </Button>
          ) : (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                form.setErrorMap({ onSubmit: undefined });
                void form.handleSubmit().catch(() => undefined);
              }}
            >
              <Button type="submit" disabled={readOnly || !changed}>
                Save changes
              </Button>
            </form>
          )}
        </div>
      </div>
      {submitError && (
        <p role="alert" className="text-12 text-danger-primary">
          {typeof submitError === "string" ? submitError : "Could not save the hierarchy."}
        </p>
      )}
      {confirm && (
        <Confirmation
          title="Saving this will break existing links"
          action="Save anyway"
          token="confirm"
          onClose={() => setConfirm(false)}
          onConfirm={() => form.handleSubmit()}
        >
          <p>
            {conflicts.length} parent and sub-work-item links do not fit this hierarchy. Saving removes those parent
            links and records the change. The work items remain.
          </p>
          <ul className="space-y-2 rounded border border-subtle p-3">
            {conflicts.map((item) => (
              <li key={item.id}>
                {item.parent} → {item.id} · {item.name}
              </li>
            ))}
          </ul>
          <Button variant="secondary" onClick={() => setConfirm(false)}>
            Go back and fix
          </Button>
        </Confirmation>
      )}
    </HierarchyLayout>
  );
}
