/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FormErrors } from "@plane/ui";
import { useForm, useStore } from "@tanstack/react-form";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { Calendar } from "@plane/propel/calendar";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";

export type InboxIssueSnoozeModalProps = {
  isOpen: boolean;
  value: Date | undefined;
  onConfirm: (value: Date) => Promise<void>;
  handleClose: () => void;
};

export function InboxIssueSnoozeModal(props: InboxIssueSnoozeModalProps) {
  return props.isOpen ? <SnoozeForm {...props} /> : null;
}

function SnoozeForm({ handleClose, value: initialDate, onConfirm }: InboxIssueSnoozeModalProps) {
  const { t } = useTranslation();
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const form = useForm({
    defaultValues: { date: initialDate ? new Date(initialDate) : today },
    onSubmit: async ({ value: draft }) => {
      form.setErrorMap({ onSubmit: undefined });
      try {
        await onConfirm(draft.date);
        handleClose();
      } catch (cause) {
        form.setErrorMap({ onSubmit: { form: "Could not snooze this work item. Please try again.", fields: {} } });
        throw cause;
      }
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  return (
    <ModalCore
      isOpen
      handleClose={() => {
        if (!busy) handleClose();
      }}
      position={EModalPosition.CENTER}
      width={EModalWidth.SM}
      className="w-auto"
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <fieldset disabled={busy} inert={busy} className="flex h-full w-full flex-col gap-y-1 px-5 py-8 sm:p-6">
          <form.Field
            name="date"
            validators={{ onSubmit: ({ value }) => (value >= today ? undefined : "Choose today or a future date.") }}
          >
            {(field) => (
              <>
                <Calendar
                  className="rounded-md border border-subtle p-3"
                  captionLayout="dropdown"
                  selected={field.state.value}
                  defaultMonth={field.state.value}
                  onSelect={(date: Date | undefined) => {
                    if (date) field.handleChange(date);
                  }}
                  mode="single"
                  disabled={[{ before: today }]}
                />
                <FormErrors errors={field.state.meta.errors} className="text-13 text-danger-primary" />
              </>
            )}
          </form.Field>
          {error && (
            <p role="alert" className="text-13 text-danger-primary">
              {error}
            </p>
          )}
          <Button type="submit" variant="primary" disabled={busy} loading={busy}>
            {t("inbox_issue.actions.snooze")}
          </Button>
        </fieldset>
      </form>
    </ModalCore>
  );
}
