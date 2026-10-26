/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TProjectLink } from "@plane/types";
import { homeLinkMutationOptions } from "@/queries/home";

export type TLinkOperations = {
  create: (data: Partial<TProjectLink>) => Promise<void>;
  update: (linkId: string, data: Partial<TProjectLink>) => Promise<void>;
  remove: (linkId: string) => Promise<void>;
};

export const useLinks = (workspaceSlug: string, closeModal: () => void) => {
  const mutation = useMutation(homeLinkMutationOptions(useQueryClient(), workspaceSlug));
  const { t } = useTranslation();
  const run = async (change: Parameters<typeof mutation.mutateAsync>[0], action: "created" | "updated" | "removed") => {
    try {
      await mutation.mutateAsync(change);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t(`links.toasts.${action}.title`),
        message: t(`links.toasts.${action}.message`),
      });
      closeModal();
    } catch (error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t(`links.toasts.not_${action}.title`),
        message: t(`links.toasts.not_${action}.message`),
      });
      throw error;
    }
  };
  const linkOperations: TLinkOperations = {
    create: (data) => run({ kind: "create", data }, "created"),
    update: (id, data) => run({ kind: "update", id, data }, "updated"),
    remove: (id) => run({ kind: "remove", id }, "removed"),
  };
  return { linkOperations };
};
