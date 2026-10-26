/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { useTranslation } from "@plane/i18n";
import { StateGroupIcon } from "@plane/propel/icons";
import { Loader } from "@plane/ui";
import { cn } from "@plane/utils/common";
// hooks
import { useProjectStates } from "@/hooks/use-project-states";

export type TReadonlyStateProps = {
  className?: string;
  iconSize?: string;
  hideIcon?: boolean;
  value: string | undefined | null;
  placeholder?: string;
  projectId: string | undefined;
  workspaceSlug: string;
};

export function ReadonlyState(props: TReadonlyStateProps) {
  const { className, iconSize = "size-4", hideIcon = false, value, placeholder, projectId, workspaceSlug } = props;
  const { t } = useTranslation();
  const stateQuery = useProjectStates(workspaceSlug, projectId);
  // derived values
  const state = stateQuery.getStateById(value);

  if (stateQuery.isPending) {
    return (
      <Loader className={cn("flex items-center gap-1 text-body-xs-regular", className)}>
        <Loader.Item height="16px" width="16px" className="rounded-full" />
        <Loader.Item height="16px" width="50px" />
      </Loader>
    );
  }

  return (
    <div className={cn("flex items-center gap-1 text-body-xs-regular", className)}>
      {!hideIcon && (
        <StateGroupIcon
          stateGroup={state?.group ?? "backlog"}
          className={cn(iconSize, "flex-shrink-0")}
          color={state?.color}
        />
      )}
      <span className="flex-grow truncate">{state?.name ?? placeholder ?? t("common.none")}</span>
    </div>
  );
}
