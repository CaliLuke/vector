/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import type { ReactNode } from "react";

export interface INavigationPaneExtensionProps<T = Record<string, unknown>> {
  page: TPage;
  extensionData?: T;
}

export interface INavigationPaneExtensionComponent<T = Record<string, unknown>> {
  (props: INavigationPaneExtensionProps<T>): ReactNode;
}

export interface INavigationPaneExtension<T = Record<string, unknown>> {
  id: string;
  triggerParam: string;
  component: INavigationPaneExtensionComponent<T>;
  data?: T;
  width?: number;
}
