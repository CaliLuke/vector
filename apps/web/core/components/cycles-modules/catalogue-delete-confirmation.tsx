/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AlertModalCore } from "@plane/ui";

type Props = {
  entity: "cycle" | "module";
  itemName: string | undefined;
  handleClose: () => void;
  handleSubmit: () => Promise<void>;
  isSubmitting: boolean;
  isOpen: boolean;
};

export function CatalogueDeleteConfirmation({
  entity,
  itemName,
  handleClose,
  handleSubmit,
  isSubmitting,
  isOpen,
}: Props) {
  const isCycle = entity === "cycle";
  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={() => {
        void handleSubmit();
      }}
      isSubmitting={isSubmitting}
      isOpen={isOpen}
      title={`Delete ${entity}`}
      content={
        isCycle ? (
          <>
            Are you sure you want to delete cycle{' "'}
            <span className="font-medium break-words text-primary">{itemName}</span>
            {'"'}? All of the data related to the cycle will be permanently removed. This action cannot be undone.
          </>
        ) : (
          <>
            Are you sure you want to delete module-{" "}
            <span className="font-medium break-all text-primary">{itemName}</span>? All of the data related to the
            module will be permanently removed. This action cannot be undone.
          </>
        )
      }
    />
  );
}
