/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm } from "@tanstack/react-form";
import { useCommittedScope } from "@/hooks/use-committed-scope";

export function useLabelCreationForm({
  onCreate,
  errorMessage,
}: {
  onCreate: (name: string, isCurrent: () => boolean) => Promise<void>;
  errorMessage: string;
}) {
  const isCurrent = useCommittedScope("label-creation");
  return useForm({
    defaultValues: { name: "" },
    validators: {
      onSubmit: ({ value }): string | undefined => {
        if (!value.name.trim()) return "Enter a label name.";
        if (value.name.trim().length > 255) return "Label names cannot exceed 255 characters.";
        return undefined;
      },
    },
    onSubmit: async ({ value, formApi }) => {
      try {
        await onCreate(value.name.trim(), isCurrent);
        if (isCurrent()) formApi.reset();
      } catch (error) {
        if (isCurrent()) formApi.setErrorMap({ onSubmit: errorMessage });
        throw error;
      }
    },
  });
}

export type TLabelCreationForm = ReturnType<typeof useLabelCreationForm>;

export function selectLabelCreationState(state: TLabelCreationForm["state"]) {
  return {
    query: state.values.name,
    submitting: state.isSubmitting,
    error: state.errorMap.onSubmit,
  };
}

export async function submitLabelCreationForm(form: TLabelCreationForm): Promise<void> {
  if (form.state.isSubmitting) return;
  form.setErrorMap({ onSubmit: undefined });
  await form.handleSubmit().catch(() => undefined);
}
