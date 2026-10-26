/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { useTranslation } from "@plane/i18n";
import type { IUserLite, TDescriptionVersion } from "@plane/types";
import { Avatar, CustomMenu } from "@plane/ui";
import { calculateTimeAgo } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";

type Props = {
  onClick: (versionId: string) => void;
  memberDetails: IUserLite | undefined;
  version: TDescriptionVersion;
};

export function DescriptionVersionsDropdownItem(props: Props) {
  const { memberDetails, onClick, version } = props;
  // translation
  const { t } = useTranslation();

  return (
    <CustomMenu.MenuItem key={version.id} className="flex items-center gap-1" onClick={() => onClick(version.id)}>
      <span className="flex-shrink-0">
        <Avatar
          name={memberDetails?.display_name ?? t("common.deactivated_user")}
          size="sm"
          src={getFileURL(memberDetails?.avatar_url ?? "")}
        />
      </span>
      <p className="flex items-center gap-1.5 text-11 text-secondary">
        <span className="font-medium">{memberDetails?.display_name ?? t("common.deactivated_user")}</span>
        <span>{calculateTimeAgo(version.last_saved_at)}</span>
      </p>
    </CustomMenu.MenuItem>
  );
}
