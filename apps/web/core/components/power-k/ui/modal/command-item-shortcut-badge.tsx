/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

const KEY_SYMBOLS = new Map([
  ["delete", "⌫"],
  ["backspace", "⌫"],
  ["enter", "↵"],
  ["return", "↵"],
  ["space", "Space"],
  ["escape", "Esc"],
  ["esc", "Esc"],
  ["tab", "Tab"],
  ["arrowup", "↑"],
  ["up", "↑"],
  ["arrowdown", "↓"],
  ["down", "↓"],
  ["arrowleft", "←"],
  ["left", "←"],
  ["arrowright", "→"],
  ["right", "→"],
]);
const MAC_MODIFIERS = new Map([
  ["cmd", "⌘"],
  ["meta", "⌘"],
  ["ctrl", "⌃"],
  ["alt", "⌥"],
  ["option", "⌥"],
  ["shift", "⇧"],
]);
const OTHER_MODIFIERS = new Map([
  ["cmd", "Ctrl"],
  ["meta", "Ctrl"],
  ["ctrl", "Ctrl"],
  ["alt", "Alt"],
  ["option", "Alt"],
  ["shift", "Shift"],
]);

function KeyBadges({ keys, sequence = false }: { keys: string[]; sequence?: boolean }) {
  const occurrences = new Map<string, number>();
  const tokens = keys.map((label) => {
    const occurrence = (occurrences.get(label) ?? 0) + 1;
    occurrences.set(label, occurrence);
    return { id: `${label}:${occurrence}`, label };
  });
  return (
    <div className="pointer-events-none inline-flex shrink-0 items-center gap-1 font-medium select-none">
      {tokens.map((token) => (
        <React.Fragment key={token.id}>
          <kbd className="inline-flex h-5 items-center justify-center rounded-sm border border-strong bg-surface-1 px-1.5 font-code text-10 font-medium text-tertiary">
            {token.label}
          </kbd>
          {sequence && token !== tokens.at(-1) && <span className="text-10 text-placeholder">then</span>}
        </React.Fragment>
      ))}
    </div>
  );
}

export function ShortcutBadge({ shortcut }: { shortcut: string | undefined }) {
  if (!shortcut) return null;
  const isMac = typeof navigator !== "undefined" && navigator.platform.toUpperCase().includes("MAC");
  const modifiers = isMac ? MAC_MODIFIERS : OTHER_MODIFIERS;
  const keys = shortcut.split("+").map((part) => {
    const key = part.toLowerCase().trim();
    return modifiers.get(key) ?? KEY_SYMBOLS.get(key) ?? part.trim().toUpperCase();
  });
  return <KeyBadges keys={keys} />;
}

export function KeySequenceBadge({ sequence }: { sequence: string | undefined }) {
  if (!sequence) return null;
  return <KeyBadges keys={sequence.toUpperCase().split("")} sequence />;
}
