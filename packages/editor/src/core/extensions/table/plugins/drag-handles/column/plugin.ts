/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { createTableDragHandlePlugin } from "../plugin";

export const TableColumnDragHandlePlugin = (editor: Editor) => createTableDragHandlePlugin(editor, "column");
