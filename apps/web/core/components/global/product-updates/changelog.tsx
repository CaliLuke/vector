/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader } from "@plane/ui";
import { ProductUpdatesFallback } from "@/components/global/product-updates/fallback";
import { instanceConfigOptions } from "@/queries/instance";

export function ProductUpdatesChangelog() {
  const { data: config } = useQuery(instanceConfigOptions);
  const url = config?.instance_changelog_url;
  const fallback = (
    <ProductUpdatesFallback
      description="We're having trouble fetching the updates. Please visit our changelog to view the latest updates."
      variant={config?.is_self_managed ? "self-managed" : "cloud"}
    />
  );
  return url ? <ChangelogFrame key={url} url={url} fallback={fallback} /> : fallback;
}

function ChangelogFrame({ url, fallback }: { url: string; fallback: ReactNode }) {
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  useEffect(() => {
    if (status !== "loading") return;
    const timeout = setTimeout(() => setStatus("error"), 15_000);
    return () => clearTimeout(timeout);
  }, [status]);

  if (status === "error") return fallback;
  const isLoading = status === "loading";
  return (
    <div className="vertical-scrollbar relative mx-0.5 flex scrollbar-xs h-[550px] flex-col overflow-hidden overflow-y-scroll px-6">
      {isLoading && (
        <Loader className="absolute inset-0 flex h-full w-full flex-col items-center justify-center gap-3">
          <Loader.Item height="95%" width="95%" />
        </Loader>
      )}
      <iframe
        title="Product updates"
        sandbox="allow-scripts allow-popups allow-popups-to-escape-sandbox"
        src={url}
        className={`h-full w-full ${isLoading ? "opacity-0" : "opacity-100"} transition-opacity duration-200`}
        onLoad={() => setStatus("ready")}
      />
    </div>
  );
}
