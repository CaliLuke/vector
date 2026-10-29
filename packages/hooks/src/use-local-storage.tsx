/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useEffect, useCallback } from "react";

const getValueFromLocalStorage = <T,>(key: string, defaultValue: T): T => {
  if (typeof window === "undefined") return defaultValue;
  try {
    const item = window.localStorage.getItem(key);
    return item ? JSON.parse(item) : defaultValue;
  } catch (_error) {
    try {
      window.localStorage.removeItem(key);
    } catch (_removeError) {
      // Storage can be unavailable in restricted browser contexts.
    }
    return defaultValue;
  }
};

const setValueIntoLocalStorage = (key: string, value: unknown) => {
  if (typeof window === "undefined") return false;
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
    return true;
  } catch (_error) {
    return false;
  }
};

export const useLocalStorage = <T,>(key: string, initialValue: T) => {
  const [stored, setStored] = useState<{ key: string; value: T | null }>(() => ({
    key,
    value: getValueFromLocalStorage(key, initialValue),
  }));

  // Reset before React commits a render for a different preference scope.
  if (stored.key !== key) {
    setStored({ key, value: getValueFromLocalStorage(key, initialValue) });
  }

  const setValue = useCallback(
    (value: T) => {
      setValueIntoLocalStorage(key, value);
      setStored({ key, value });
      if (typeof window !== "undefined") window.dispatchEvent(new Event(`local-storage:${key}`));
    },
    [key]
  );

  const clearValue = useCallback(() => {
    if (typeof window !== "undefined") {
      try {
        window.localStorage.removeItem(key);
      } catch (_error) {
        // Storage can be unavailable in restricted browser contexts.
      }
      window.dispatchEvent(new Event(`local-storage:${key}`));
    }
    setStored({ key, value: null });
  }, [key]);

  const reHydrate = useCallback(() => {
    const data = getValueFromLocalStorage(key, initialValue);
    setStored({ key, value: data });
  }, [key, initialValue]);

  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    window.addEventListener(`local-storage:${key}`, reHydrate);
    return () => {
      window.removeEventListener(`local-storage:${key}`, reHydrate);
    };
  }, [key, reHydrate]);

  return { storedValue: stored.value, setValue, clearValue } as const;
};
