/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFormHook, createFormHookContexts, useStore } from "@tanstack/react-form";
import { useState } from "react";
import type { ReactNode } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Button } from "@plane/propel/button";
import { Input } from "@plane/ui";

const { fieldContext, formContext, useFieldContext, useFormContext } = createFormHookContexts();

export type SettingsInput = {
  type: "text" | "password" | "email";
  label: string;
  description?: ReactNode;
  placeholder: string;
  required: boolean;
};

export function requiredField(label: string) {
  return ({ value }: { value: string | undefined }) => (value?.trim() ? undefined : `${label} is required.`);
}

function TextField({ type, label, description, placeholder, required }: SettingsInput) {
  const field = useFieldContext<string | undefined>();
  const isSubmitting = useStore(field.form.store, (state) => state.isSubmitting);
  const [showPassword, setShowPassword] = useState(false);
  const errors = field.state.meta.errors.filter((error): error is string => typeof error === "string");
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={field.name} className="text-13 text-tertiary">
        {label}
      </label>
      <div className="relative">
        <Input
          id={field.name}
          name={field.name}
          type={type === "password" && showPassword ? "text" : type}
          value={field.state.value ?? ""}
          onBlur={field.handleBlur}
          onChange={(event) => field.handleChange(event.target.value)}
          required={required}
          disabled={isSubmitting}
          hasError={errors.length > 0}
          aria-invalid={errors.length > 0}
          aria-describedby={`${field.name}-feedback`}
          placeholder={placeholder}
          className={`w-full rounded-md font-medium ${type === "password" ? "pr-10" : ""}`}
        />
        {type === "password" && (
          <button
            type="button"
            aria-label={showPassword ? "Hide password" : "Show password"}
            className="absolute top-2.5 right-3 flex items-center justify-center text-placeholder"
            onClick={() => setShowPassword(!showPassword)}
          >
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        )}
      </div>
      <div id={`${field.name}-feedback`}>
        {description && <p className="pt-0.5 text-11 text-tertiary">{description}</p>}
        {errors.length > 0 && (
          <p role="alert" className="text-13 text-danger-primary">
            {errors.join(" ")}
          </p>
        )}
      </div>
    </div>
  );
}

function SaveButton({ allowUnchanged = false }: { allowUnchanged?: boolean }) {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting, state.isDefaultValue]}>
      {([canSubmit, isSubmitting, isDefaultValue]) => (
        <Button
          type="submit"
          variant="primary"
          size="lg"
          loading={isSubmitting}
          disabled={!canSubmit || (isDefaultValue && !allowUnchanged)}
        >
          {isSubmitting ? "Saving" : "Save changes"}
        </Button>
      )}
    </form.Subscribe>
  );
}

export const { useAppForm: useSettingsForm } = createFormHook({
  fieldContext,
  formContext,
  fieldComponents: { TextField },
  formComponents: { SaveButton },
});

export function SettingsForm({
  onSubmit,
  children,
  className = "space-y-8",
}: {
  onSubmit: () => Promise<void>;
  children: ReactNode;
  className?: string;
}) {
  return (
    <form
      className={className}
      onSubmit={(event) => {
        event.preventDefault();
        void onSubmit().catch(() => {});
      }}
    >
      {children}
    </form>
  );
}
