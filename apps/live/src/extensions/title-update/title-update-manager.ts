/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { logger } from "@plane/logger";
import { AppError } from "@/lib/errors";
import { getPageService } from "@/services/page/handler";
import type { HocusPocusServerContext } from "@/types";
import { DebounceManager } from "./debounce";

/**
 * Manages title update operations for a single document
 * Handles debouncing, aborting, and force saving title updates
 */
export class TitleUpdateManager {
  private documentName: string;
  private context: HocusPocusServerContext;
  private debounceManager: DebounceManager;
  private lastTitle: string | null = null;
  private lastKnownTitle: string | null;
  private inFlightTitle: string | null = null;
  private titleAfterInFlight: string | null = null;
  private inFlightCompletion: Promise<void> | null = null;
  private resolveInFlightCompletion: (() => void) | null = null;

  /**
   * Create a new TitleUpdateManager instance
   */
  constructor(
    documentName: string,
    context: HocusPocusServerContext,
    wait: number = 5000,
    loadedTitle: string | null = null
  ) {
    this.documentName = documentName;
    this.context = context;
    this.lastKnownTitle = loadedTitle;

    // Set up debounce manager with logging
    this.debounceManager = new DebounceManager({
      wait,
      logPrefix: `TitleManager[${documentName.substring(0, 8)}]`,
    });
  }

  /**
   * Schedule a debounced title update
   */
  scheduleUpdate(title: string): void {
    if (this.inFlightTitle !== null) {
      // Keep only the latest intent until the active PATCH settles. A later
      // write is needed when the user reverts while a different title is in flight.
      this.lastTitle = title;
      this.titleAfterInFlight = title;
      return;
    }

    if (title === this.lastKnownTitle) {
      this.debounceManager.cancel();
      this.lastTitle = null;
      return;
    }

    // Store the latest title
    this.lastTitle = title;

    // Schedule the update with the debounce manager
    this.debounceManager.schedule(this.updateTitle.bind(this), title);
  }

  /**
   * Update the title - will be called by the debounce manager
   */
  private async updateTitle(title: string, signal?: AbortSignal): Promise<void> {
    const service = getPageService(this.context.documentType, this.context);
    if (!service.updatePageProperties) {
      logger.warn(`No updateTitle method found for document ${this.documentName}`);
      return;
    }

    try {
      this.inFlightTitle = title;
      this.inFlightCompletion = new Promise<void>((resolve) => {
        this.resolveInFlightCompletion = resolve;
      });
      await service.updatePageProperties(this.documentName, {
        data: { name: title },
        abortSignal: signal,
      });
      this.lastKnownTitle = title;

      // Clear last title only if it matches what we just updated
      if (this.lastTitle === title) {
        this.lastTitle = null;
      }
    } catch (error) {
      const appError = new AppError(error, {
        context: { operation: "updateTitle", documentName: this.documentName },
      });
      logger.error("Error updating title", appError);
    } finally {
      if (this.inFlightTitle === title) {
        this.inFlightTitle = null;
      }
      const titleAfterInFlight = this.titleAfterInFlight;
      this.titleAfterInFlight = null;
      if (titleAfterInFlight !== null) {
        if (titleAfterInFlight === this.lastKnownTitle) {
          this.lastTitle = null;
          this.debounceManager.cancel();
        } else {
          this.lastTitle = titleAfterInFlight;
          this.debounceManager.schedule(this.updateTitle.bind(this), titleAfterInFlight);
        }
      }
      this.resolveInFlightCompletion?.();
      this.resolveInFlightCompletion = null;
      this.inFlightCompletion = null;
    }
  }

  /**
   * Force save the current title immediately
   */
  async forceSave(): Promise<void> {
    if (this.inFlightCompletion) {
      await this.inFlightCompletion;
    }

    // Ensure we have the current title
    if (this.lastTitle === null) {
      return;
    }

    // Use the debounce manager to flush the operation
    await this.debounceManager.flush(this.updateTitle.bind(this));
  }

  /**
   * Cancel any pending updates
   */
  cancel(): void {
    this.debounceManager.cancel();
    this.lastTitle = null;
  }
}
