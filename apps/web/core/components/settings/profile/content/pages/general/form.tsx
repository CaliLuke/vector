/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useQuery, useMutation } from "@tanstack/react-query";
import { CircleUserRound } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EFileAssetType } from "@plane/types";
import type { IUser } from "@plane/types";
import { Input } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
import { validatePersonName, validateDisplayName } from "@plane/utils/validation";
import { DeactivateAccountModal } from "@/components/account/deactivate-account-modal";
import { ImagePickerPopover } from "@/components/core/image-picker-popover";
import { ChangeEmailModal } from "@/components/core/modals/change-email-modal";
import { UserImageUploadModal } from "@/components/core/modals/user-image-upload-modal";
import { CoverImage } from "@/components/common/cover-image";
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
import { handleCoverImageChange } from "@/helpers/cover-image.helper";
import { instanceConfigOptions } from "@/queries/instance";
import { useUpdateCurrentUser } from "@/hooks/use-current-user";

const nameFields = [
  { name: "first_name", required: true, placeholder: "Enter your first name" },
  { name: "last_name", required: false, placeholder: "Enter your last name" },
  { name: "display_name", required: true, placeholder: "Enter your display name" },
] as const;
function userDefaults(user: IUser) {
  return {
    avatar_url: user.avatar_url ?? "",
    cover_image_url: user.cover_image_url ?? "",
    first_name: user.first_name ?? "",
    last_name: user.last_name ?? "",
    display_name: user.display_name ?? "",
  };
}
export function GeneralProfileSettingsForm({ user }: { user: IUser }) {
  const [imageOpen, setImageOpen] = useState(false);
  const [deactivateOpen, setDeactivateOpen] = useState(false);
  const [emailOpen, setEmailOpen] = useState(false);
  const { t } = useTranslation();
  const { mutateAsync: updateCurrentUser } = useUpdateCurrentUser();
  const { data: config } = useQuery(instanceConfigOptions);
  const cover = useMutation({
    mutationFn: async (url: string) => ({
      url,
      payload: await handleCoverImageChange(user.cover_image_url, url, {
        entityIdentifier: "",
        entityType: EFileAssetType.USER_COVER,
        isUserAsset: true,
      }),
    }),
  });
  const form = useForm({
    defaultValues: userDefaults(user),
    validators: { onSubmit: (): string | undefined => undefined },
    onSubmit: async ({ value, formApi }) => {
      const processed =
        cover.data?.url === value.cover_image_url ? cover.data : await cover.mutateAsync(value.cover_image_url);
      const saved = await updateCurrentUser({
        first_name: value.first_name,
        last_name: value.last_name,
        display_name: value.display_name,
        avatar_url: value.avatar_url,
        ...processed.payload,
      });
      if (!saved) throw new Error("Profile update did not complete");
      formApi.reset(userDefaults(saved));
      cover.reset();
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Profile updated successfully." });
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const removeAvatar = async () => {
    await updateCurrentUser({ avatar_url: "" });
    form.setFieldValue("avatar_url", "");
  };
  return (
    <>
      <DeactivateAccountModal isOpen={deactivateOpen} onClose={() => setDeactivateOpen(false)} />
      <ChangeEmailModal isOpen={emailOpen} onClose={() => setEmailOpen(false)} />
      {imageOpen && (
        <UserImageUploadModal
          isOpen
          onClose={() => setImageOpen(false)}
          handleRemove={removeAvatar}
          value={values.avatar_url || null}
          onSuccess={async (url) => {
            const saved = await updateCurrentUser({ avatar_url: url });
            if (!saved) throw new Error("Profile update did not complete");
            form.setFieldValue("avatar_url", saved.avatar_url ?? "");
          }}
        />
      )}
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() =>
            form.setErrorMap({
              onSubmit: { form: "Could not update your profile. Please try again.", fields: {} },
            })
          );
        }}
        noValidate
        className="w-full"
      >
        <fieldset disabled={busy} inert={busy} className="flex w-full flex-col gap-7">
          <div className="relative h-44 w-full">
            <CoverImage src={values.cover_image_url} className="h-44 w-full rounded-lg" alt="Cover image" />
            <button
              type="button"
              aria-label="Change profile picture"
              onClick={() => setImageOpen(true)}
              className="absolute -bottom-6 left-6 grid size-16 place-items-center rounded-lg bg-surface-2"
            >
              {values.avatar_url ? (
                <img
                  src={getFileURL(values.avatar_url)}
                  className="size-16 rounded-lg object-cover"
                  alt={user.display_name}
                />
              ) : (
                <CircleUserRound className="size-12 text-secondary" />
              )}
            </button>
            <div className="absolute right-3 bottom-3">
              <ImagePickerPopover
                label={t("change_cover")}
                value={values.cover_image_url}
                onChange={(url) => form.setFieldValue("cover_image_url", url)}
                isProfileCover
              />
            </div>
          </div>
          <div className="mt-6">
            <div className="text-16 font-medium text-secondary">
              {values.first_name} {values.last_name}
            </div>
            <span className="text-13 text-tertiary">{user.email}</span>
          </div>
          <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2 xl:grid-cols-3">
            {nameFields.map(({ name, required, placeholder }) => (
              <form.Field
                key={name}
                name={name}
                validators={{
                  onChange: ({ value }) => {
                    if (required && !value.trim()) return "This field is required.";
                    const result = name === "display_name" ? validateDisplayName(value) : validatePersonName(value);
                    return result === true ? undefined : result;
                  },
                }}
              >
                {(field) => (
                  <label className="flex flex-col gap-1 text-13 font-medium text-secondary">
                    {t(name)}
                    {required && <span className="text-danger-primary">*</span>}
                    <Input
                      name={field.name}
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                      onBlur={field.handleBlur}
                      hasError={!field.state.meta.isValid}
                      placeholder={placeholder}
                      className="w-full rounded-md"
                      maxLength={50}
                    />
                    {field.state.meta.errors.map((message) => (
                      <span role="alert" key={String(message)} className="text-11 text-danger-primary">
                        {message}
                      </span>
                    ))}
                  </label>
                )}
              </form.Field>
            ))}
            <div className="flex flex-col gap-1">
              <label className="text-13 font-medium text-secondary">
                {t("auth.common.email.label")}
                <Input type="email" value={user.email} disabled className="w-full rounded-md !bg-surface-2" />
              </label>
              {config?.is_smtp_configured && (
                <button
                  type="button"
                  onClick={() => setEmailOpen(true)}
                  className="w-fit text-11 text-secondary underline"
                >
                  {t("account_settings.profile.change_email_modal.title")}
                </button>
              )}
            </div>
          </div>
          {error && (
            <p role="alert" className="text-13 text-danger-primary">
              {error}
            </p>
          )}
          <div>
            <Button type="submit" variant="primary" loading={busy}>
              {t("save_changes")}
            </Button>
          </div>
        </fieldset>
      </form>
      <div className="mt-10">
        <SettingsBoxedControlItem
          title={t("deactivate_account")}
          description={t("deactivate_account_description")}
          control={
            <Button type="button" variant="error-outline" disabled={busy} onClick={() => setDeactivateOpen(true)}>
              {t("deactivate_account")}
            </Button>
          }
        />
      </div>
    </>
  );
}
