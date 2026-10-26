/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { logoPropsFromSelection } from "@/helpers/emoji.helper";
import { useEffect, useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import type { LucideIcon } from "lucide-react";
import { ETabIndices, EPageAccess } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EmojiPicker, EmojiIconPickerTypes, Logo } from "@plane/propel/emoji-icon-picker";
import { GlobeIcon, LockIcon, PageIcon } from "@plane/propel/icons";
import type { ISvgIcons } from "@plane/propel/icons";
import type { TLogoProps, TPage } from "@plane/types";
import { Input, ModalCore, EModalPosition, EModalWidth } from "@plane/ui";
import { getTabIndex } from "@plane/utils/tab-indices";
import { AccessField } from "@/components/common/access-field";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  pageAccess?: EPageAccess;
  handleModalClose: () => void;
  handleFormSubmit: (values: Partial<TPage>) => Promise<void>;
};
const PAGE_ACCESS_SPECIFIERS: { key: EPageAccess; i18n_label: string; icon: LucideIcon | React.FC<ISvgIcons> }[] = [
  { key: EPageAccess.PUBLIC, i18n_label: "common.access.public", icon: GlobeIcon },
  { key: EPageAccess.PRIVATE, i18n_label: "common.access.private", icon: LockIcon },
];
export function PageForm({ pageAccess = EPageAccess.PUBLIC, handleModalClose, handleFormSubmit }: Props) {
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const [pickerOpen, setPickerOpen] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const [initial] = useState<{ name: string; access: EPageAccess; logo_props: TLogoProps | undefined }>(() => ({
    name: "",
    access: pageAccess,
    logo_props: undefined,
  }));
  const form = useForm({
    defaultValues: initial,
    onSubmit: async ({ value }) => {
      form.setErrorMap({ onSubmit: undefined });
      try {
        await handleFormSubmit(value);
      } catch (cause) {
        form.setErrorMap({
          onSubmit: { form: "Could not create the page. Your changes are still here; please try again.", fields: {} },
        });
        throw cause;
      }
    },
  });
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const values = useStore(form.store, (state) => state.values);
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_PAGE, isMobile);
  const close = () => {
    if (!busy) handleModalClose();
  };
  useEffect(() => {
    input.current?.focus();
  }, []);
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <fieldset disabled={busy} inert={busy}>
          <div className="space-y-5 p-5">
            <h3 className="text-18 font-medium text-secondary">Create page</h3>
            <div className="flex min-h-9 w-full items-start gap-2">
              <EmojiPicker
                isOpen={pickerOpen}
                handleToggle={setPickerOpen}
                disabled={busy}
                className="flex-shrink0 flex items-center justify-center"
                buttonClassName="flex items-center justify-center bg-layer-2 hover:bg-layer-2-hover rounded-md"
                label={
                  <span className="grid size-9 place-items-center rounded-md">
                    {values.logo_props?.in_use ? (
                      <Logo logo={values.logo_props} size={18} type="lucide" />
                    ) : (
                      <PageIcon className="size-4 text-tertiary" />
                    )}
                  </span>
                }
                onChange={(selection) => {
                  form.setFieldValue("logo_props", logoPropsFromSelection(selection));
                  setPickerOpen(false);
                }}
                defaultIconColor={values.logo_props?.icon?.color}
                defaultOpen={
                  values.logo_props?.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON
                }
              />
              <form.Field
                name="name"
                validators={{
                  onChange: ({ value }) =>
                    !value.trim()
                      ? "Title is required"
                      : value.length > 255
                        ? "Title must be at most 255 characters"
                        : undefined,
                }}
              >
                {(field) => (
                  <div className="w-full space-y-1">
                    <Input
                      ref={input}
                      name={field.name}
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                      onBlur={field.handleBlur}
                      hasError={!field.state.meta.isValid}
                      placeholder="Title"
                      className="w-full resize-none text-14"
                      tabIndex={getIndex("name")}
                    />
                    {field.state.meta.errors.map((message) => (
                      <p key={message} role="alert" className="text-11 text-danger-primary">
                        {message}
                      </p>
                    ))}
                  </div>
                )}
              </form.Field>
            </div>
            {error && (
              <p role="alert" className="text-13 text-danger-primary">
                {error}
              </p>
            )}
          </div>
          <div className="flex items-center justify-between gap-2 border-t-[0.5px] border-subtle px-5 py-4">
            <div className="flex items-center gap-2">
              <AccessField
                value={values.access}
                onChange={(access) => form.setFieldValue("access", access)}
                accessSpecifiers={PAGE_ACCESS_SPECIFIERS}
                isMobile={isMobile}
              />
              <h6 className="text-11 font-medium">
                {t(values.access === EPageAccess.PUBLIC ? "common.access.public" : "common.access.private")}
              </h6>
            </div>
            <div className="flex items-center justify-end gap-2">
              <Button variant="secondary" size="lg" type="button" onClick={close} tabIndex={getIndex("cancel")}>
                Cancel
              </Button>
              <Button
                variant="primary"
                size="lg"
                type="submit"
                loading={busy}
                disabled={busy}
                tabIndex={getIndex("submit")}
              >
                {busy ? "Creating" : "Create Page"}
              </Button>
            </div>
          </div>
        </fieldset>
      </form>
    </ModalCore>
  );
}
