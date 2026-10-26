import type { PreviewContext, PreviewOption, PreviewProperty } from "./preview-content";

function sameScope(left: string[], right: string[]) {
  return left.length === right.length && left.every((id) => right.includes(id));
}
function sameCatalogScopes(before: PreviewContext[], after: PreviewContext[]) {
  return (
    before.length === after.length &&
    before.every((context) => {
      const next = after.find((item) => item.id === context.id);
      return (
        next &&
        sameScope(context.projects, next.projects) &&
        sameScope(context.types, next.types) &&
        (context.options !== undefined) === (next.options !== undefined)
      );
    })
  );
}

export function cascadeChanges(before: PreviewProperty, after: PreviewProperty) {
  if (before.kind !== "Cascading") return [];
  const scopesChanged =
    !sameCatalogScopes(before.contexts, after.contexts) &&
    [...before.contexts, ...after.contexts].some((context) => context.options !== undefined);
  const catalogs = [
    { id: "default", name: "Default", options: before.options, next: after.options },
    ...before.contexts
      .filter((context) => context.options !== undefined)
      .map((context) => ({
        id: context.id,
        name: context.name,
        options: context.options ?? [],
        next: after.contexts.find((next) => next.id === context.id)?.options ?? [],
      })),
  ];
  return catalogs
    .map((catalog) => ({
      id: catalog.id,
      name: catalog.name,
      scopeChanged: scopesChanged,
      paths: catalog.options
        .filter((option) => scopesChanged || pathChanged(option, catalog.options, catalog.next, before, after))
        .map((option) => ({ id: option.id, name: optionPath(option, catalog.options) })),
    }))
    .filter((catalog) => catalog.paths.length > 0);
}

function pathChanged(
  option: PreviewOption,
  previous: PreviewOption[],
  next: PreviewOption[],
  before: PreviewProperty,
  after: PreviewProperty
): boolean {
  const replacement = next.find((item) => item.id === option.id);
  if (!replacement || replacement.parent !== option.parent || replacement.level !== option.level) return true;
  if (before.levels[option.level]?.id !== after.levels[option.level]?.id) return true;
  const parent = previous.find((item) => item.id === option.parent);
  return parent ? pathChanged(parent, previous, next, before, after) : false;
}

function optionPath(option: PreviewOption, options: PreviewOption[]): string {
  const parent = options.find((item) => item.id === option.parent);
  return parent ? `${optionPath(parent, options)} → ${option.name}` : option.name;
}

export function CascadeChangeSummary({ before, after }: { before: PreviewProperty; after: PreviewProperty }) {
  const catalogs = cascadeChanges(before, after);
  if (!catalogs.length) return null;
  return (
    <section
      aria-label="Changed cascading choices"
      className="max-h-52 space-y-3 overflow-y-auto rounded-md border border-subtle p-3"
    >
      <h3 className="font-medium text-primary">Previous choices in changed configurations</h3>
      <p>Only affected work items' selected paths are preserved, using their former labels.</p>
      {catalogs.map((catalog) => (
        <div key={catalog.id}>
          <p className="font-medium">
            {catalog.name}
            {catalog.scopeChanged ? " · potential paths affected by scope changes" : " · removed or changed paths"}
          </p>
          <ul className="list-inside list-disc">
            {catalog.paths.map((path) => (
              <li key={path.id}>{path.name}</li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}
