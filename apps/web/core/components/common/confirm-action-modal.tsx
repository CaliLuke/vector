/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { AlertTriangle } from "lucide-react";
import { Button } from "@plane/propel/button";
import { Input, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";

type Props = {
  isOpen: boolean;
  title: string;
  description: ReactNode;
  name: string;
  confirmation: string;
  actionLabel: string;
  pendingLabel: string;
  errorMessage: string;
  onConfirm: () => Promise<void>;
  onConfirmed?: () => void;
  onClose: () => void;
};

export function ConfirmActionModal(props: Props) {
  return props.isOpen ? <ConfirmActionDialog {...props} /> : null;
}

function ConfirmActionDialog({
  title,
  description,
  name,
  confirmation,
  actionLabel,
  pendingLabel,
  errorMessage,
  onConfirm,
  onConfirmed,
  onClose,
}: Props) {
  const form = useForm({
    defaultValues: { name: "", confirmation: "" },
    validators: {
      onSubmit: ({ value }) =>
        value.name === name && value.confirmation === confirmation ? undefined : "Confirmation does not match.",
    },
    onSubmit: async () => {
      try {
        await onConfirm();
      } catch (cause) {
        form.setErrorMap({ onSubmit: { form: errorMessage, fields: {} } });
        throw cause;
      }
      onClose();
      onConfirmed?.();
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const canConfirm = useStore(
    form.store,
    (state) => state.values.name === name && state.values.confirmation === confirmation
  );
  const close = () => {
    if (!isSubmitting) onClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
        className="flex flex-col gap-6 p-6"
      >
        <div className="flex items-center gap-6">
          <span className="rounded-full bg-danger-subtle p-4">
            <AlertTriangle className="size-6 text-danger-primary" aria-hidden="true" />
          </span>
          <h3 className="text-18 font-medium">{title}</h3>
        </div>
        <div className="text-13 leading-7 text-secondary">{description}</div>
        {(["name", "confirmation"] as const).map((key) => (
          <form.Field key={key} name={key}>
            {(field) => (
              <label className="text-13 text-secondary">
                {key === "name" ? "Enter the name " : "To confirm, type "}
                <span className="font-medium text-primary">{key === "name" ? name : confirmation}</span> to continue:
                <Input
                  name={field.name}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  disabled={isSubmitting}
                  autoComplete="off"
                  className="mt-2 w-full"
                />
              </label>
            )}
          </form.Field>
        ))}
        {error && (
          <p role="alert" className="text-13 text-danger-primary">
            {typeof error === "string" ? error : errorMessage}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" size="lg" onClick={close} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="error-fill"
            size="lg"
            disabled={!canConfirm || isSubmitting}
            loading={isSubmitting}
          >
            {isSubmitting ? pendingLabel : actionLabel}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
