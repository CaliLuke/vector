/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { useRef } from "react";
import { XCircle } from "lucide-react";
import { Button } from "@plane/propel/button";
import { checkEmailValidity } from "@plane/utils/string";
import { Input } from "../form-fields/input";
import { cn } from "../utils";

type Labels = { email: string; placeholder: string; clear: string; invalid: string; continue: string; failure: string };
const defaultLabels: Labels = {
  email: "Email",
  placeholder: "name@company.com",
  clear: "Clear email",
  invalid: "Email is invalid",
  continue: "Continue",
  failure: "Could not continue. Please try again.",
};
type Props = {
  defaultEmail: string;
  onSubmit: (data: { email: string }) => Promise<void>;
  labels?: Labels;
  className?: string;
};

export function EmailEntryForm({ defaultEmail, onSubmit, labels = defaultLabels, className }: Props) {
  const input = useRef<HTMLInputElement>(null);
  const form = useForm({
    defaultValues: { email: defaultEmail },
    validators: {
      onSubmit: ({ value }) => (checkEmailValidity(value.email) ? undefined : labels.invalid),
    },
    onSubmit: async ({ value }) => onSubmit(value),
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const canSubmit = useStore(form.store, (state) => checkEmailValidity(state.values.email));
  const submitError = useStore(form.store, (state) => state.errorMap.onSubmit);
  return (
    <form
      className={cn("space-y-4", className)}
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => form.setErrorMap({ onSubmit: { form: labels.failure, fields: {} } }));
      }}
      noValidate
    >
      <form.Field
        name="email"
        validators={{ onChange: ({ value }) => (checkEmailValidity(value) ? undefined : labels.invalid) }}
      >
        {(field) => (
          <div className="space-y-1">
            <label htmlFor="email" className="text-13 font-medium text-tertiary">
              {labels.email}
            </label>
            <div className="relative flex items-center rounded-md border border-strong bg-surface-1">
              <Input
                ref={input}
                id="email"
                name={field.name}
                type="email"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                disabled={busy}
                placeholder={labels.placeholder}
                autoComplete="email"
                className="h-10 w-full border-0 pr-10 disable-autofill-style placeholder:text-placeholder"
                aria-invalid={!field.state.meta.isValid}
              />
              {field.state.value && (
                <button
                  type="button"
                  disabled={busy}
                  aria-label={labels.clear}
                  className="absolute right-3 grid size-5 place-items-center"
                  onClick={() => {
                    field.handleChange("");
                    input.current?.focus();
                  }}
                >
                  <XCircle className="size-5 stroke-placeholder" />
                </button>
              )}
            </div>
            {field.state.meta.isBlurred &&
              field.state.meta.errors.map((error) => (
                <p key={error} role="alert" className="text-11 text-danger-primary">
                  {error}
                </p>
              ))}
          </div>
        )}
      </form.Field>
      {submitError && (
        <p role="alert" className="text-11 text-danger-primary">
          {labels.failure}
        </p>
      )}
      <Button type="submit" variant="primary" className="w-full" size="xl" disabled={!canSubmit || busy} loading={busy}>
        {labels.continue}
      </Button>
    </form>
  );
}
