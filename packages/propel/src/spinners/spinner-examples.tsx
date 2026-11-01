/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";

const colors = [
  ["Blue", "text-blue-500"],
  ["Green", "text-success-primary"],
  ["Red", "text-danger-primary"],
  ["Purple", "text-purple-500"],
  ["Orange", "text-orange-500"],
] as const;

export function SpinnerColorExamples({ renderSpinner }: { renderSpinner: (className: string) => ReactNode }) {
  return (
    <div className="flex items-center gap-6">
      {colors.map(([label, className]) => (
        <div key={label} className="text-center">
          {renderSpinner(className)}
          <p className="text-gray-600 mt-2 text-11">{label}</p>
        </div>
      ))}
    </div>
  );
}
