/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { stringAttribute } from "@/helpers/node-attributes";
// plane types
import type { TSearchEntities } from "@plane/types";
import { SEARCH_ENTITY_TYPES } from "@plane/types";

export enum EMentionComponentAttributeNames {
  ID = "id",
  ENTITY_IDENTIFIER = "entity_identifier",
  ENTITY_NAME = "entity_name",
}

export type TMentionComponentAttributes = {
  [EMentionComponentAttributeNames.ID]: string | null;
  [EMentionComponentAttributeNames.ENTITY_IDENTIFIER]: string | null;
  [EMentionComponentAttributeNames.ENTITY_NAME]: TSearchEntities | null;
};

export function getMentionAttributes(attributes: Record<string, unknown>): TMentionComponentAttributes {
  return {
    id: stringAttribute(attributes, "id") ?? null,
    entity_identifier: stringAttribute(attributes, "entity_identifier") ?? null,
    entity_name: SEARCH_ENTITY_TYPES.find((entity) => entity === attributes.entity_name) ?? null,
  };
}
