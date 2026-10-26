/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useEffect, useRef } from "react";

// hooks
import { usePowerK } from "@/hooks/use-power-k";
import { usePowerKCommandRegistry } from "@/hooks/use-power-k-command-registry";
// local imports
import { detectContextFromURL } from "./core/context-detector";
import { ShortcutHandler } from "./core/shortcut-handler";
import type { TPowerKCommandConfig, TPowerKContext } from "./core/types";
import { ShortcutsModal } from "./ui/modal/shortcuts-root";

type GlobalShortcutsProps = {
  context: TPowerKContext;
  commands: TPowerKCommandConfig[];
};

/**
 * Global shortcuts component - sets up keyboard listeners and context detection
 * Should be mounted once at the app root level
 */
export function GlobalShortcutsProvider(props: GlobalShortcutsProps) {
  const { context, commands } = props;
  // router
  const params = useParams({ strict: false });
  // store hooks
  const { isShortcutsListModalOpen, setActiveContext, togglePowerKModal, toggleShortcutsListModal } = usePowerK();
  const { commandRegistry, registerCommands } = usePowerKCommandRegistry();

  // Detect context from URL and update store
  useEffect(() => {
    const detected = detectContextFromURL(params);
    setActiveContext(detected);
  }, [params, setActiveContext]);

  // Register commands on mount
  useEffect(() => registerCommands(commands), [commands, registerCommands]);

  // Store context in ref to avoid recreation on context changes
  const contextRef = useRef(context);
  useEffect(() => {
    contextRef.current = context;
  }, [context]);

  // Store handler in ref to avoid recreation on context changes
  const handlerRef = useRef<ShortcutHandler | null>(null);

  // Setup global shortcut handler - only recreate when commandRegistry or togglePowerKModal changes
  useEffect(() => {
    // Clean up previous handler if it exists
    if (handlerRef.current) {
      document.removeEventListener("keydown", handlerRef.current.handleKeyDown);
      handlerRef.current.destroy();
    }

    // Create new handler with function that reads from ref
    handlerRef.current = new ShortcutHandler(
      commandRegistry,
      () => contextRef.current,
      () => togglePowerKModal(true)
    );

    document.addEventListener("keydown", handlerRef.current.handleKeyDown);

    return () => {
      if (handlerRef.current) {
        document.removeEventListener("keydown", handlerRef.current.handleKeyDown);
        handlerRef.current.destroy();
        handlerRef.current = null;
      }
    };
  }, [commandRegistry, togglePowerKModal]);

  return <ShortcutsModal isOpen={isShortcutsListModalOpen} onClose={() => toggleShortcutsListModal(false)} />;
}
