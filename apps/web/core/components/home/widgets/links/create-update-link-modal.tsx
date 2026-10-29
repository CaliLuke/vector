/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import type { TLinkEditableFields } from "@plane/types";
import { LinkModal } from "@/components/common/link-modal";
import type { TLinkOperations } from "./use-links";

type TLinkOperationsModal = Exclude<TLinkOperations, "remove">;
type TLinkCreateFormFieldOptions = TLinkEditableFields & { id?: string };
export type TLinkCreateEditModal = {
  isModalOpen: boolean;
  handleOnClose?: () => void;
  linkOperations: TLinkOperationsModal;
  preloadedData?: TLinkCreateFormFieldOptions;
};

export function LinkCreateUpdateModal({
  isModalOpen,
  handleOnClose,
  linkOperations,
  preloadedData,
}: TLinkCreateEditModal) {
  const { t } = useTranslation();
  return (
    <LinkModal
      isOpen={isModalOpen}
      onClose={() => handleOnClose?.()}
      link={preloadedData?.id ? { ...preloadedData, id: preloadedData.id } : undefined}
      heading={`${preloadedData?.id ? t("update") : t("add")} ${t("home.quick_links.title")}`}
      onSave={(value) =>
        preloadedData?.id ? linkOperations.update(preloadedData.id, value) : linkOperations.create(value)
      }
    />
  );
}
