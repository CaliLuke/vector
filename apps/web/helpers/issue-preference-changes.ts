/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { ISSUE_DISPLAY_PROPERTIES_KEYS } from "@plane/constants";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties, TSupportedFilterForUpdate } from "@plane/types";
const propertyKeys = new Set<string>(ISSUE_DISPLAY_PROPERTIES_KEYS);
const filterKeys = new Set([
  "calendar",
  "group_by",
  "sub_group_by",
  "layout",
  "order_by",
  "show_empty_groups",
  "sub_issue",
]);
export function isDisplayFilterChange(value: TSupportedFilterForUpdate): value is IIssueDisplayFilterOptions {
  return (
    Object.keys(value).every((key) => filterKeys.has(key)) &&
    !("group_by" in value && Array.isArray(value.group_by)) &&
    !("sub_group_by" in value && Array.isArray(value.sub_group_by))
  );
}
export function isDisplayPropertyChange(value: TSupportedFilterForUpdate): value is IIssueDisplayProperties {
  return Object.entries(value).every(
    ([key, enabled]) =>
      (propertyKeys.has(key) || /^customproperty_[a-zA-Z0-9_-]+$/.test(key)) &&
      (enabled === undefined || typeof enabled === "boolean")
  );
}
