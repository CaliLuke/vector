/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";

// ui
import { SITES_ISSUE_LAYOUTS } from "@plane/constants";
// plane i18n
import { useTranslation } from "@plane/i18n";
import { Tooltip } from "@plane/propel/tooltip";
// helpers
// hooks
import { usePublicIssueView } from "@/hooks/use-public-issue-view";
import { IssueLayoutIcon } from "./layout-icon";

type Props = {
  anchor: string;
};

export const IssuesLayoutSelection = function IssuesLayoutSelection(props: Props) {
  const { anchor } = props;
  // hooks
  const { t } = useTranslation();
  const { layout: activeLayout, setLayout } = usePublicIssueView(anchor);
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));

  return (
    <div className="flex items-center gap-1 rounded-sm bg-layer-2 p-1">
      {SITES_ISSUE_LAYOUTS.map((layout) => {
        const allowed = publishSettings?.view_props ? publishSettings.view_props[layout.key] : layout.key === "list";
        if (!allowed) return null;

        return (
          <Tooltip key={layout.key} tooltipContent={t(layout.titleTranslationKey)}>
            <button
              type="button"
              className={`group grid h-[22px] w-7 place-items-center overflow-hidden rounded-sm bg-layer-transparent transition-all hover:bg-layer-transparent-hover ${
                activeLayout == layout.key ? "bg-layer-transparent-active hover:bg-layer-transparent-selected" : ""
              }`}
              onClick={() => setLayout(layout.key)}
            >
              <IssueLayoutIcon
                layout={layout.key}
                className={`size-3.5 ${activeLayout == layout.key ? "text-primary" : "text-secondary"}`}
              />
            </button>
          </Tooltip>
        );
      })}
    </div>
  );
};
