/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation } from "@tanstack/react-router";
import { useState } from "react";

import { useQueryClient } from "@tanstack/react-query";
// constants
import { EXPORT_SERVICES_LIST } from "@plane/constants";
// local imports
import { ExportForm } from "./export-form";
import { PrevExports } from "./prev-exports";

export const ExportGuide = function ExportGuide() {
  const queryClient = useQueryClient();
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const provider = searchParams.get("provider");
  // state
  const per_page = 10;
  const [cursor, setCursor] = useState<string | undefined>(`10:0:0`);

  return (
    <>
      <div className="flex size-full flex-col gap-y-13">
        <ExportForm
          workspaceSlug={workspaceSlug}
          provider={provider}
          mutateServices={() =>
            queryClient.invalidateQueries({ queryKey: EXPORT_SERVICES_LIST(workspaceSlug, `${cursor}`, `${per_page}`) })
          }
        />
        <PrevExports workspaceSlug={workspaceSlug} cursor={cursor} per_page={per_page} setCursor={setCursor} />
      </div>
    </>
  );
};
