/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
// plane types
// plane ui
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { Input, ModalCore } from "@plane/ui";

type LinkValues = { title: string; url: string };
type Props = {
  isOpen: boolean;
  onClose: () => void;
  onSave: (value: LinkValues) => Promise<void>;
  link?: { id: string; title?: string; url: string } | null;
  heading?: string;
};

export function LinkModal(props: Props) {
  if (!props.isOpen) return null;
  return <LinkForm key={props.link?.id ?? "new"} {...props} />;
}

function LinkForm({ onClose, onSave, link, heading }: Props) {
  const { t } = useTranslation();
  const form = useForm({
    defaultValues: { title: link?.title ?? "", url: link?.url ?? "" },
    onSubmit: async ({ value }) => {
      const url = /^https?:\/\//i.test(value.url) ? value.url : `http://${value.url}`;
      await onSave({ title: value.title, url });
      onClose();
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);

  return (
    <ModalCore
      isOpen
      handleClose={() => {
        if (!isSubmitting) onClose();
      }}
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <div className="space-y-5 p-5">
          <h3 className="text-18 font-medium text-secondary">
            {heading ?? (link ? t("common.update_link") : t("common.add_link"))}
          </h3>
          <div className="mt-2 space-y-3">
            <div>
              <label htmlFor="url" className="mb-2 text-14 font-medium text-secondary">
                {t("link.modal.url.text")}
                <span className="block text-10">{t("required")}</span>
              </label>
              <form.Field
                name="url"
                validators={{
                  onChange: ({ value }) => (value.trim() ? undefined : t("link.modal.url.required")),
                  onSubmit: ({ value }) => (value.trim() ? undefined : t("link.modal.url.required")),
                }}
              >
                {(field) => (
                  <>
                    <Input
                      id="url"
                      name="url"
                      type="text"
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                      onBlur={field.handleBlur}
                      hasError={!field.state.meta.isValid}
                      placeholder={t("link.modal.url.placeholder")}
                      className="w-full"
                      disabled={isSubmitting}
                    />
                    {!field.state.meta.isValid && (
                      <span role="alert" className="text-11 text-danger-primary">
                        {field.state.meta.errors.join(" ")}
                      </span>
                    )}
                  </>
                )}
              </form.Field>
            </div>
            <div>
              <label htmlFor="title" className="mb-2 text-14 font-medium text-secondary">
                {t("link.modal.title.text")}
                <span className="block text-10">{t("optional")}</span>
              </label>
              <form.Field name="title">
                {(field) => (
                  <Input
                    id="title"
                    name="title"
                    type="text"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                    onBlur={field.handleBlur}
                    placeholder={t("link.modal.title.placeholder")}
                    className="w-full"
                    disabled={isSubmitting}
                  />
                )}
              </form.Field>
            </div>
          </div>
        </div>
        <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
          <Button variant="secondary" type="button" size="lg" onClick={onClose} disabled={isSubmitting}>
            {t("Cancel")}
          </Button>
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
            {link?.id ? (isSubmitting ? t("updating") : t("update")) : isSubmitting ? t("adding") : t("add")}{" "}
            {t("common.link")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
