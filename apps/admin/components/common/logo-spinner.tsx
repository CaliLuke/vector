/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import logoSrc from "@/app/assets/vector.svg?url";

export function LogoSpinner() {
  return (
    <div className="flex items-center justify-center">
      <img src={logoSrc} alt="Vector" className="h-6 w-auto sm:h-11" />
    </div>
  );
}
