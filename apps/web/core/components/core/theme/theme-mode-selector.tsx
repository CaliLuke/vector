/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

type Props = { value: boolean; onChange: (value: boolean) => void; disabled?: boolean };

export function CustomThemeModeSelector({ value, onChange, disabled }: Props) {
  return (
    <div>
      <h6 className="text-h6-medium">
        Choose color mode<span className="text-danger-primary">*</span>
      </h6>
      <div className="mt-2 flex items-center gap-3">
        <label className="flex cursor-pointer items-center gap-1.5 rounded-lg border border-subtle-1 bg-layer-2 px-3 py-2 text-body-sm-regular transition-colors hover:bg-layer-2-hover">
          <input
            type="radio"
            disabled={disabled}
            name="darkPalette"
            value="false"
            checked={!value}
            onChange={() => onChange(false)}
            className="cursor-pointer"
          />
          Light mode
        </label>
        <label className="flex cursor-pointer items-center gap-1.5 rounded-lg border border-subtle-1 bg-layer-2 px-3 py-2 text-body-sm-regular transition-colors hover:bg-layer-2-hover">
          <input
            type="radio"
            disabled={disabled}
            name="darkPalette"
            value="true"
            checked={value}
            onChange={() => onChange(true)}
            className="cursor-pointer"
          />
          Dark mode
        </label>
      </div>
    </div>
  );
}
