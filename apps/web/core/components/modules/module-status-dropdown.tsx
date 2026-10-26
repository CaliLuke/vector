/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation } from "@tanstack/react-query";
import type { IModule } from "@plane/types";
import { ModuleStatusSelect } from "./select/status";

type Props = {
  isDisabled: boolean;
  moduleDetails: IModule;
  handleModuleDetailsChange: (payload: Partial<IModule>) => Promise<void>;
};

export const ModuleStatusDropdown = function ModuleStatusDropdown({
  isDisabled,
  moduleDetails,
  handleModuleDetailsChange,
}: Props) {
  const mutation = useMutation({ mutationFn: handleModuleDetailsChange });
  return (
    <ModuleStatusSelect
      badge
      value={moduleDetails.status ?? "backlog"}
      disabled={isDisabled || mutation.isPending}
      onChange={(status) => mutation.mutate({ status })}
    />
  );
};
