/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TNameDescriptionLoader } from "@plane/types";
import { TextArea } from "@plane/ui";
import { cn } from "@plane/utils/common";
import type { TIssueOperations } from "./issue-detail";
import { useAutosaveFormSubmit } from "@/hooks/use-autosave-form-submit";
import { useItemTextEditor, useItemTextSession } from "./use-item-edit-coordination";

export type IssueTitleInputProps = {
  disabled?: boolean;
  value: string | undefined | null;
  workspaceSlug: string;
  setIsSubmitting: (value: TNameDescriptionLoader) => void;
  issueOperations: TIssueOperations;
  projectId: string;
  issueId: string;
  className?: string;
  containerClassName?: string;
};

type TitleSnapshot = { name: string; revision: number };

export function IssueTitleInput(props: IssueTitleInputProps) {
  const session = useItemTextSession();
  return <TitleEditor key={`${session.key}/${props.workspaceSlug}/${props.projectId}/${props.issueId}`} {...props} />;
}

function TitleEditor(props: IssueTitleInputProps) {
  const {
    disabled,
    value,
    workspaceSlug,
    setIsSubmitting: notifyStatus,
    issueId,
    issueOperations,
    projectId,
    className,
    containerClassName,
  } = props;
  const {
    report: setIsSubmitting,
    locked,
    canEdit,
    current,
  } = useItemTextEditor(`${workspaceSlug}/${projectId}/${issueId}`, notifyStatus);
  const { t } = useTranslation();
  const [isLengthVisible, setIsLengthVisible] = useState(false);
  const [defaults, setDefaults] = useState(() => ({ name: value ?? "" }));
  const revision = useRef(0);
  const queuedRevision = useRef(0);
  const persisted = useRef<TitleSnapshot>({ name: defaults.name, revision: 0 });
  const failedRevision = useRef<number | undefined>(undefined);
  const previousValue = useRef(value);
  const mounted = useRef(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const form = useForm({
    defaultValues: defaults,
    validators: {
      onSubmit: ({ value: draft }) => (draft.name.trim() ? undefined : t("form.title.required")),
    },
    onSubmit: async ({ value: draft, formApi }) => {
      clearTimeout(timer.current);
      timer.current = undefined;
      const snapshot = { name: draft.name.trim(), revision: revision.current };
      queuedRevision.current = snapshot.revision;
      failedRevision.current = undefined;
      try {
        await save.mutateAsync(snapshot);
      } catch (cause) {
        if (revision.current === snapshot.revision) {
          formApi.setErrorMap({ onSubmit: "Could not save the title." });
        }
        throw cause;
      }
    },
  });
  const submissionError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const { submit } = useAutosaveFormSubmit(form);
  const save = useMutation({
    scope: { id: `issue-text/${workspaceSlug}/${projectId}/${issueId}` },
    retry: false,
    mutationFn: (snapshot: TitleSnapshot) => {
      if (!current()) throw new Error("The account session changed.");
      return issueOperations.updateText(workspaceSlug, projectId, issueId, { name: snapshot.name });
    },
    onSuccess: (_result, snapshot) => {
      persisted.current = snapshot;
      if (failedRevision.current === snapshot.revision) failedRevision.current = undefined;
      if (!mounted.current || revision.current !== snapshot.revision) return;
      setDefaults({ name: snapshot.name });
      form.reset({ name: snapshot.name });
      setIsSubmitting("submitted");
    },
    onError: (_error, snapshot) => {
      failedRevision.current = snapshot.revision;
      if (mounted.current && revision.current === snapshot.revision) setIsSubmitting("error");
      if (!mounted.current && revision.current === snapshot.revision)
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Title not saved",
          message: "The last title edit could not be saved. Please reopen the work item and try again.",
        });
    },
  });
  const scheduleSave = (snapshot: TitleSnapshot) => {
    timer.current = setTimeout(() => {
      queuedRevision.current = snapshot.revision;
      failedRevision.current = undefined;
      submit();
    }, 1500);
  };

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      clearTimeout(timer.current);
      if (!current()) return;
      const name = form.state.values.name.trim();
      if (name && (revision.current > queuedRevision.current || failedRevision.current === revision.current)) {
        queuedRevision.current = revision.current;
        failedRevision.current = undefined;
        submit();
      }
    };
  }, [form, submit, current]);

  useEffect(() => {
    if (previousValue.current === value) return;
    previousValue.current = value;
    // Incoming optimistic updates and refetches must not replace an unsaved draft.
    if (revision.current === persisted.current.revision) {
      persisted.current = { name: value ?? "", revision: revision.current };
      setDefaults({ name: value ?? "" });
      form.reset({ name: value ?? "" });
    }
  }, [value, form]);

  const changeTitle = (name: string) => {
    if (disabled || !canEdit()) return;
    form.setErrorMap({ onSubmit: undefined });
    form.setFieldValue("name", name);
    revision.current += 1;
    clearTimeout(timer.current);
    if (name === persisted.current.name && queuedRevision.current === persisted.current.revision) {
      persisted.current = { name, revision: revision.current };
      queuedRevision.current = revision.current;
      setIsSubmitting("saved");
      return;
    }
    setIsSubmitting("submitting");
    if (name.trim()) scheduleSave({ name: name.trim(), revision: revision.current });
  };

  return (
    <form.Field
      name="name"
      validators={{ onChange: ({ value: name }) => (name.trim() ? undefined : t("form.title.required")) }}
    >
      {(field) =>
        disabled || locked ? (
          <div className="text-20 font-medium whitespace-pre-line">{field.state.value}</div>
        ) : (
          <div className="flex flex-col gap-1.5">
            <div className={cn("relative", containerClassName)}>
              <TextArea
                aria-label={t("issue.title.label")}
                className={cn(
                  "block w-full resize-none overflow-hidden rounded-sm border-none bg-transparent px-3 py-0 text-20 font-medium ring-0 outline-none",
                  { "mx-2.5 ring-1 ring-danger-strong": !field.state.value.trim() },
                  className
                )}
                value={field.state.value}
                onChange={(event) => changeTitle(event.target.value)}
                maxLength={255}
                placeholder={t("issue.title.label")}
                onFocus={() => setIsLengthVisible(true)}
                onBlur={() => {
                  setIsLengthVisible(false);
                  field.handleBlur();
                  const normalized = field.state.value.trim() || persisted.current.name;
                  if (normalized === field.state.value) return;
                  // Nonblank autosaves already contain the trimmed value. Blur must not queue that write twice.
                  if (field.state.value.trim()) form.setFieldValue("name", normalized);
                  else changeTitle(normalized);
                }}
              />
              <div
                className={cn(
                  "pointer-events-none absolute right-1 bottom-1 z-[2] rounded-sm bg-surface-1 p-0.5 text-11 text-secondary opacity-0 transition-opacity",
                  { "opacity-100": isLengthVisible }
                )}
              >
                <span
                  className={
                    field.state.value.length === 0 || field.state.value.length > 255 ? "text-danger-primary" : ""
                  }
                >
                  {field.state.value.length}
                </span>
                /255
              </div>
            </div>
            {field.state.meta.errors.length > 0 && (
              <span className="text-13 font-medium text-danger-primary">{field.state.meta.errors.join(", ")}</span>
            )}
            {submissionError && failedRevision.current === revision.current && (
              <div role="alert" className="flex items-center gap-2 text-13 text-danger-primary">
                {submissionError}
                <button
                  type="button"
                  onClick={() => {
                    clearTimeout(timer.current);
                    queuedRevision.current = revision.current;
                    failedRevision.current = undefined;
                    setIsSubmitting("submitting");
                    submit();
                  }}
                >
                  Retry
                </button>
              </div>
            )}
          </div>
        )
      }
    </form.Field>
  );
}
