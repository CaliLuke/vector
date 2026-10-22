/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
// helpers
import { Button } from "@plane/propel/button";

type Props = Omit<React.ComponentPropsWithoutRef<typeof Button>, "children" | "size" | "variant"> & {
  icon: React.ReactNode;
  title: string;
};

export const IssueDetailWidgetButton = React.forwardRef<HTMLButtonElement, Props>(
  function IssueDetailWidgetButton(props, ref) {
    const { icon, title, ...buttonProps } = props;
    return (
      <Button ref={ref} variant="secondary" size="lg" {...buttonProps}>
        {icon}
        <span className="text-body-xs-medium">{title}</span>
      </Button>
    );
  }
);
