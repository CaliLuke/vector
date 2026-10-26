/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

type Props = { isChecked: boolean; disabled?: boolean; handleChange: (checked: boolean) => void };
export function MarketingConsent({ isChecked, disabled, handleChange }: Props) {
  return (
    <label className="flex items-center justify-center gap-1.5 text-13 text-tertiary">
      <input
        type="checkbox"
        checked={isChecked}
        disabled={disabled}
        onChange={(event) => handleChange(event.target.checked)}
        className="accent-accent-primary size-4"
      />
      I agree to Vector marketing communications
    </label>
  );
}
