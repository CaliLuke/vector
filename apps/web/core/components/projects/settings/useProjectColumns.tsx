/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { renderFormattedDate } from "@plane/utils/datetime";
// components
import { MemberHeaderColumn } from "@/components/project/member-header-column";
import { AccountTypeColumn, NameColumn } from "@/components/project/settings/member-columns";
import type { RowData } from "@/components/project/settings/member-columns";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import type { IMemberFilters } from "@/helpers/member-ordering";

type TUseProjectColumnsProps = {
  projectId: string;
  workspaceSlug: string;
  displayFilters: IMemberFilters;
  handleDisplayFilterUpdate: (filters: Partial<IMemberFilters>) => void;
};

export const useProjectColumns = (props: TUseProjectColumnsProps) => {
  const { projectId, workspaceSlug, displayFilters, handleDisplayFilterUpdate } = props;
  const [removeMemberModal, setRemoveMemberModal] = useState<RowData | null>(null);

  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const isAdmin = canProject([EUserPermissions.ADMIN]);

  const columns = [
    {
      key: "Full Name",
      content: "Full name",
      thClassName: "text-left",
      thRender: () => (
        <MemberHeaderColumn
          property="full_name"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
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
      key: "Display Name",
      content: "Display name",
      thRender: () => (
        <MemberHeaderColumn
          property="display_name"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: RowData) => <div className="w-32">{rowData.member.display_name}</div>,
    },
    {
      key: "Email",
      content: "Email",
      thRender: () => (
        <MemberHeaderColumn
          property="email"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: RowData) => <div className="w-48 text-secondary">{rowData.member.email}</div>,
    },
    {
      key: "Account Type",
      content: "Account type",
      thRender: () => (
        <MemberHeaderColumn
          property="role"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: RowData) => (
        <AccountTypeColumn rowData={rowData} projectId={projectId} workspaceSlug={workspaceSlug} />
      ),
    },
    {
      key: "Joining Date",
      content: "Joining date",
      thRender: () => (
        <MemberHeaderColumn
          property="joining_date"
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFilterUpdate}
        />
      ),
      tdRender: (rowData: RowData) => <div>{renderFormattedDate(rowData?.member?.joining_date)}</div>,
    },
  ];
  return {
    columns,
    removeMemberModal,
    setRemoveMemberModal,
    displayFilters,
    handleDisplayFilterUpdate,
  };
};
