import type { IssueDraft } from "./form-state";

export function issueSaveError(cause: unknown, fallback: string): string {
  const body = errorBody(cause, false);
  if (!isRecord(body)) return fallback;
  for (const key of ["error", "message", "detail"]) {
    const value = body[key];
    if (typeof value === "string" && value.trim()) return value;
  }
  return fallback;
}

const fields = [
  "project_id",
  "name",
  "description_html",
  "type_id",
  "state_id",
  "priority",
  "assignee_ids",
  "label_ids",
  "cycle_id",
  "module_ids",
  "start_date",
  "target_date",
  "estimate_point",
  "parent_id",
] as const;
export function issueFieldErrors(cause: unknown): Partial<Record<keyof IssueDraft, string>> {
  const error = errorBody(cause);
  const body = isRecord(error) && isRecord(error.fields) ? error.fields : error;
  if (!isRecord(body)) return {};
  const result: Partial<Record<keyof IssueDraft, string>> = {};
  for (const field of fields) {
    if (!Object.hasOwn(body, field)) continue;
    const value = body[field];
    const messages = Array.isArray(value) ? value.filter((message) => typeof message === "string") : [value];
    if (messages.every((message) => typeof message === "string")) result[field] = messages.join(" ");
  }
  return result;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object";
}
function errorBody(cause: unknown, includeCause = true): unknown {
  if (!isRecord(cause)) return undefined;
  const response = isRecord(cause.response) ? cause.response : undefined;
  return response?.data ?? cause.data ?? (includeCause ? cause : undefined);
}

export function issueSourceChanged(cause: unknown): boolean {
  const error = errorBody(cause);
  const body = isRecord(error) && isRecord(error.fields) ? error.fields : error;
  return isRecord(body) && body.source_changed === true;
}

export function issueConfigurationChanged(cause: unknown): boolean {
  const error = errorBody(cause);
  const body = isRecord(error) && isRecord(error.fields) ? error.fields : error;
  return isRecord(body) && body.review_again === true;
}
