/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState, useRef, useEffect } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { PlusIcon, CloseIcon } from "@plane/propel/icons";
import { Input, InputColorPicker } from "@plane/ui";
import type { IIssueLabel } from "@plane/types";
import type { TLabelOperations } from "./root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  values: string[];
  labelOperations: TLabelOperations;
  disabled?: boolean;
};

export function LabelCreate(props: Props) {
  const [open, setOpen] = useState(false);
  return open ? (
    <CreateLabelForm {...props} onClose={() => setOpen(false)} />
  ) : (
    <button
      type="button"
      disabled={props.disabled}
      onClick={() => setOpen(true)}
      className="flex items-center gap-1 rounded-full border border-subtle px-2 py-0.5 text-11 text-tertiary hover:bg-surface-2"
    >
      <PlusIcon className="size-2.5" /> New
    </button>
  );
}

function CreateLabelForm({
  workspaceSlug,
  projectId,
  issueId,
  values,
  labelOperations,
  disabled,
  onClose,
}: Props & { onClose: () => void }) {
  const input = useRef<HTMLInputElement>(null);
  const create = useMutation({
    mutationFn: (draft: Pick<IIssueLabel, "name" | "color">) => labelOperations.createLabel(draft),
  });
  const form = useForm({
    defaultValues: { name: "", color: "#ff0000" },
    validators: {
      onSubmit: ({ value }) => (value.name.trim() ? undefined : "Enter a label name."),
    },
    onSubmit: async ({ value }) => {
      form.setErrorMap({ onSubmit: undefined });
      try {
        const label = create.data ?? (await create.mutateAsync(value));
        await labelOperations.updateIssue(workspaceSlug, projectId, issueId, {
          label_ids: [...new Set([...values, label.id])],
        });
        onClose();
      } catch (cause) {
        form.setErrorMap({ onSubmit: { form: "Could not add the label. Please try again.", fields: {} } });
        throw cause;
      }
    },
  });
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const color = useStore(form.store, (state) => state.values.color);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  useEffect(() => {
    input.current?.focus();
  }, []);
  const locked = disabled || busy || create.isSuccess;
  return (
    <form
      className="flex flex-wrap items-center gap-2 p-1"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        event.stopPropagation();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {});
      }}
    >
      <InputColorPicker
        name="label-color"
        placeholder="Color"
        hasError={false}
        value={color}
        onChange={(value) => form.setFieldValue("color", value)}
        disabled={locked}
      />
      <form.Field
        name="name"
        validators={{ onChange: ({ value }) => (value.trim() ? undefined : "Enter a label name.") }}
      >
        {(field) => (
          <div>
            <Input
              ref={input}
              name={field.name}
              value={field.state.value}
              onChange={(event) => field.handleChange(event.target.value)}
              onBlur={field.handleBlur}
              disabled={locked}
              hasError={!field.state.meta.isValid}
              placeholder="Title"
              className="px-1.5 py-1 text-11"
            />
            {field.state.meta.errors.map((message) => (
              <p key={message} role="alert" className="text-11 text-danger-primary">
                {message}
              </p>
            ))}
          </div>
        )}
      </form.Field>
      <button
        type="button"
        aria-label="Cancel new label"
        disabled={busy}
        onClick={onClose}
        className="rounded-sm bg-danger-primary p-1"
      >
        <CloseIcon className="size-3.5 text-on-color" />
      </button>
      <button
        type="submit"
        aria-label={busy ? "Adding label" : "Add label"}
        disabled={busy || disabled}
        className="rounded-sm bg-success-primary p-1"
      >
        <PlusIcon className="size-3.5 text-on-color" />
      </button>
      {error && (
        <p role="alert" className="w-full text-11 text-danger-primary">
          {error}
        </p>
      )}
    </form>
  );
}
