import type { ReleaseValues } from "@/services/project-releases.service";
const fields = ["name", "description", "target_date"] as const;
export const emptyReleaseValues: ReleaseValues = { name: "", description: "", target_date: "" };
/** Explicit reload keeps local edits and adopts newer untouched fields. */
export function reloadReleaseDraft(
  draft: ReleaseValues,
  previous: ReleaseValues,
  current: ReleaseValues
): ReleaseValues {
  const result = { ...draft };
  for (const field of fields) {
    if (draft[field] === previous[field]) result[field] = current[field];
  }
  return result;
}

export function releaseValues(value: ReleaseValues): ReleaseValues {
  return { name: value.name, description: value.description, target_date: value.target_date };
}
