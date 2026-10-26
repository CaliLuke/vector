/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useForm, useStore } from "@tanstack/react-form";
import { PROJECT_ERROR_MESSAGES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { AlertModalCore } from "@plane/ui";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { isAuthorizationError } from "@/queries/authorization-errors";
import type { InboxIssueConfirmationProps } from "./confirmation-types";

type Props = InboxIssueConfirmationProps & { action: "delete" | "decline" };

export function InboxIssueConfirmation(props: Props) {
  if (!props.isOpen) return null;
  return <ConfirmationDialog key={`${props.action}/${props.data.id}`} {...props} />;
}

function ConfirmationDialog({ action, data, projectIdentifier, onSubmit, onClose, onSuccess, isCurrent }: Props) {
  const mounted = useCommittedScope(`${action}/${data.id}`);
  const { t } = useTranslation();
  const ownsDialog = () => mounted() && isCurrent();
  const form = useForm({
    defaultValues: {},
    onSubmit: async ({ formApi }) => {
      if (!ownsDialog()) throw new Error("This confirmation is no longer active.");
      try {
        await onSubmit();
      } catch (cause) {
        if (ownsDialog()) {
          const error =
            action === "delete"
              ? isAuthorizationError(cause)
                ? PROJECT_ERROR_MESSAGES.permissionError
                : PROJECT_ERROR_MESSAGES.issueDeleteError
              : undefined;
          const message = error?.i18n_message ? t(error.i18n_message) : "Please try again.";
          formApi.setErrorMap({ onSubmit: { form: message, fields: {} } });
          setToast({
            type: TOAST_TYPE.ERROR,
            title: error ? t(error.i18n_title) : "Could not decline work item",
            message,
          });
        }
        throw cause;
      }
      if (!ownsDialog()) return;
      if (action === "delete") {
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("inbox_issue.modals.delete.success") });
      }
      onSuccess?.();
      onClose();
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) =>
    typeof state.errorMap.onSubmit === "string" ? state.errorMap.onSubmit : undefined
  );
  return (
    <AlertModalCore
      isOpen
      isSubmitting={busy}
      handleClose={() => {
        if (!form.state.isSubmitting && ownsDialog()) onClose();
      }}
      handleSubmit={() => {
        if (form.state.isSubmitting || !ownsDialog()) return;
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {});
      }}
      title={t(`inbox_issue.modals.${action}.title`)}
      content={
        <>
          Are you sure you want to {action} work item{" "}
          <span className="font-medium break-words text-primary">
            {projectIdentifier}-{data.sequence_id}
          </span>
          {action === "delete"
            ? "? The work item will only be deleted from the intake and this action cannot be undone."
            : "? This action cannot be undone."}
          {error && (
            <span role="alert" className="mt-2 block text-danger-primary">
              {error}
            </span>
          )}
        </>
      }
      primaryButtonText={action === "decline" ? { loading: t("declining"), default: t("decline") } : undefined}
    />
  );
}
