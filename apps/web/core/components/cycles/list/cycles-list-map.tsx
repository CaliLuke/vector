/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ICycle } from "@plane/types";
// components
import { CyclesListItem } from "./cycles-list-item";

type Props = {
  cycles: ICycle[];
  projectId: string;
  workspaceSlug: string;
};

export function CyclesListMap(props: Props) {
  const { cycles, projectId, workspaceSlug } = props;

  return (
    <>
      {cycles.map((cycle) => (
        <CyclesListItem key={cycle.id} cycle={cycle} workspaceSlug={workspaceSlug} projectId={projectId} />
      ))}
    </>
  );
}
