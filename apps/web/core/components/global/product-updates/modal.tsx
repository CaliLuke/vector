/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// components
import { ProductUpdatesFooter } from "@/components/global";
// plane web components
import { ProductUpdatesChangelog } from "@/components/global/product-updates/changelog";
import { ProductUpdatesHeader } from "@/components/global/product-updates/header";

export type ProductUpdatesModalProps = {
  isOpen: boolean;
  handleClose: () => void;
};

export function ProductUpdatesModal(props: ProductUpdatesModalProps) {
  const { isOpen, handleClose } = props;

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXXXL}>
      <ProductUpdatesHeader />
      <ProductUpdatesChangelog />
      <ProductUpdatesFooter />
    </ModalCore>
  );
}
