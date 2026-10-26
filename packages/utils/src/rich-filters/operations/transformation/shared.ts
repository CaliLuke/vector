/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TFilterGroupNode, TFilterProperty } from "@plane/types";
import { processGroupNode } from "../../types/shared";
import type { TTreeTransformFn, TTreeTransformResult } from "./core";
import { transformExpressionTree, transformGroupWithChildren } from "./core";

/**
 * Transforms groups by processing children.
 * Handles AND/OR groups with children and NOT groups with single child.
 * @param group - The group to transform
 * @param transformFn - The transformation function
 * @returns The transformation result
 */
export const transformGroup = <P extends TFilterProperty>(
  group: TFilterGroupNode<P>,
  transformFn: TTreeTransformFn<P>
): TTreeTransformResult<P> =>
  processGroupNode(group, {
    onOrGroup: (orGroup) => transformGroupWithChildren(orGroup, transformFn),
    onAndGroup: (andGroup) => transformGroupWithChildren(andGroup, transformFn),
    onNotGroup: (notGroup) => {
      const result = transformExpressionTree(notGroup.child, transformFn);
      return { ...result, expression: result.expression ? { ...notGroup, child: result.expression } : null };
    },
  });
