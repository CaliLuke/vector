/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ModuleLinkOperations } from "./link-operations";

// Preserve the compact link-service API while sharing the transport and error contracts.
export class ModuleLinkService extends ModuleLinkOperations {
  create(...args: Parameters<ModuleLinkOperations["createModuleLink"]>) {
    return this.createModuleLink(...args);
  }

  update(...args: Parameters<ModuleLinkOperations["updateModuleLink"]>) {
    return this.updateModuleLink(...args);
  }

  destroy(...args: Parameters<ModuleLinkOperations["deleteModuleLink"]>) {
    return this.deleteModuleLink(...args);
  }
}
