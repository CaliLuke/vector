/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { Fragment } from "react";
// components
import { cn } from "../utils";
import { Popover } from "./popover";
// helpers
// types
import type { TPopoverMenu } from "./types";

export function PopoverMenu<T>(props: TPopoverMenu<T>) {
  const {
    popperPosition = "bottom-end",
    popperPadding = 0,
    buttonClassName = "",
    button,
    disabled,
    panelClassName = "",
    data,
    popoverClassName = "",
    keyExtractor,
    render,
    popoverButtonRef,
    buttonAsChild,
  } = props;

  return (
    <Popover
      popperPosition={popperPosition}
      popperPadding={popperPadding}
      buttonClassName={buttonClassName}
      button={button}
      disabled={disabled}
      panelClassName={cn(
        "my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2 text-11 shadow-raised-200 focus:outline-none",
        panelClassName
      )}
      popoverClassName={popoverClassName}
      popoverButtonRef={popoverButtonRef}
      buttonAsChild={buttonAsChild}
    >
      <Fragment>
        {data.map((item, index) => (
          // The caller supplies the stable item key; index is optional context.
          // eslint-disable-next-line react/no-array-index-key
          <Fragment key={keyExtractor(item, index)}>{render(item, index)}</Fragment>
        ))}
      </Fragment>
    </Popover>
  );
}
