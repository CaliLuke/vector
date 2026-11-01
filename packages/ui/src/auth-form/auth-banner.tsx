/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Info } from "lucide-react";
import { Banner } from "@plane/propel/banner";

export function AuthMessageBanner({
  bannerData,
  handleBannerData,
}: {
  bannerData: { message: React.ReactNode } | undefined;
  handleBannerData?: (value: undefined) => void;
}) {
  if (!bannerData) return null;
  return (
    <Banner
      role="alert"
      variant="info"
      className="h-auto gap-2 rounded-md border border-accent-strong/50 bg-accent-primary/10 p-2 text-accent-primary"
      icon={<Info size={16} className="text-accent-primary" />}
      title={<span className="text-accent-primary">{bannerData.message}</span>}
      dismissible={!!handleBannerData}
      onDismiss={handleBannerData ? () => handleBannerData(undefined) : undefined}
    />
  );
}
