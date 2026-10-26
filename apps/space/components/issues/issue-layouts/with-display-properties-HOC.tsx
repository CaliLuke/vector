/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { IIssueDisplayProperties } from "@plane/types";

interface IWithDisplayPropertiesHOC {
  displayProperties: IIssueDisplayProperties;
  shouldRenderProperty?: (displayProperties: IIssueDisplayProperties) => boolean;
  displayPropertyKey: keyof IIssueDisplayProperties | (keyof IIssueDisplayProperties)[];
  children: React.ReactNode;
}

export const WithDisplayPropertiesHOC = function WithDisplayPropertiesHOC({
  displayProperties,
  shouldRenderProperty,
  displayPropertyKey,
  children,
}: IWithDisplayPropertiesHOC) {
  const keys = Array.isArray(displayPropertyKey) ? displayPropertyKey : [displayPropertyKey];
  if (!keys.every((key) => displayProperties[key])) return null;
  if (shouldRenderProperty && !shouldRenderProperty(displayProperties)) return null;

  return <>{children}</>;
};
