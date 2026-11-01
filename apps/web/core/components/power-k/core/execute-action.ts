import { copyTextToClipboard } from "@plane/utils/string";
import type { TPowerKCommandConfig, TPowerKContext } from "./types";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";

function reportCommandFailure() {
  setToast({
    type: TOAST_TYPE.ERROR,
    title: "Could not complete command",
    message: "Please try again.",
  });
}

// Commands start synchronously so navigation and palette dismissal retain their
// ordering. The palette owns failures that an individual command does not handle.
export function executePowerKAction(action: () => void | Promise<void>): void {
  try {
    void Promise.resolve(action()).catch(reportCommandFailure);
  } catch {
    reportCommandFailure();
  }
}

export function selectPowerKCommand(
  command: TPowerKCommandConfig,
  context: TPowerKContext,
  resetSearch: () => void,
  alwaysClose = false
): void {
  if (command.type === "action") {
    executePowerKAction(() => command.action(context));
    if (alwaysClose || command.closeOnSelect) context.closePalette();
  } else if (command.type === "change-page") {
    context.setActiveCommand(command);
    context.setActivePage(command.page);
    resetSearch();
  }
}

export function selectPowerKPageData(data: unknown, context: TPowerKContext, alwaysClose = false): void {
  const command = context.activeCommand;
  if (command?.type === "change-page") {
    executePowerKAction(() => command.onSelect(data, context));
  }
  if (alwaysClose || command?.closeOnSelect === true) context.closePalette();
}

export async function copyPowerKCurrentUrl(successTitle: string, errorTitle: string): Promise<void> {
  try {
    await copyTextToClipboard(window.location.href);
    setToast({ type: TOAST_TYPE.SUCCESS, title: successTitle });
  } catch {
    setToast({ type: TOAST_TYPE.ERROR, title: errorTitle });
  }
}
