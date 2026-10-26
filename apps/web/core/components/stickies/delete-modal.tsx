/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
// ui
import { useTranslation } from "@plane/i18n";
import { AlertModalCore } from "@plane/ui";

interface IStickyDelete {
  isOpen: boolean;
  handleSubmit: () => Promise<void>;
  handleClose: () => void;
}

export function StickyDeleteModal(props: IStickyDelete) {
  return props.isOpen ? <StickyDeleteConfirmation {...props} /> : null;
}

function StickyDeleteConfirmation({ isOpen, handleClose, handleSubmit }: IStickyDelete) {
  const { t } = useTranslation();
  const form = useForm({
    defaultValues: {},
    onSubmit: async ({ formApi }) => {
      try {
        await handleSubmit();
      } catch (error) {
        formApi.setErrorMap({ onSubmit: { form: t("stickies.toasts.not_removed.message"), fields: {} } });
        throw error;
      }
    },
  });
  const pending = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) =>
    typeof state.errorMap.onSubmit === "string" ? state.errorMap.onSubmit : undefined
  );
  return (
    <AlertModalCore
      handleClose={() => {
        if (!form.state.isSubmitting) handleClose();
      }}
      handleSubmit={() => {
        if (form.state.isSubmitting) return;
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {});
      }}
      isSubmitting={pending}
      isOpen={isOpen}
      title={t("stickies.delete")}
      content={
        <>
          {t("stickies.delete_confirmation")}
          {error && (
            <span role="alert" className="block text-danger-primary">
              {error}
            </span>
          )}
        </>
      }
    />
  );
}
