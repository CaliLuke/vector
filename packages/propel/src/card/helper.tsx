/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export enum ECardVariant {
  WITHOUT_SHADOW = "without-shadow",
  WITH_SHADOW = "with-shadow",
}
export enum ECardDirection {
  ROW = "row",
  COLUMN = "column",
}
export enum ECardSpacing {
  SM = "sm",
  LG = "lg",
}
export type TCardVariant = ECardVariant.WITHOUT_SHADOW | ECardVariant.WITH_SHADOW;
export type TCardDirection = ECardDirection.ROW | ECardDirection.COLUMN;
export type TCardSpacing = ECardSpacing.SM | ECardSpacing.LG;

interface ICardProperties {
  [key: string]: string;
}

const DEFAULT_STYLE = "bg-surface-1 rounded-lg border-[0.5px] border-subtle w-full flex flex-col";
const containerStyle: ICardProperties = {
  [ECardVariant.WITHOUT_SHADOW]: "",
  [ECardVariant.WITH_SHADOW]: "hover:shadow-raised-200 duration-300",
};
const spacings = {
  [ECardSpacing.SM]: "p-4",
  [ECardSpacing.LG]: "p-6",
};
const directions = {
  [ECardDirection.ROW]: "flex-row space-x-3",
  [ECardDirection.COLUMN]: "flex-col space-y-3",
};
export const getCardStyle = (variant: TCardVariant, spacing: TCardSpacing, direction: TCardDirection) =>
  DEFAULT_STYLE + " " + directions[direction] + " " + containerStyle[variant] + " " + spacings[spacing];
