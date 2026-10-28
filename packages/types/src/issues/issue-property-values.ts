/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type TIssuePropertyValues = Record<string, unknown>;
export type TIssuePropertyValueErrors = Record<string, unknown>;

/** Authorized effective values returned only when a list requests projections. */
export type TWorkItemPropertyKind =
  | "Text"
  | "Email"
  | "File"
  | "Number"
  | "Dropdown"
  | "Boolean"
  | "Date"
  | "Member picker"
  | "Work item picker"
  | "Release picker"
  | "Rich text"
  | "URL"
  | "Formula"
  | "Cascading";
export type TWorkItemProjectedValue = {
  kind: TWorkItemPropertyKind;
  result_kind?: "Text" | "Number" | "Boolean" | "Date";
  context_id: string;
  catalog_id: string;
  status: "not_applicable" | "empty" | "value" | "error";
  value: string | number | boolean | string[] | null;
  error: string;
  levels?: Record<string, string | null>;
  format?: string;
};
