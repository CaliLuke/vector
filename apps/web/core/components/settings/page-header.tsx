/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Breadcrumbs, Header } from "@plane/ui";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";

type Props = {
  leftItem?: React.ReactNode;
  rightItem?: React.ReactNode;
};

export function SettingsPageHeader(props: Props) {
  const { leftItem, rightItem } = props;

  return (
    <Header>
      {leftItem && <Header.LeftItem>{leftItem}</Header.LeftItem>}
      {rightItem && <Header.RightItem>{rightItem}</Header.RightItem>}
    </Header>
  );
}

export function SettingsBreadcrumbHeader({ label, icon }: { label: string; icon: React.ReactNode }) {
  return (
    <SettingsPageHeader
      leftItem={
        <div className="flex items-center gap-2">
          <Breadcrumbs>
            <Breadcrumbs.Item component={<BreadcrumbLink label={label} icon={icon} />} />
          </Breadcrumbs>
        </div>
      }
    />
  );
}
