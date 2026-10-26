/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Command } from "cmdk";

// hooks
import useTimezone from "@/hooks/use-timezone";
// local imports
import { PowerKModalCommandItem } from "../../modal/command-item";

type Props = {
  onSelect: (timezone: string) => void;
};

export const PowerKPreferencesTimezonesMenu = function PowerKPreferencesTimezonesMenu(props: Props) {
  const { onSelect } = props;
  // timezones
  const { timezones } = useTimezone();

  return (
    <Command.Group>
      {timezones.map((timezone) => (
        <PowerKModalCommandItem
          key={timezone.value}
          onSelect={() => onSelect(timezone.value)}
          label={timezone.content}
        />
      ))}
    </Command.Group>
  );
};
