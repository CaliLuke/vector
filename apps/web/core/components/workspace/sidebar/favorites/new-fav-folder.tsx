/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useEffect, useRef } from "react";

import { useForm, useStore } from "@tanstack/react-form";
import { useOutsideClickDetector } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
// plane helpers
// plane ui
import { FavoriteFolderIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Input } from "@plane/ui";
// hooks
import { useFavorites } from "@/hooks/use-favorites";

type TProps = {
  setCreateNewFolder: (value: boolean | string | null) => void;
  actionType: "create" | "rename";
  defaultName?: string;
  favoriteId?: string;
};
export function NewFavoriteFolder(props: TProps) {
  const { setCreateNewFolder, actionType, defaultName, favoriteId } = props;
  const { t } = useTranslation();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { addFavorite, updateFavorite, existingFolders } = useFavorites(workspaceSlug);

  // ref
  const ref = useRef(null);
  const inputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const form = useForm({
    defaultValues: { name: defaultName ?? "" },
    validators: {
      onChange: ({ value }) => {
        const name = value.name.trim();
        if (!name) return t("folder_name_cannot_be_empty");
        if (name !== defaultName && existingFolders.includes(name)) return t("folder_already_exists");
        return undefined;
      },
      onSubmit: (): string | undefined =>
        actionType === "rename" && !favoriteId ? "Unable to rename this folder." : undefined,
    },
    onSubmit: async ({ value }) => {
      try {
        const name = value.name.trim();
        if (actionType === "rename") {
          if (!favoriteId) throw new Error("A favorite folder is required to rename.");
          await updateFavorite(favoriteId, { name });
        } else {
          await addFavorite({
            name,
            entity_type: "folder",
            is_folder: true,
            parent: null,
            project_id: null,
          });
        }
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("success"),
          message: t(actionType === "rename" ? "favorite_updated_successfully" : "favorite_created_successfully"),
        });
        setCreateNewFolder(false);
      } catch (error) {
        setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("something_went_wrong") });
        throw error;
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const errors = useStore(form.store, (state) => state.errors);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  useOutsideClickDetector(ref, () => {
    if (!isSubmitting) setCreateNewFolder(false);
  });
  return (
    <div className="flex items-center gap-1.5 px-2 py-[1px]" ref={ref}>
      <FavoriteFolderIcon className="size-4" />
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {
            form.setErrorMap({ onSubmit: t("something_went_wrong") });
          });
        }}
      >
        <form.Field name="name">
          {(field) => (
            <Input
              name={field.name}
              value={field.state.value}
              onChange={(event) => field.handleChange(event.target.value)}
              onBlur={field.handleBlur}
              ref={inputRef}
              disabled={isSubmitting}
              className="w-full"
              placeholder={t("new_folder")}
              aria-label={t("aria_labels.projects_sidebar.enter_folder_name")}
              hasError={errors.length > 0}
            />
          )}
        </form.Field>
        {errors.map((error) => (
          <p key={error} role="alert" className="text-11 text-danger-primary">
            {error}
          </p>
        ))}
        {formError && (
          <p role="alert" className="text-11 text-danger-primary">
            {formError}
          </p>
        )}
      </form>
    </div>
  );
}
