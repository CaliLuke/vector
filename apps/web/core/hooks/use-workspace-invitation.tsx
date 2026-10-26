/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import { EUserPermissions } from "@plane/constants";
import { checkEmailValidity } from "@plane/utils/string";

export type InvitationFormValues = { emails: { email: string; role: EUserPermissions }[] };
export const emptyInvitation = () => ({ id: crypto.randomUUID(), email: "", role: EUserPermissions.MEMBER });

export function useInvitationForm({
  onSubmit,
  initialRows = 1,
  allowEmptyRows = false,
}: {
  onSubmit: (data: InvitationFormValues) => Promise<void> | undefined;
  initialRows?: number;
  allowEmptyRows?: boolean;
}) {
  const [defaultValues] = useState(() => ({ emails: Array.from({ length: initialRows }, emptyInvitation) }));
  return useForm({
    defaultValues,
    validators: {
      onChange: ({ value }) => {
        const emails = allowEmptyRows ? value.emails.filter((entry) => entry.email.trim()) : value.emails;
        return emails.length > 0 && emails.every((entry) => checkEmailValidity(entry.email))
          ? undefined
          : "Enter valid email addresses.";
      },
    },
    onSubmit: async ({ value, formApi }) => {
      const submission = onSubmit({
        emails: value.emails.filter((entry) => entry.email.trim()).map(({ email, role }) => ({ email, role })),
      });
      if (!submission) return;
      await submission;
      formApi.reset();
    },
  });
}

export type InvitationFormApi = ReturnType<typeof useInvitationForm>;
