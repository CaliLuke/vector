/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useEffectEvent } from "react";
import debounce from "lodash-es/debounce.js";

const AUTO_SAVE_TIME = 30000;

const useAutoSave = (handleSaveDescription: () => void | Promise<void>) => {
  const save = useEffectEvent((reason: string) => {
    const reportFailure = (error: unknown) => console.error(`${reason} failed:`, error);
    try {
      void Promise.resolve(handleSaveDescription()).catch(reportFailure);
    } catch (error) {
      reportFailure(error);
    }
  });

  useEffect(() => {
    const scheduleAutosave = () => setInterval(() => save("Autosave"), AUTO_SAVE_TIME);
    let interval = scheduleAutosave();
    const debouncedSave = debounce(() => {
      save("Manual save");
      clearInterval(interval);
      interval = scheduleAutosave();
    }, 500);
    const handleSave = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        event.stopPropagation();
        debouncedSave();
      }
    };
    window.addEventListener("keydown", handleSave);
    return () => {
      window.removeEventListener("keydown", handleSave);
      debouncedSave.cancel();
      clearInterval(interval);
    };
  }, []);
};

export default useAutoSave;
