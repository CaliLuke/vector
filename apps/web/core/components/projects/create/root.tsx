/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { FormErrors } from "@plane/ui";
import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { ETabIndices } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EFileAssetType } from "@plane/types";
import type { TProject } from "@plane/types";
import { Input, TextArea, ModalCore, EModalPosition, EModalWidth } from "@plane/ui";
import { getTabIndex } from "@plane/utils/tab-indices";
import { getFileURL } from "@plane/utils/file";
import { projectIdentifierSanitizer } from "@plane/utils/project";
import ProjectCreateHeader from "@/components/project/create/header";
import { ProjectNetworkSelect } from "@/components/project/project-network-select";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { getCoverImageType, uploadCoverImage } from "@/helpers/cover-image.helper";
import { useFavorites } from "@/hooks/use-favorites";
import { useProjectMutations } from "@/hooks/use-projects";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { getProjectFormValues } from "./utils";

type Props = {
  setToFavorite?: boolean;
  workspaceSlug: string;
  onClose: () => void;
  handleNextStep: (id: string) => void;
  data?: Partial<TProject>;
  updateCoverImageStatus: (id: string, url: string) => Promise<void>;
};
export function CreateProjectForm({
  setToFavorite,
  workspaceSlug,
  data,
  onClose,
  handleNextStep,
  updateCoverImageStatus,
}: Props) {
  const { t } = useTranslation();
  const projectMutation = useProjectMutations(workspaceSlug, null);
  const { addProjectFavorite } = useFavorites(workspaceSlug);
  const { isMobile } = usePlatformOS();
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_CREATE, isMobile);
  const [initial] = useState(() => getProjectFormValues(data));
  const [syncIdentifier, setSyncIdentifier] = useState(!data?.identifier);
  const create = useMutation({
    mutationFn: async (value: Partial<TProject>) => {
      const result = await projectMutation.mutateAsync({ type: "create", data: value });
      if (!result.project) throw new Error("The server did not return the created project.");
      return result.project;
    },
  });
  const cover = useMutation({
    mutationFn: async (source: string) => ({
      source,
      url: await uploadCoverImage(source, {
        workspaceSlug,
        entityIdentifier: "",
        entityType: EFileAssetType.PROJECT_COVER,
        isUserAsset: false,
      }),
    }),
  });
  const finalize = useMutation({
    mutationFn: ({ id, url }: { id: string; url: string }) => updateCoverImageStatus(id, url),
  });
  const favorite = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => addProjectFavorite(id, name),
  });
  const form = useForm({
    defaultValues: initial,
    onSubmit: async ({ value }) => {
      let coverUrl = value.cover_image_url;
      if (coverUrl && getCoverImageType(coverUrl) === "local_static") {
        coverUrl = (cover.data?.source === coverUrl ? cover.data : await cover.mutateAsync(coverUrl)).url;
      }
      const project =
        create.data ??
        (await create.mutateAsync({
          ...value,
          cover_image: coverUrl ? getFileURL(coverUrl) || coverUrl : null,
          cover_image_url: coverUrl,
        }));
      if (coverUrl) {
        if (!finalize.isSuccess) await finalize.mutateAsync({ id: project.id, url: coverUrl });
      }
      if (setToFavorite && !favorite.isSuccess) await favorite.mutateAsync({ id: project.id, name: project.name });
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("project_created_successfully") });
      handleNextStep(project.id);
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const locked = busy || create.isSuccess;
  const identityFields = {
    name: {
      placeholder: t("project_name"),
      validate: (value: string) =>
        !value.trim()
          ? t("name_is_required")
          : value.length > 255
            ? t("title_should_be_less_than_255_characters")
            : undefined,
      onChange: (value: string) => {
        form.setFieldValue("name", value);
        if (syncIdentifier)
          form.setFieldValue("identifier", projectIdentifierSanitizer(value).toUpperCase().slice(0, 10));
      },
      containerClassName: "md:col-span-3",
      inputClassName: "w-full",
    },
    identifier: {
      placeholder: t("project_id"),
      validate: (value: string) =>
        !value ? t("project_id_is_required") : value.length > 10 ? t("project_id_max_char") : undefined,
      onChange: (value: string) => {
        setSyncIdentifier(false);
        form.setFieldValue("identifier", projectIdentifierSanitizer(value).toUpperCase());
      },
      containerClassName: "",
      inputClassName: "w-full uppercase",
    },
  };
  const close = () => {
    if (!busy) onClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.TOP} width={EModalWidth.XXXXL}>
      <ProjectCreateHeader
        workspaceSlug={workspaceSlug}
        onClose={close}
        cover={values.cover_image_url}
        logo={values.logo_props}
        busy={busy}
        disabled={locked}
        onCoverChange={(url) => form.setFieldValue("cover_image_url", url)}
        onLogoChange={(logo) => form.setFieldValue("logo_props", logo)}
      />
      <form
        className="px-3"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {
            form.setErrorMap({
              onSubmit: {
                form: "Could not finish setting up the project. Your progress is saved; please try again.",
                fields: {},
              },
            });
          });
        }}
      >
        <fieldset disabled={locked} inert={locked} className="mt-9 space-y-6 pb-5">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-4">
            {(["name", "identifier"] as const).map((name) => {
              const input = identityFields[name];
              return (
                <form.Field key={name} name={name} validators={{ onChange: ({ value }) => input.validate(value) }}>
                  {(field) => (
                    <div className={input.containerClassName}>
                      <Input
                        name={field.name}
                        value={field.state.value}
                        placeholder={input.placeholder}
                        className={input.inputClassName}
                        tabIndex={getIndex(name)}
                        onChange={(event) => input.onChange(event.target.value)}
                        onBlur={field.handleBlur}
                        hasError={!field.state.meta.isValid}
                      />
                      <FormErrors errors={field.state.meta.errors} className="text-11" />
                    </div>
                  )}
                </form.Field>
              );
            })}
            <TextArea
              name="description"
              value={values.description}
              onChange={(event) => form.setFieldValue("description", event.target.value)}
              placeholder={t("description")}
              className="!h-24 text-13 md:col-span-4"
              tabIndex={getIndex("description")}
            />
          </div>
          <div className="flex gap-2">
            <ProjectNetworkSelect
              value={values.network}
              onChange={(value) => form.setFieldValue("network", value)}
              disabled={locked}
              tabIndex={getIndex("network")}
            />
            <MemberDropdown
              workspaceSlug={workspaceSlug}
              value={values.project_lead}
              onChange={(value) => form.setFieldValue("project_lead", value)}
              multiple={false}
              placeholder={t("lead")}
              buttonVariant="border-with-text"
              disabled={locked}
              tabIndex={getIndex("lead")}
            />
          </div>
        </fieldset>
        {formError && (
          <p role="alert" className="text-13 text-danger-primary">
            {typeof formError === "string"
              ? formError
              : "Could not finish setting up the project. Your progress is saved; please try again."}
          </p>
        )}
        <div className="flex justify-end gap-2 border-t border-subtle py-4">
          <Button type="button" variant="secondary" disabled={busy} onClick={close}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" variant="primary" loading={busy}>
            {create.isSuccess ? "Finish setup" : t("create_project")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
