/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { InputColorPicker } from "@plane/ui";

type Props = {
  name: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  disabled?: boolean;
};

export function CustomThemeColorInput({ name, label, value, onChange, error, disabled }: Props) {
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={name} className="text-body-sm-medium">
        {label}
        <span className="text-danger-primary">*</span>
      </label>
      <InputColorPicker
        name={name}
        value={value}
        onChange={(color) => onChange(color && !color.startsWith("#") ? `#${color}` : (color ?? ""))}
        placeholder={name === "primary" ? "#3f76ff" : "#1a1a1a"}
        className="w-full placeholder:text-placeholder"
        style={{ backgroundColor: value, color: "#ffffff" }}
        hasError={!!error}
        disabled={disabled}
      />
      {error && (
        <p role="alert" className="text-11 text-danger-primary">
          {error}
        </p>
      )}
    </div>
  );
}
