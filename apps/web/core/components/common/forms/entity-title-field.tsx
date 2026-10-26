/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Ref } from "react";
import { Input } from "@plane/ui";

type Props = {
  id: string;
  name: string;
  value: string;
  onChange: (value: string) => void;
  onBlur: () => void;
  disabled: boolean;
  hasError: boolean;
  placeholder: string;
  tabIndex?: number;
  errors: readonly (string | undefined)[];
  inputRef?: Ref<HTMLInputElement>;
  className?: string;
};

export function EntityTitleField(props: Props) {
  const { id, name, value, onChange, onBlur, disabled, hasError, placeholder, tabIndex, errors, inputRef, className } =
    props;
  return (
    <div className={className ?? "space-y-1"}>
      <Input
        ref={inputRef}
        id={id}
        name={name}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onBlur={onBlur}
        disabled={disabled}
        hasError={hasError}
        placeholder={placeholder}
        className="w-full text-14"
        tabIndex={tabIndex}
      />
      {errors.map((error) =>
        error === undefined ? null : (
          <p key={error} role="alert" className="text-11 text-danger-primary">
            {error}
          </p>
        )
      )}
    </div>
  );
}
