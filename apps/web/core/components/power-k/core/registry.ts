/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPowerKCommandConfig, TPowerKContext, TPowerKCommandGroup } from "./types";

export interface IPowerKCommandRegistry {
  // Registered commands
  readonly commands: ReadonlyMap<string, TPowerKCommandConfig>;
  // Retrieval
  getCommand(id: string): TPowerKCommandConfig | undefined;
  getAllCommands(): TPowerKCommandConfig[];
  getAllCommandsWithShortcuts(): TPowerKCommandConfig[];
  getVisibleCommands(ctx: TPowerKContext): TPowerKCommandConfig[];
  getCommandsByGroup(group: TPowerKCommandGroup, ctx: TPowerKContext): TPowerKCommandConfig[];
  // Shortcut lookup
  getShortcutMap: (ctx: TPowerKContext) => Map<string, string>; // key -> command id
  getKeySequenceMap: (ctx: TPowerKContext) => Map<string, string>; // sequence -> command id
  getModifierShortcutMap: (ctx: TPowerKContext) => Map<string, string>; // modifier shortcut -> command id
  findByShortcut(ctx: TPowerKContext, key: string): TPowerKCommandConfig | undefined;
  findByKeySequence(ctx: TPowerKContext, sequence: string): TPowerKCommandConfig | undefined;
  findByModifierShortcut(ctx: TPowerKContext, shortcut: string): TPowerKCommandConfig | undefined;
}

/**
 * Stores commands and provides lookup by shortcuts, search, etc.
 */
export class PowerKCommandRegistry implements IPowerKCommandRegistry {
  readonly commands: ReadonlyMap<string, TPowerKCommandConfig>;

  constructor(commands: readonly TPowerKCommandConfig[] = []) {
    this.commands = new Map(commands.map((command) => [command.id, command]));
  }

  // ============================================================================
  // Retrieval
  // ============================================================================

  getCommand: IPowerKCommandRegistry["getCommand"] = (id) => this.commands.get(id);

  getAllCommands: IPowerKCommandRegistry["getAllCommands"] = () => Array.from(this.commands.values());

  getAllCommandsWithShortcuts: IPowerKCommandRegistry["getAllCommandsWithShortcuts"] = () =>
    Array.from(this.commands.values()).filter(
      (command) => command.shortcut || command.keySequence || command.modifierShortcut
    );

  getVisibleCommands: IPowerKCommandRegistry["getVisibleCommands"] = (ctx) =>
    Array.from(this.commands.values()).filter((command) => this.isCommandVisible(command, ctx));

  getCommandsByGroup: IPowerKCommandRegistry["getCommandsByGroup"] = (group, ctx) =>
    this.getVisibleCommands(ctx).filter((command) => command.group === group);

  // ============================================================================
  // Shortcut Lookup
  // ============================================================================

  getShortcutMap: IPowerKCommandRegistry["getShortcutMap"] = (ctx) => {
    const shortcutMap = new Map<string, string>();
    this.getVisibleCommands(ctx).forEach((command) => {
      if (command.shortcut) {
        shortcutMap.set(command.shortcut.toLowerCase(), command.id);
      }
    });
    return shortcutMap;
  };

  getKeySequenceMap: IPowerKCommandRegistry["getKeySequenceMap"] = (ctx) => {
    const keySequenceMap = new Map<string, string>();
    this.getVisibleCommands(ctx).forEach((command) => {
      if (command.keySequence) {
        keySequenceMap.set(command.keySequence.toLowerCase(), command.id);
      }
    });
    return keySequenceMap;
  };

  getModifierShortcutMap: IPowerKCommandRegistry["getModifierShortcutMap"] = (ctx) => {
    const modifierShortcutMap = new Map<string, string>();
    this.getVisibleCommands(ctx).forEach((command) => {
      if (command.modifierShortcut) {
        modifierShortcutMap.set(command.modifierShortcut.toLowerCase(), command.id);
      }
    });
    return modifierShortcutMap;
  };

  findByShortcut: IPowerKCommandRegistry["findByShortcut"] = (ctx, key) => {
    const commandId = this.getShortcutMap(ctx).get(key.toLowerCase());
    return commandId ? this.commands.get(commandId) : undefined;
  };

  findByKeySequence: IPowerKCommandRegistry["findByKeySequence"] = (ctx, sequence) => {
    const commandId = this.getKeySequenceMap(ctx).get(sequence.toLowerCase());
    return commandId ? this.commands.get(commandId) : undefined;
  };

  findByModifierShortcut: IPowerKCommandRegistry["findByModifierShortcut"] = (ctx, shortcut) => {
    const commandId = this.getModifierShortcutMap(ctx).get(shortcut.toLowerCase());
    return commandId ? this.commands.get(commandId) : undefined;
  };

  // ============================================================================
  // Private Helpers
  // ============================================================================

  private isCommandVisible(command: TPowerKCommandConfig, ctx: TPowerKContext): boolean {
    // Check custom visibility function
    if (command.isVisible && !command.isVisible(ctx)) {
      return false;
    }

    // Check context type filtering
    if ("contextType" in command) {
      // Command requires specific context
      if (!ctx.activeContext || ctx.activeContext !== command.contextType) {
        return false;
      }

      if (!ctx.shouldShowContextBasedActions) {
        return false;
      }
    }

    return true;
  }
}
