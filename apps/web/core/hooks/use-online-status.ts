/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useSyncExternalStore } from "react";

function subscribeToConnection(notify: () => void) {
  window.addEventListener("online", notify);
  window.addEventListener("offline", notify);
  return () => {
    window.removeEventListener("online", notify);
    window.removeEventListener("offline", notify);
  };
}

const getOnlineStatus = () => navigator.onLine;
const getServerOnlineStatus = () => true;

const useOnlineStatus = () => ({
  isOnline: useSyncExternalStore(subscribeToConnection, getOnlineStatus, getServerOnlineStatus),
});

export default useOnlineStatus;
