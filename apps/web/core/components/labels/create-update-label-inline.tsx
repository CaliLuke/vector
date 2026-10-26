/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { forwardRef, useEffect, useRef, useState } from "react";
import type { Dispatch, SetStateAction, ForwardedRef } from "react";
import { TwitterPicker } from "react-color";
import { useForm, useStore } from "@tanstack/react-form";
import { Popover } from "@headlessui/react";
import { getRandomLabelColor, LABEL_COLOR_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueLabel } from "@plane/types";
import { Input } from "@plane/ui";

export type TLabelOperationsCallbacks = {
  createLabel: (data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
  updateLabel: (labelId: string, data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
};
type Props = {
  labelForm: boolean;
  setLabelForm: Dispatch<SetStateAction<boolean>>;
  isUpdating: boolean;
  labelOperationsCallbacks: TLabelOperationsCallbacks;
  labelToUpdate?: IIssueLabel;
  onClose?: () => boolean | void;
};

export const CreateUpdateLabelInline = forwardRef(function CreateUpdateLabelInline(
  props: Props,
  ref: ForwardedRef<HTMLDivElement>
) {
  return props.labelForm ? <LabelForm key={props.labelToUpdate?.id ?? "new"} {...props} containerRef={ref} /> : null;
});

function LabelForm({
  setLabelForm,
  isUpdating,
  labelOperationsCallbacks,
  labelToUpdate,
  onClose,
  containerRef,
}: Props & { containerRef: ForwardedRef<HTMLDivElement> }) {
  const { t } = useTranslation();
  const lifecycleRef = useRef(false);
  useEffect(() => {
    lifecycleRef.current = true;
    return () => {
      lifecycleRef.current = false;
    };
  }, []);
  const [defaultValues] = useState(() => ({
    name: labelToUpdate?.name ?? "",
    color: labelToUpdate?.color || getRandomLabelColor(),
  }));
  const close = () => {
    if (!lifecycleRef.current || onClose?.() === false) return;
    setLabelForm(false);
  };
  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: ({ value }) => {
        if (!value.name.trim()) return t("project_settings.labels.label_title_is_required");
        if (value.name.length > 255) return t("project_settings.labels.label_max_char");
        if (isUpdating && !labelToUpdate) return t("common.something_went_wrong");
        return undefined;
      },
    },
    onSubmit: async ({ value }) => {
      try {
        const payload = { ...value, name: value.name.trim() };
        if (isUpdating) {
          if (!labelToUpdate) throw new Error("Cannot update a label without a target.");
          await labelOperationsCallbacks.updateLabel(labelToUpdate.id, payload);
        } else await labelOperationsCallbacks.createLabel(payload);
        close();
      } catch (error) {
        if (!lifecycleRef.current) throw error;
        const data = error && typeof error === "object" ? error : {};
        const duplicate =
          "name" in data &&
          (typeof data.name === "string" || Array.isArray(data.name)) &&
          data.name.includes("LABEL_NAME_ALREADY_EXISTS");
        const message = duplicate
          ? t("label.create.already_exists")
          : "detail" in data && typeof data.detail === "string"
            ? data.detail
            : "error" in data && typeof data.error === "string"
              ? data.error
              : t("common.something_went_wrong");
        setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message });
        throw error;
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  return (
    <div ref={containerRef} className="w-full scroll-m-8 bg-surface-1">
      <form
        className="flex w-full items-start gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <form.Field name="color">
          {(field) => (
            <Popover className="relative z-10 flex h-full shrink-0 items-center justify-center">
              <Popover.Button type="button" disabled={isSubmitting} aria-label="Label color" className="p-2">
                <span className="block h-4 w-4 rounded-full" style={{ backgroundColor: field.state.value }} />
              </Popover.Button>
              <Popover.Panel className="absolute top-full left-0 z-20 mt-3 w-screen max-w-xs px-2 sm:px-0">
                <TwitterPicker
                  colors={LABEL_COLOR_OPTIONS}
                  color={field.state.value}
                  onChange={(color) => {
                    if (!isSubmitting) field.handleChange(color.hex);
                  }}
                />
              </Popover.Panel>
            </Popover>
          )}
        </form.Field>
        <form.Field
          name="name"
          validators={{
            onChange: ({ value }) =>
              !value.trim()
                ? t("project_settings.labels.label_title_is_required")
                : value.length > 255
                  ? t("project_settings.labels.label_max_char")
                  : undefined,
            onSubmit: ({ value }) =>
              !value.trim()
                ? t("project_settings.labels.label_title_is_required")
                : value.length > 255
                  ? t("project_settings.labels.label_max_char")
                  : undefined,
          }}
        >
          {(field) => (
            <div className="flex flex-1 flex-col">
              <Input
                name={field.name}
                autoFocus
                disabled={isSubmitting}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                hasError={field.state.meta.errors.length > 0}
                placeholder={t("project_settings.labels.label_title")}
                className="w-full"
              />
              {field.state.meta.errors.map((error) => (
                <p key={error} role="alert" className="text-13 text-danger-primary">
                  {error}
                </p>
              ))}
            </div>
          )}
        </form.Field>
        <Button type="button" variant="secondary" disabled={isSubmitting} onClick={close}>
          {t("cancel")}
        </Button>
        <Button type="submit" variant="primary" loading={isSubmitting}>
          {isUpdating ? (isSubmitting ? t("updating") : t("update")) : isSubmitting ? t("adding") : t("add")}
        </Button>
      </form>
    </div>
  );
}
