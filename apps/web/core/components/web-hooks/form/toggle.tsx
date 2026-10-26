/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ToggleSwitch } from "@plane/ui";

type Props = { value: boolean; onChange: (value: boolean) => void; disabled?: boolean };

export function WebhookToggle({ value, onChange, disabled }: Props) {
  return (
    <div className="flex gap-6">
      <div className="text-13 font-medium">Enable webhook</div>
      <ToggleSwitch value={value} onChange={onChange} disabled={disabled} label="Enable webhook" size="sm" />
    </div>
  );
}
