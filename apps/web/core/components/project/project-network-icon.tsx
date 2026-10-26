/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TNetworkChoiceIconKey } from "@plane/constants";
// plane imports
import { GlobeIcon, LockIcon } from "@plane/propel/icons";
import { cn } from "@plane/utils/common";

type Props = {
  iconKey: TNetworkChoiceIconKey;
  className?: string;
};

export function ProjectNetworkIcon(props: Props) {
  const { iconKey, className } = props;
  const iconClassName = cn("h-3 w-3", className);
  switch (iconKey) {
    case "Lock":
      return <LockIcon className={iconClassName} />;
    case "Globe2":
      return <GlobeIcon className={iconClassName} />;
    default:
      return null;
  }
}
