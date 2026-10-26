/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
// plane types
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { InstructionType, TSticky } from "@plane/types";
// plane utils
import { isCommentEmpty } from "@plane/utils/string";
// components
import { STICKY_COLORS_LIST } from "@/components/editor/sticky-editor/color-palette";
// hooks
import { useStickies } from "@/components/stickies/provider";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { changeStickyOptions, createStickyOptions, stickyKeys } from "@/queries/stickies";
import { reorderedSortOrder } from "@/lib/reorder";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";

export type TOperations = {
  create: (data?: Partial<TSticky>) => Promise<void>;
  update: (stickyId: string, data: Partial<TSticky>) => Promise<void>;
  remove: (stickyId: string) => Promise<void>;
  updatePosition: (
    workspaceSlug: string,
    sourceId: string,
    droppedId: string,
    instruction: InstructionType
  ) => Promise<void>;
};

type TProps = {
  workspaceSlug: string;
  stickyId?: string;
};

const getRandomStickyColor = (): string => {
  const randomIndex = Math.floor(Math.random() * STICKY_COLORS_LIST.length);
  return STICKY_COLORS_LIST[randomIndex].key;
};

export const useStickyOperations = (props: TProps) => {
  const { workspaceSlug, stickyId } = props;
  const { stickies, stickyIds } = useStickies();
  const client = useQueryClient();
  const accountId = useCurrentUserId();
  const generation = useIdentityGeneration(client);
  const identity = useMemo(() => {
    const captured = captureIdentityScope(client, accountId);
    return captured?.generation === generation ? captured : undefined;
  }, [client, accountId, generation]);
  const isCurrent = useCallback(() => isIdentityScopeCurrent(client, identity), [client, identity]);
  const sessionKey = JSON.stringify([accountId, generation]);
  const { mutateAsync: create } = useMutation(createStickyOptions(client, workspaceSlug));
  const { mutateAsync: change } = useMutation(changeStickyOptions(client, workspaceSlug, stickyId ?? "reorder"));
  const { t } = useTranslation();

  const isValid = (data: Partial<TSticky>) => {
    if (data.name && data.name.length > 100) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("stickies.toasts.not_updated.title"),
        message: t("stickies.toasts.errors.wrong_name"),
      });
      return false;
    }
    return true;
  };

  const createSticky = async (data?: Partial<TSticky>) => {
    if (!isCurrent()) return;
    if (client.isMutating({ mutationKey: [...stickyKeys.create(workspaceSlug), accountId, generation] })) return;
    const latest = stickies.get(stickyIds[0]);
    if (latest && (!latest.description_html || isCommentEmpty(latest.description_html))) {
      setToast({
        type: TOAST_TYPE.WARNING,
        title: t("stickies.toasts.not_created.title"),
        message: t("stickies.toasts.errors.already_exists"),
      });
      return;
    }
    const payload = { background_color: getRandomStickyColor(), ...data };
    if (!isValid(payload)) return;
    try {
      await create(payload);
      if (!isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("stickies.toasts.created.title"),
        message: t("stickies.toasts.created.message"),
      });
    } catch (error) {
      if (!isCurrent()) return;
      console.error("Error creating sticky", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("stickies.toasts.not_created.title"),
        message: t("stickies.toasts.not_created.message"),
      });
    }
  };
  const update = useCallback(
    async (id: string, patch: Partial<TSticky>) => {
      if (!isCurrent()) throw new Error("The account session changed.");
      if (patch.name && patch.name.length > 100) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("stickies.toasts.not_updated.title"),
          message: t("stickies.toasts.errors.wrong_name"),
        });
        throw new Error("Sticky name must be 100 characters or fewer.");
      }
      try {
        await change({ kind: "update", id, patch });
      } catch (error) {
        if (isCurrent()) {
          console.error("Error updating sticky", error);
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("stickies.toasts.not_updated.title"),
            message: t("stickies.toasts.not_updated.message"),
          });
        }
        throw error;
      }
    },
    [change, t, isCurrent]
  );
  const remove = async (id: string) => {
    if (!isCurrent()) return;
    try {
      await change({ kind: "remove", id });
      if (!isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("stickies.toasts.removed.title"),
        message: t("stickies.toasts.removed.message"),
      });
    } catch (error) {
      if (isCurrent()) {
        console.error("Error removing sticky", error);
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("stickies.toasts.not_removed.title"),
          message: t("stickies.toasts.not_removed.message"),
        });
      }
      throw error;
    }
  };
  const updatePosition: TOperations["updatePosition"] = async (_workspace, sourceId, targetId, edge) => {
    if (sourceId === targetId || (edge !== "reorder-above" && edge !== "reorder-below")) return;
    const sort_order = reorderedSortOrder([...stickies.values()], sourceId, targetId, edge);
    try {
      await update(sourceId, { sort_order });
    } catch {
      // The update operation already reports failures to the user.
    }
  };
  return { isCurrent, sessionKey, stickyOperations: { create: createSticky, update, remove, updatePosition } };
};
