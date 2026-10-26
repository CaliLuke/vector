/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useEffect, useRef } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import type { FileRejection } from "react-dropzone";
import { useDropzone } from "react-dropzone";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { useFileSize } from "@/hooks/use-file-size";
import type { TAttachmentHelpers } from "../issue-detail-widgets/attachments/helper";

export function useAttachmentDropzone({
  workspaceSlug,
  create,
  disabled,
  onUploaded,
}: {
  workspaceSlug: string;
  create: TAttachmentHelpers["operations"]["create"];
  disabled?: boolean;
  onUploaded?: () => void;
}) {
  const { t } = useTranslation();
  const { maxFileSize } = useFileSize();
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const form = useForm({
    defaultValues: { file: null as File | null },
    validators: {
      onSubmit: ({ value }) => {
        if (!value.file) return t("attachment.error");
        if (value.file.size > maxFileSize) return t("attachment.file_size_limit", { size: maxFileSize / 1024 / 1024 });
        return undefined;
      },
    },
    onSubmit: async ({ value, formApi }) => {
      if (disabled || !workspaceSlug || !value.file) return;
      try {
        await create(value.file);
      } catch (cause) {
        const message = t("attachment.error");
        formApi.setErrorMap({ onSubmit: message });
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message });
        throw cause;
      }
      formApi.reset();
      if (mounted.current) onUploaded?.();
    },
  });
  const isUploading = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const hasFile = useStore(form.store, (state) => state.values.file !== null);
  const submit = useCallback(
    () =>
      form.handleSubmit().catch(() => {
        // Form retains the failed file and error; the upload operation reports it to the user.
      }),
    [form]
  );
  const onDrop = useCallback(
    (accepted: File[], rejected: FileRejection[]) => {
      if (disabled || form.state.isSubmitting || !workspaceSlug) return;
      if (rejected.length) {
        form.setFieldValue("file", null);
        const message =
          accepted.length + rejected.length > 1
            ? t("attachment.only_one_file_allowed")
            : t("attachment.file_size_limit", { size: maxFileSize / 1024 / 1024 });
        form.setErrorMap({ onSubmit: message });
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message,
        });
        return;
      }
      const file = accepted[0];
      if (!file || !workspaceSlug) return;
      form.setFieldValue("file", file);
      void submit();
    },
    [disabled, form, maxFileSize, submit, t, workspaceSlug]
  );
  const dropzone = useDropzone({ onDrop, maxSize: maxFileSize, multiple: false, disabled: isUploading || disabled });
  return { ...dropzone, isUploading, error, canRetry: hasFile && !!error, retry: submit };
}
