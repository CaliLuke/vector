/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Form } from "@plane/ui";
import { FormErrors } from "@plane/ui";
import { useEffect, useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { TwitterPicker } from "react-color";
import { Button } from "@plane/propel/button";
import type { IState } from "@plane/types";
import { Popover, Input, TextArea } from "@plane/ui";

type Props = {
  data: Partial<IState>;
  onSubmit: (draft: Partial<IState>) => Promise<void>;
  onCancel: () => void;
  buttonTitle: string;
};
export function StateForm({ data, onSubmit, onCancel, buttonTitle }: Props) {
  const nameInput = useRef<HTMLInputElement>(null);
  useEffect(() => {
    nameInput.current?.focus();
  }, []);
  const form = useForm({
    defaultValues: { name: data.name ?? "", description: data.description ?? "", color: data.color ?? "#000000" },
    onSubmit: async ({ value }) => onSubmit({ ...value, name: value.name.trim() }),
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  return (
    <Form
      onSubmit={() => {
        form.setErrorMap({ onSubmit: undefined });
        return form.handleSubmit().catch((error: unknown) => {
          form.setErrorMap({
            onSubmit: {
              form: error instanceof Error ? error.message : "Could not save this state. Please try again.",
              fields: {},
            },
          });
        });
      }}
      noValidate
    >
      <fieldset disabled={busy} inert={busy} className="relative flex space-x-2 rounded-sm bg-surface-1 p-3">
        <form.Field name="color">
          {(field) => (
            <div className="mt-2 h-full shrink-0">
              <Popover
                button={
                  <span
                    aria-label="State color"
                    className="inline-flex size-5 rounded-sm"
                    style={{ backgroundColor: field.state.value }}
                  />
                }
                panelClassName="mt-4 -ml-3"
              >
                <TwitterPicker color={field.state.value} onChange={(value) => field.handleChange(value.hex)} />
              </Popover>
            </div>
          )}
        </form.Field>
        <div className="w-full space-y-2">
          <form.Field
            name="name"
            validators={{
              onChange: ({ value }) =>
                !value.trim()
                  ? "Name is required"
                  : value.length > 100
                    ? "Name must be at most 100 characters."
                    : undefined,
            }}
          >
            {(field) => (
              <>
                <Input
                  ref={nameInput}
                  type="text"
                  name={field.name}
                  aria-label="State name"
                  placeholder="Name"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  hasError={!field.state.meta.isValid}
                  className="w-full"
                  maxLength={100}
                />
                <FormErrors errors={field.state.meta.errors} className="text-13 text-danger-primary" />
              </>
            )}
          </form.Field>
          <form.Field name="description">
            {(field) => (
              <TextArea
                name={field.name}
                aria-label="State description"
                placeholder="Describe this state for your members."
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                className="min-h-14 w-full resize-none text-13"
              />
            )}
          </form.Field>
          {formError && (
            <p role="alert" className="text-13 text-danger-primary">
              {typeof formError === "string" ? formError : "Could not save this state. Please try again."}
            </p>
          )}
          <div className="flex items-center space-x-2">
            <Button type="submit" variant="primary" size="lg" disabled={busy} loading={busy}>
              {buttonTitle}
            </Button>
            <Button type="button" variant="secondary" size="lg" disabled={busy} onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </div>
      </fieldset>
    </Form>
  );
}
