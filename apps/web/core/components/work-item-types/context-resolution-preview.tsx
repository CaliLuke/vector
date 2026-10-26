import { useId, useState } from "react";
import { propertyDefaultLabel } from "./property-default-input";
import { inputStyle } from "./controls";
import { resolvePropertyContext } from "./context-resolution";
import type { PreviewProperty, PreviewContext } from "./preview-content";

type Choice = { id: string; name: string };
export function ContextResolutionPreview({
  property,
  projects,
  types,
  members,
}: {
  property: PreviewProperty;
  projects: Choice[];
  types: Choice[];
  members?: Choice[];
}) {
  const id = useId();
  const [project, setProject] = useState(projects[0]?.id ?? "");
  const [type, setType] = useState(types[0]?.id ?? "");
  return (
    <section className="space-y-3 rounded-md border border-subtle p-4" aria-label="Effective property settings">
      <h3 className="text-13 font-medium">Check effective settings</h3>
      <div className="grid gap-3 sm:grid-cols-2">
        <label htmlFor={`${id}-project`} className="text-12">
          Project
          <select
            id={`${id}-project`}
            className={`${inputStyle} mt-1`}
            value={project}
            onChange={(event) => setProject(event.target.value)}
          >
            <option value="">Without a project</option>
            {projects.map((choice) => (
              <option key={choice.id} value={choice.id}>
                {choice.name}
              </option>
            ))}
          </select>
        </label>
        <label htmlFor={`${id}-type`} className="text-12">
          Work item type
          <select
            id={`${id}-type`}
            className={`${inputStyle} mt-1`}
            value={type}
            onChange={(event) => setType(event.target.value)}
          >
            <option value="">Without a type</option>
            {types.map((choice) => (
              <option key={choice.id} value={choice.id}>
                {choice.name}
              </option>
            ))}
          </select>
        </label>
      </div>
      <ResolvedSettings property={property} project={project} type={type} members={members} />
    </section>
  );
}
function ResolvedSettings({
  property,
  project,
  type,
  members,
}: {
  property: PreviewProperty;
  project: string;
  type: string;
  members?: Choice[];
}) {
  let resolved;
  try {
    resolved = resolvePropertyContext(property, project, type);
  } catch (cause) {
    return (
      <p role="alert" className="text-12 text-danger-primary">
        {cause instanceof Error ? cause.message : "Resolve conflicting contexts."}
      </p>
    );
  }
  const { context, effective } = resolved;
  const contextName = context?.name ?? "Default";
  const rows = effectiveRows(property, effective, context, members);
  return (
    <div className="space-y-3 text-12">
      <p>
        Winning context: <strong>{contextName}</strong>
      </p>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
        {rows.map((row) => (
          <div key={row.name} className="contents">
            <dt>{row.name}</dt>
            <dd>
              {row.value}
              <span className="ml-2 text-tertiary">{row.origin}</span>
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

function effectiveRows(
  property: PreviewProperty,
  effective: PreviewProperty,
  context: PreviewContext | undefined,
  members?: Choice[]
) {
  const contextName = context?.name ?? "Default";
  const origin = (override: unknown) => (override == null ? "Base setting" : contextName);
  const rows = [
    { name: "Mandatory", value: effective.required ? "Yes" : "No", origin: origin(context?.required) },
    { name: "Active", value: effective.active ? "Visible" : "Hidden", origin: origin(context?.active) },
  ];
  if (["Number", "Dropdown", "Member picker"].includes(property.kind))
    rows.push({
      name: "Default value",
      value: propertyDefaultLabel(effective, members),
      origin: effective.required ? "Cleared by mandatory setting" : origin(context?.defaultValue),
    });
  if (property.kind === "Date")
    rows.push({ name: "Date format", value: effective.format, origin: origin(context?.format) });
  if (property.kind === "Cascading")
    rows.push({
      name: "Option catalog",
      value: `${effective.options.length} options`,
      origin: origin(context?.options),
    });
  return rows;
}
