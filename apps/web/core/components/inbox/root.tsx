/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { PanelLeft } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { IntakeIcon } from "@plane/propel/icons";
import { EInboxIssueCurrentTab } from "@plane/types";
import { cn } from "@plane/utils/common";
// components
import { InboxContentRoot } from "@/components/inbox/content";
import { InboxSidebar } from "@/components/inbox/sidebar";
import { InboxLayoutLoader } from "@/components/ui/loader/layouts/project-inbox/inbox-layout-loader";
// hooks
import { IntakePreferencesScope, useIntakePreferences } from "@/providers/intake-preferences";
import { useIntakeList } from "@/hooks/use-intake";

type TInboxIssueRoot = {
  workspaceSlug: string;
  projectId: string;
  inboxIssueId: string | undefined;
  inboxAccessible: boolean;
  navigationTab?: EInboxIssueCurrentTab | undefined;
};

export function InboxIssueRoot(props: TInboxIssueRoot) {
  return (
    <IntakePreferencesScope
      workspaceSlug={props.workspaceSlug}
      projectId={props.projectId}
      navigationTab={props.navigationTab}
    >
      <InboxIssueScreen {...props} />
    </IntakePreferencesScope>
  );
}

function InboxIssueScreen(props: TInboxIssueRoot) {
  const { workspaceSlug, projectId, inboxIssueId, inboxAccessible } = props;
  // states
  const [isMobileSidebar, setIsMobileSidebar] = useState(true);
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const list = useIntakeList(inboxAccessible);
  const { tab } = useIntakePreferences();
  const issueIds = list.rows.map((row) => row.issue.id);
  if (!inboxAccessible || list.accessDenied) return <div className="p-4">Intake is unavailable for this project.</div>;

  // loader
  if (list.isPending)
    return (
      <div className="relative flex h-full w-full flex-col">
        <InboxLayoutLoader />
      </div>
    );

  // error
  if (list.isError && !list.isFetchNextPageError && !list.data)
    return (
      <div className="relative flex h-full w-full flex-col items-center justify-center gap-3">
        <IntakeIcon className="size-[60px]" strokeWidth={1.5} />
        <div className="text-secondary">Could not load intake work items.</div>
        <button type="button" onClick={() => void list.retry()}>
          Retry
        </button>
      </div>
    );

  return (
    <div className="flex h-full flex-col">
      {list.isError && !list.isFetchNextPageError && (
        <div role="alert" className="p-3">
          Could not refresh intake work items.{" "}
          <button type="button" onClick={() => void list.retry()}>
            Retry
          </button>
        </div>
      )}
      {!inboxIssueId && (
        <div className="flex h-12 w-full items-center border-b border-subtle px-4 lg:hidden">
          <PanelLeft
            onClick={() => setIsMobileSidebar(!isMobileSidebar)}
            className={cn("h-4 w-4", isMobileSidebar ? "text-accent-primary" : "text-secondary")}
          />
        </div>
      )}
      <div className="flex h-full w-full overflow-hidden bg-surface-1">
        <div
          className={cn(
            "absolute top-[50px] bottom-0 z-10 w-full flex-shrink-0 bg-surface-1 transition-all lg:!relative lg:!top-0 lg:w-2/6",
            isMobileSidebar ? "translate-x-0" : "-translate-x-full lg:!translate-x-0"
          )}
        >
          <InboxSidebar
            setIsMobileSidebar={setIsMobileSidebar}
            workspaceSlug={workspaceSlug.toString()}
            projectId={projectId.toString()}
            inboxIssueId={inboxIssueId}
          />
        </div>

        {inboxIssueId ? (
          <InboxContentRoot
            navigation={{ tab, issueIds }}
            setIsMobileSidebar={setIsMobileSidebar}
            isMobileSidebar={isMobileSidebar}
            workspaceSlug={workspaceSlug.toString()}
            projectId={projectId.toString()}
            inboxIssueId={inboxIssueId.toString()}
          />
        ) : (
          <EmptyStateCompact
            assetKey="intake"
            title={t("project_empty_state.intake_main.title")}
            assetClassName="size-20"
          />
        )}
      </div>
    </div>
  );
}
