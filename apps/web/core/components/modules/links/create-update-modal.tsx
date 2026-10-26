/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { LinkModal } from "@/components/common/link-modal";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ILinkDetails, ModuleLink } from "@plane/types";

type LinkReceipt = { captured: { isCurrent: () => boolean } };
type Props = {
  createLink: (formData: ModuleLink) => Promise<LinkReceipt>;
  data?: ILinkDetails | null;
  isOpen: boolean;
  handleClose: () => void;
  updateLink: (formData: ModuleLink, linkId: string) => Promise<LinkReceipt>;
};

export function CreateUpdateModuleLinkModal({ isOpen, handleClose, createLink, updateLink, data }: Props) {
  const current = useCommittedScope(JSON.stringify([isOpen, data?.id]));
  const completion = useRef<(() => boolean) | undefined>(undefined);
  return (
    <LinkModal
      isOpen={isOpen}
      onClose={() => {
        if (current() && (!completion.current || completion.current())) handleClose();
      }}
      link={data}
      onSave={async (value) => {
        try {
          const result = data ? await updateLink(value, data.id) : await createLink(value);
          completion.current = result.captured.isCurrent;
          if (!current() || !result.captured.isCurrent()) return;
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: "Success!",
            message: `Module link ${data ? "updated" : "created"} successfully.`,
          });
        } catch (error) {
          if (current())
            setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: "Some error occurred. Please try again." });
          throw error;
        }
      }}
    />
  );
}
