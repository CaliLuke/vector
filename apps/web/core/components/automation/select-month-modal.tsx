/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IProject } from "@plane/types";
import { Input, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";

type Props = {
  isOpen: boolean;
  type: "auto-close" | "auto-archive";
  initialValues: Partial<IProject>;
  handleClose: () => void;
  handleChange: (formData: Partial<IProject>) => Promise<void>;
};

export function SelectMonthModal(props: Props) {
  if (!props.isOpen) return null;
  return <MonthForm key={props.type} {...props} />;
}

function MonthForm({ type, initialValues, handleClose, handleChange }: Props) {
  const fieldName = type === "auto-close" ? "close_in" : "archive_in";
  const form = useForm({
    defaultValues: { months: String(initialValues[fieldName] ?? "") },
    onSubmit: async ({ value }) => {
      try {
        await handleChange({ [fieldName]: Number(value.months) });
        handleClose();
      } catch (cause) {
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not save time range", message: "Please try again." });
        throw cause;
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const onClose = () => {
    if (!isSubmitting) handleClose();
  };
  return (
    <ModalCore isOpen handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <h3 className="text-16 leading-6 font-medium text-primary">Customize time range</h3>
        <form.Field
          name="months"
          validators={{
            onChange: ({ value }) => {
              const months = Number(value);
              return Number.isInteger(months) && months >= 1 && months <= 12
                ? undefined
                : "Select a month between 1 and 12.";
            },
          }}
        >
          {(field) => (
            <div className="relative mt-8 flex w-full flex-col justify-center gap-1">
              <Input
                id={fieldName}
                name={fieldName}
                type="number"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                hasError={!field.state.meta.isValid}
                placeholder="Enter Months"
                aria-label="Months"
                className="w-full border-subtle"
                min={1}
                max={12}
                disabled={isSubmitting}
              />
              <span className="absolute top-2.5 right-8 text-13 text-secondary">Months</span>
              {!field.state.meta.isValid && (
                <span role="alert" className="px-1 text-13 text-danger-primary">
                  {field.state.meta.errors.join(" ")}
                </span>
              )}
            </div>
          )}
        </form.Field>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" type="button" size="lg" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
            {isSubmitting ? "Submitting..." : "Submit"}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
