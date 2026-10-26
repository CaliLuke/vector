/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Command } from "cmdk";

// plane imports
import { THEME_OPTIONS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// local imports
import { PowerKModalCommandItem } from "../../modal/command-item";

type Props = {
  onSelect: (theme: string) => void;
};

export const PowerKPreferencesThemesMenu = function PowerKPreferencesThemesMenu(props: Props) {
  const { onSelect } = props;
  // hooks
  const { t } = useTranslation();
  return (
    <Command.Group>
      {THEME_OPTIONS.map((theme) => (
        <PowerKModalCommandItem key={theme.value} onSelect={() => onSelect(theme.value)} label={t(theme.i18n_label)} />
      ))}
    </Command.Group>
  );
};
