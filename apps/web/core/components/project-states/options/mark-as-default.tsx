/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

// plane imports
import type { TStateOperationsCallbacks } from "@plane/types";
import { cn } from "@plane/utils/common";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { useCommittedScope } from "@/hooks/use-committed-scope";

type TStateMarksAsDefault = {
  stateId: string;
  isDefault: boolean;
  markStateAsDefaultCallback: TStateOperationsCallbacks["markStateAsDefault"];
};

export function StateMarksAsDefault(props: TStateMarksAsDefault) {
  return <DefaultStateAction key={props.stateId} {...props} />;
}

const DefaultStateAction = function DefaultStateAction(props: TStateMarksAsDefault) {
  const { stateId, isDefault, markStateAsDefaultCallback } = props;
  // states
  const [isLoading, setIsLoading] = useState(false);
  const isCurrent = useCommittedScope(stateId);

  const handleMarkAsDefault = async () => {
    if (!stateId || isDefault || isLoading) return;
    setIsLoading(true);

    try {
      await markStateAsDefaultCallback(stateId);
    } catch {
      if (isCurrent())
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not change the default state", message: "Please try again." });
    } finally {
      if (isCurrent()) setIsLoading(false);
    }
  };

  return (
    <button
      className={cn(
        "text-11 whitespace-nowrap transition-colors",
        isDefault ? "text-tertiary" : "text-secondary hover:text-primary"
      )}
      disabled={isDefault || isLoading}
      onClick={() => void handleMarkAsDefault()}
    >
      {isLoading ? "Marking as default" : isDefault ? `Default` : `Mark as default`}
    </button>
  );
};
