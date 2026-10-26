/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IState, TStateOperationsCallbacks } from "@plane/types";
// components
import { StateForm } from "./form";

type TStateUpdate = {
  state: IState;
  updateStateCallback: TStateOperationsCallbacks["updateState"];
  shouldTrackEvents: boolean;
  handleClose: () => void;
};

export const StateUpdate = function StateUpdate(props: TStateUpdate) {
  const { state, updateStateCallback, handleClose } = props;
  const onSubmit = async (formData: Partial<IState>) => {
    try {
      await updateStateCallback(state.id, formData);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State updated successfully.",
      });
      handleClose();
    } catch (error) {
      const duplicate = typeof error === "object" && error !== null && "status" in error && error.status === 400;
      throw new Error(
        duplicate
          ? "Another state exists with the same name. Please use a different name."
          : "State could not be updated. Please try again.",
        { cause: error }
      );
    }
  };

  return <StateForm key={state.id} data={state} onSubmit={onSubmit} onCancel={handleClose} buttonTitle="Update" />;
};
