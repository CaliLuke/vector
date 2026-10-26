/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";

import { EUserPermissions, LOGIN_MEDIUM_LABELS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { renderFormattedDate } from "@plane/utils/datetime";
import { MemberHeaderColumn } from "@/components/project/member-header-column";
import type { RowData } from "@/components/workspace/settings/member-columns";
import { AccountTypeColumn, NameColumn } from "@/components/workspace/settings/member-columns";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import type { IMemberFilters } from "@/helpers/member-ordering";

export const useMemberColumns = (props: {
  filters: IMemberFilters;
  onFilterUpdate: (updates: Partial<IMemberFilters>) => void;
}) => {
  const { filters, onFilterUpdate } = props;
  // states
  const [removeMemberModal, setRemoveMemberModal] = useState<RowData | null>(null);

  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });

  const { data: currentUser } = useCurrentUser();
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const { t } = useTranslation();

  // derived values
  const isAdmin = canWorkspace([EUserPermissions.ADMIN]);

  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const isSuspended = (rowData: RowData) => rowData.is_active === false;

  // handlers
  const columns = [
    {
      key: "Full name",
      content: t("workspace_settings.settings.members.details.full_name"),
      thClassName: "text-left",
      thRender: () => (
        <MemberHeaderColumn property="full_name" displayFilters={filters} handleDisplayFilterUpdate={onFilterUpdate} />
      ),
      tdRender: (rowData: RowData) => (
        <NameColumn
          rowData={rowData}
          workspaceSlug={workspaceSlug}
          isAdmin={isAdmin}
          currentUser={currentUser ?? undefined}
          setRemoveMemberModal={setRemoveMemberModal}
        />
      ),
    },

    {
      key: "Display name",
      content: t("workspace_settings.settings.members.details.display_name"),
      tdRender: (rowData: RowData) => (
        <div className={`w-32 ${isSuspended(rowData) ? "text-placeholder" : ""}`}>{rowData.member.display_name}</div>
      ),
      thRender: () => (
        <MemberHeaderColumn
          property="display_name"
          displayFilters={filters}
          handleDisplayFilterUpdate={onFilterUpdate}
        />
      ),
    },

    {
      key: "Email address",
      content: t("workspace_settings.settings.members.details.email_address"),
      tdRender: (rowData: RowData) => (
        <div className={`w-48 truncate ${isSuspended(rowData) ? "text-placeholder" : ""}`}>{rowData.member.email}</div>
      ),
      thRender: () => (
        <MemberHeaderColumn property="email" displayFilters={filters} handleDisplayFilterUpdate={onFilterUpdate} />
      ),
    },

    {
      key: "Account type",
      content: t("workspace_settings.settings.members.details.account_type"),
      thRender: () => (
        <MemberHeaderColumn property="role" displayFilters={filters} handleDisplayFilterUpdate={onFilterUpdate} />
      ),
      tdRender: (rowData: RowData) => <AccountTypeColumn rowData={rowData} workspaceSlug={workspaceSlug} />,
    },

    {
      key: "Authentication",
      content: t("workspace_settings.settings.members.details.authentication"),
      tdRender: (rowData: RowData) => {
        if (isSuspended(rowData)) return null;
        const loginMedium = rowData.last_login_medium;
        if (!loginMedium) return null;
        return <div>{LOGIN_MEDIUM_LABELS[loginMedium]}</div>;
      },
    },

    {
      key: "Joining date",
      content: t("workspace_settings.settings.members.details.joining_date"),
      tdRender: (rowData: RowData) =>
        isSuspended(rowData) ? null : <div>{renderFormattedDate(rowData?.member?.joining_date)}</div>,
      thRender: () => (
        <MemberHeaderColumn
          property="joining_date"
          displayFilters={filters}
          handleDisplayFilterUpdate={onFilterUpdate}
        />
      ),
    },
  ];
  return { columns, workspaceSlug, removeMemberModal, setRemoveMemberModal };
};
