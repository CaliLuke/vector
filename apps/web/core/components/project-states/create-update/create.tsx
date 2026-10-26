/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { STATE_GROUPS } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IState, TStateGroups, TStateOperationsCallbacks } from "@plane/types";
// components
import { StateForm } from "./form";

type TStateCreate = {
  groupKey: TStateGroups;
  shouldTrackEvents?: boolean;
  createStateCallback: TStateOperationsCallbacks["createState"];
  handleClose: () => void;
};

export const StateCreate = function StateCreate(props: TStateCreate) {
  const { groupKey, createStateCallback, handleClose } = props;

  const onSubmit = async (formData: Partial<IState>) => {
    try {
      await createStateCallback({ ...formData, group: groupKey });

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State created successfully.",
      });
      handleClose();
    } catch (error) {
      const duplicate = typeof error === "object" && error !== null && "status" in error && error.status === 400;
      throw new Error(
        duplicate
          ? "Another state exists with the same name. Please use a different name."
          : "State could not be created. Please try again.",
        { cause: error }
      );
    }
  };

  return (
    <StateForm
      data={{ name: "", description: "", color: STATE_GROUPS[groupKey].color }}
      onSubmit={onSubmit}
      onCancel={handleClose}
      buttonTitle="Create"
    />
  );
};
