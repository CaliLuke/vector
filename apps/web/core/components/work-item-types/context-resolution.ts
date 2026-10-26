import type { PreviewContext, PreviewProperty } from "./preview-content";

const specificity = (context: PreviewContext) => Number(context.projects.length > 0) + Number(context.types.length > 0);

export function resolvePropertyContext(property: PreviewProperty, project: string, type: string) {
  const matching = property.contexts.filter(
    (context) =>
      (!context.projects.length || context.projects.some((id) => id.toLowerCase() === project.toLowerCase())) &&
      (!context.types.length || context.types.some((id) => id.toLowerCase() === type.toLowerCase()))
  );
  const ranks = matching.map(specificity);
  if (new Set(ranks).size !== ranks.length)
    throw new Error("Matching contexts have conflicting scopes. Narrow their projects or types.");
  const context = matching.reduce<PreviewContext | undefined>(
    (best, current) => (!best || specificity(current) > specificity(best) ? current : best),
    undefined
  );
  const required = context?.required ?? property.required;
  const effective: PreviewProperty = {
    ...property,
    required,
    active: context?.active ?? property.active,
    defaultValue: required ? "" : (context?.defaultValue ?? property.defaultValue),
    format: context?.format ?? property.format,
    options: context?.options ?? property.options,
  };
  return { context, effective };
}
