/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useEffect, useCallback } from "react";

type LocalStorageChange<T> = { type: "set"; value: T } | { type: "clear" };

function notifyLocalStorageChange<T>(key: string, persisted: boolean, change: LocalStorageChange<T>) {
  if (typeof window === "undefined") return;
  const name = `local-storage:${key}`;
  window.dispatchEvent(persisted ? new Event(name) : new window.CustomEvent(name, { detail: change }));
}

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

  const updateStoredValue = useCallback(
    (value: T | null) => setStored((current) => (current.key === key ? { key, value } : current)),
    [key]
  );

  const setValue = useCallback(
    (value: T) => {
      const persisted = setValueIntoLocalStorage(key, value);
      updateStoredValue(value);
      notifyLocalStorageChange(key, persisted, { type: "set", value });
    },
    [key, updateStoredValue]
  );

  const clearValue = useCallback(() => {
    let persisted = false;
    if (typeof window !== "undefined") {
      try {
        window.localStorage.removeItem(key);
        persisted = true;
      } catch (_error) {
        // Storage can be unavailable in restricted browser contexts.
      }
    }
    notifyLocalStorageChange(key, persisted, { type: "clear" });
    updateStoredValue(null);
  }, [key, updateStoredValue]);

  const reHydrate = useCallback(
    (event: Event | CustomEvent<LocalStorageChange<T>>) => {
      const change = "detail" in event ? event.detail : undefined;
      const data =
        change?.type === "set"
          ? change.value
          : change?.type === "clear"
            ? initialValue
            : getValueFromLocalStorage(key, initialValue);
      updateStoredValue(data);
    },
    [key, initialValue, updateStoredValue]
  );

  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    window.addEventListener(`local-storage:${key}`, reHydrate);
    return () => {
      window.removeEventListener(`local-storage:${key}`, reHydrate);
    };
  }, [key, reHydrate]);

  return { storedValue: stored.value, setValue, clearValue } as const;
};
