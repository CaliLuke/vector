/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useForm } from "@tanstack/react-form";
import { DEFAULT_WORK_ITEM_FORM_VALUES } from "@plane/constants";
import { getChangedIssuefields } from "@plane/utils/work-item";
import type { TIssue } from "@plane/types";

type EditableFields =
  | "project_id"
  | "name"
  | "description_html"
  | "state_id"
  | "priority"
  | "assignee_ids"
  | "label_ids"
  | "cycle_id"
  | "module_ids"
  | "start_date"
  | "target_date"
  | "estimate_point"
  | "parent_id"
  | "type_id";
export type IssueDraft = Pick<TIssue, EditableFields> & Partial<TIssue>;

export function issueTitleValidators(requiredMessage: string, tooLongMessage: string, required = true) {
  const validate = ({ value }: { value: string }) =>
    required && !value.trim() ? requiredMessage : value.length > 255 ? tooLongMessage : undefined;

  return { onChange: validate, onSubmit: validate };
}

export function issueDraftDefaults(projectId: string | null, data: Partial<TIssue>): IssueDraft {
  const { project_id = projectId } = data;
  return {
    ...DEFAULT_WORK_ITEM_FORM_VALUES,
    ...data,
    project_id,
    name: data.name ?? "",
    description_html: data.description_html ?? "<p></p>",
    ...issuePropertyDefaults(data),
  };
}
function issuePropertyDefaults(data: Partial<TIssue>) {
  return {
    state_id: data.state_id ?? "",
    priority: data.priority ?? "none",
    assignee_ids: data.assignee_ids ?? [],
    label_ids: data.label_ids ?? [],
    cycle_id: data.cycle_id ?? null,
    module_ids: data.module_ids ?? [],
    start_date: data.start_date ?? null,
    target_date: data.target_date ?? null,
    estimate_point: data.estimate_point ?? null,
    parent_id: data.parent_id ?? null,
    type_id: data.type_id ?? null,
    custom_properties: data.custom_properties ?? {},
  };
}

export function useIssueDraftForm(
  defaultValues: IssueDraft,
  onSubmit: (
    value: IssueDraft,
    action: "save" | "move",
    formApi: Pick<IssueDraftForm, "reset" | "setErrorMap">,
    fieldMeta: object
  ) => Promise<void>,
  validateOnSubmit?: (value: IssueDraft) => string | undefined
) {
  const onSubmitMeta: { action: "save" | "move" } = { action: "save" };
  return useForm({
    defaultValues,
    onSubmitMeta,
    validators: validateOnSubmit
      ? {
          onSubmit: ({ value }) => validateOnSubmit(value),
        }
      : undefined,
    onSubmit: ({ value, meta, formApi }) => onSubmit(value, meta.action, formApi, formApi.state.fieldMeta),
  });
}
export type IssueDraftForm = ReturnType<typeof useIssueDraftForm>;

export function issueSubmissionPayload(
  value: IssueDraft,
  dirtyFields: Record<string, boolean | undefined>,
  id: string | undefined,
  isDraft: boolean,
  customProperties: TIssue["custom_properties"]
): Partial<TIssue> {
  if (!id) return { ...value, custom_properties: customProperties };
  return {
    ...getChangedIssuefields(value, dirtyFields),
    ...(!isDraft ? { project_id: value.project_id, description_html: value.description_html } : {}),
    id,
    custom_properties: customProperties,
    ...(value.is_draft === undefined ? {} : { is_draft: value.is_draft }),
  };
}
