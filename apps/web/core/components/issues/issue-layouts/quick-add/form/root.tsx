/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useOutsideClickDetector } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
import type { TIssue } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
import { createIssuePayload } from "@plane/utils/work-item";
import { cn } from "@plane/utils/common";
import { useProjectDetail } from "@/hooks/use-projects";
import { IssueCreationIncompleteError, useIssueCreation } from "../../../issue-modal/use-issue-creation";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useParams } from "@tanstack/react-router";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useQueryClient } from "@tanstack/react-query";
import { currentParentCatalogueScope, captureParentAccess } from "@/queries/cycle-module-scope";
import useKeypress from "@/hooks/use-keypress";
import { notifyIssueCreated } from "../../../create-issue-toast-action-items";

type Props = {
  workspaceSlug: string;
  projectId: string;
  layout: EIssueLayoutTypes;
  prePopulatedData?: Partial<TIssue>;
  onClose: () => void;
  isEpic: boolean;
  onCreate?: (id: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
};
const styles: Record<EIssueLayoutTypes, { container: string; row: string; input: string; hint?: string }> = {
  list: {
    container: "shadow-raised-200",
    row: "flex items-center gap-3 border border-t-0 border-subtle bg-surface-1 px-3",
    input: "px-2 py-3",
    hint: "px-3 py-2 text-secondary",
  },
  gantt_chart: {
    container: "shadow-raised-200",
    row: "flex items-center gap-3 border border-subtle bg-surface-1 px-3",
    input: "px-2 py-3",
    hint: "bg-surface-1 px-3 py-2 text-secondary",
  },
  kanban: {
    container: "m-1 overflow-hidden rounded-sm bg-layer-2 shadow-raised-200",
    row: "flex flex-col items-start p-3",
    input: "py-1.5",
    hint: "bg-layer-3 px-3 py-2 text-tertiary",
  },
  spreadsheet: {
    container: "pb-2",
    row: "flex items-center gap-5 border border-t-0 border-subtle bg-surface-1 px-4 shadow-raised-200",
    input: "py-3",
    hint: "mt-3 ml-3 text-secondary",
  },
  calendar: {
    container: "z-20 w-full",
    row: "flex items-center gap-2 rounded-sm border-subtle bg-surface-1 px-2 md:border md:shadow-raised-100",
    input: "py-1.5 pr-2 md:text-11",
  },
};
export function QuickAddIssueForm({
  workspaceSlug,
  projectId,
  layout,
  prePopulatedData,
  onClose,
  onCreate,
  isEpic,
}: Props) {
  const { t } = useTranslation();
  const { project } = useProjectDetail(workspaceSlug, projectId);
  const storeType = useIssueListType();
  const actions = useIssuesActions(storeType);
  const client = useQueryClient();
  const { cycleId, moduleId } = useParams({ strict: false });
  const current = useCommittedScope(
    JSON.stringify([workspaceSlug, projectId, storeType, cycleId, moduleId, isEpic, layout])
  );
  const creation = useIssueCreation<TIssue>(
    {
      create: async ({ payload }) => {
        const issue = await onCreate?.(projectId, payload);
        if (!issue) throw new Error("Creation did not complete.");
        return issue;
      },
      assignCycle: async (issue, { payload }) => {
        if (isEpic || !payload.cycle_id || payload.cycle_id === "None") return;
        await actions.addCycleToIssue(projectId, payload.cycle_id, issue.id);
      },
      assignModules: async (issue, { payload }) => {
        const ids = payload.module_ids?.filter((id) => id !== "None") ?? [];
        if (isEpic || ids.length === 0) return;
        await actions.changeModulesInIssue(projectId, issue.id, ids, []);
      },
    },
    JSON.stringify([workspaceSlug, projectId, storeType, cycleId, moduleId, isEpic, layout])
  );
  const container = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const form = useForm({
    defaultValues: { name: "" },
    validators: {
      onSubmit: () => (onCreate ? undefined : "Issue creation is unavailable."),
    },
    onSubmit: async ({ value, formApi }) => {
      if (!onCreate) throw new Error("Issue creation is unavailable.");
      formApi.setErrorMap({ onSubmit: undefined });
      try {
        const payload = quickAddPayload(projectId, prePopulatedData, value, storeType, cycleId, moduleId);
        const access = captureParentAccess(
          client,
          currentParentCatalogueScope(client, workspaceSlug, projectId).project
        );
        const issue = await creation.submit({ payload, isDraft: false });
        if (!current() || !access.isCurrent()) return;
        creation.reset();
        formApi.reset();
        input.current?.focus();
        notifyIssueCreated({
          workspaceSlug,
          projectId,
          issueId: issue.id,
          isEpic,
          title: t("common.success"),
          message: t(isEpic ? "epic.create.success" : "issue.create.success"),
        });
      } catch (cause) {
        if (current())
          formApi.setErrorMap({
            onSubmit: {
              form: cause instanceof IssueCreationIncompleteError ? cause.message : t("common.error.message"),
              fields: {},
            },
          });
        throw cause;
      }
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const close = () => {
    if (!busy) onClose();
  };
  useOutsideClickDetector(container, close);
  useKeypress("Escape", close);
  useEffect(() => {
    if (project?.id) input.current?.focus();
  }, [project?.id]);
  if (!project) return null;
  const style = styles[layout];
  return (
    <div ref={container} className={style.container}>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <form.Field
          name="name"
          validators={{
            onChange: ({ value }) =>
              !value.trim()
                ? t(isEpic ? "epic.title.required" : "issue.title.required")
                : value.length > 255
                  ? "Title must be at most 255 characters."
                  : undefined,
          }}
        >
          {(field) => (
            <>
              <div className={style.row}>
                <span className="shrink-0 text-11 font-medium text-placeholder">{project.identifier}</span>
                <input
                  ref={input}
                  name={field.name}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  disabled={busy || creation.hasCreated || !onCreate}
                  autoComplete="off"
                  maxLength={255}
                  aria-label={t(isEpic ? "epic.title.label" : "issue.title.label")}
                  placeholder={t(isEpic ? "epic.title.label" : "issue.title.label")}
                  aria-invalid={!field.state.meta.isValid}
                  className={cn(
                    "w-full rounded-md bg-transparent text-13 leading-5 font-medium text-secondary outline-none",
                    style.input
                  )}
                />
              </div>
              {field.state.meta.errors.map((message) => (
                <p key={message} role="alert" className="px-3 text-11 text-danger-primary">
                  {message}
                </p>
              ))}
            </>
          )}
        </form.Field>
      </form>
      {creation.hasCreated && error && (
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            void form.handleSubmit().catch(() => {});
          }}
        >
          Retry finishing work item
        </button>
      )}
      {error && (
        <p role="alert" className="px-3 py-2 text-11 text-danger-primary">
          {error}
        </p>
      )}
      {style.hint && (
        <p className={cn("text-11 italic", style.hint)}>
          {t(isEpic ? "epic.add.press_enter" : "issue.add.press_enter")}
        </p>
      )}
    </div>
  );
}

function quickAddPayload(
  projectId: string,
  prePopulatedData: Partial<TIssue> | undefined,
  value: Pick<TIssue, "name">,
  storeType: EIssuesStoreType,
  cycleId: string | undefined,
  moduleId: string | undefined
) {
  const moduleIds = moduleId
    ? (prePopulatedData?.module_ids?.length ?? 0) > 1
      ? prePopulatedData?.module_ids
      : [moduleId]
    : undefined;
  return createIssuePayload(projectId, {
    ...prePopulatedData,
    ...value,
    ...(storeType === EIssuesStoreType.CYCLE && cycleId ? { cycle_id: cycleId } : {}),
    ...(storeType === EIssuesStoreType.MODULE && moduleIds ? { module_ids: moduleIds } : {}),
  });
}
