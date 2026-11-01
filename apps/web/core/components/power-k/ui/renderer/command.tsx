/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Command } from "cmdk";
// plane imports
import { useTranslation } from "@plane/i18n";
// local imports
import type { TPowerKCommandConfig, TPowerKCommandGroup, TPowerKContext } from "../../core/types";
import { PowerKModalCommandItem } from "../modal/command-item";
import { CONTEXT_ENTITY_MAP } from "../pages/context-based";
import { POWER_K_GROUP_PRIORITY, POWER_K_GROUP_I18N_TITLES } from "./shared";

type Props = {
  commands: TPowerKCommandConfig[];
  context: TPowerKContext;
  onCommandSelect: (command: TPowerKCommandConfig) => void;
};

export function CommandRenderer(props: Props) {
  const { commands, context, onCommandSelect } = props;
  // derived values
  const { activeContext } = context;
  // translation
  const { t } = useTranslation();

  const commandsByGroup = new Map<TPowerKCommandGroup, TPowerKCommandConfig[]>();
  for (const command of commands) {
    const group = commandsByGroup.get(command.group);
    if (group) group.push(command);
    else commandsByGroup.set(command.group, [command]);
  }
  const sortedGroups = [...commandsByGroup].toSorted(
    ([a], [b]) => POWER_K_GROUP_PRIORITY[a] - POWER_K_GROUP_PRIORITY[b]
  );

  return (
    <>
      {sortedGroups.map(([groupKey, groupCommands]) => {
        if (!groupCommands || groupCommands.length === 0) return null;

        const title =
          groupKey === "contextual" && activeContext
            ? t(CONTEXT_ENTITY_MAP[activeContext].i18n_title)
            : t(POWER_K_GROUP_I18N_TITLES[groupKey]);

        return (
          <Command.Group key={groupKey} heading={title}>
            {groupCommands.map((command) => (
              <PowerKModalCommandItem
                key={command.id}
                icon={command.icon}
                iconNode={command.iconNode}
                label={t(command.i18n_title)}
                keySequence={command.keySequence}
                shortcut={command.shortcut || command.modifierShortcut}
                onSelect={() => onCommandSelect(command)}
              />
            ))}
          </Command.Group>
        );
      })}
    </>
  );
}
