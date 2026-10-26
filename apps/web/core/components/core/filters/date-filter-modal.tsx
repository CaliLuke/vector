/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { useState } from "react";
import { Button } from "@plane/propel/button";
import { Calendar } from "@plane/propel/calendar";
import { CloseIcon } from "@plane/propel/icons";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { renderFormattedPayloadDate, renderFormattedDate } from "@plane/utils/datetime";
import { DateFilterSelect } from "./date-filter-select";
type Props = {
  title: string;
  handleClose: () => void;
  isOpen: boolean;
  onSelect: (val: string[]) => void;
};

type TFormValues = {
  filterType: "before" | "after" | "range";
  date1: Date;
  date2: Date;
};

export function DateFilterModal(props: Props) {
  if (!props.isOpen) return null;
  return <DateFilterForm {...props} />;
}

function DateFilterForm({ title, handleClose, onSelect }: Props) {
  const [defaultValues] = useState<TFormValues>(() => {
    const today = new Date();
    return {
      filterType: "range",
      date1: today,
      date2: new Date(today.getFullYear(), today.getMonth() + 1, today.getDate()),
    };
  });
  const form = useForm({
    defaultValues,
    validators: {
      onChange: ({ value }) =>
        value.filterType === "range" && value.date1 > value.date2
          ? "The end date must be on or after the start date."
          : undefined,
    },
    onSubmit: ({ value: { filterType, date1, date2 } }) => {
      onSelect(
        filterType === "range"
          ? [`${renderFormattedPayloadDate(date1)};after`, `${renderFormattedPayloadDate(date2)};before`]
          : [`${renderFormattedPayloadDate(date1)};${filterType}`]
      );
      handleClose();
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const isInvalid = useStore(form.store, (state) => !state.isValid);

  return (
    <ModalCore isOpen handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
        className="space-y-4 px-5 py-8 sm:p-6"
      >
        <div className="flex w-full justify-between">
          <form.Field name="filterType">
            {(field) => <DateFilterSelect title={title} value={field.state.value} onChange={field.handleChange} />}
          </form.Field>
          <button type="button" aria-label="Close date filter" onClick={handleClose}>
            <CloseIcon className="h-4 w-4" />
          </button>
        </div>
        <div className="flex w-full justify-between gap-4">
          {(["date1", "date2"] as const)
            .filter((name) => name === "date1" || values.filterType === "range")
            .map((name) => (
              <form.Field key={name} name={name}>
                {(field) => (
                  <Calendar
                    className="rounded-md border border-subtle p-3"
                    captionLayout="dropdown"
                    selected={field.state.value}
                    defaultMonth={field.state.value}
                    onSelect={(date) => {
                      if (date) field.handleChange(date);
                    }}
                    mode="single"
                    disabled={
                      values.filterType === "range"
                        ? [name === "date1" ? { after: values.date2 } : { before: values.date1 }]
                        : undefined
                    }
                  />
                )}
              </form.Field>
            ))}
        </div>
        {values.filterType === "range" && (
          <h6 className="flex items-center gap-1 text-11">
            <span className="text-secondary">After:</span>
            <span>{renderFormattedDate(values.date1)}</span>
            <span className="ml-1 text-secondary">Before:</span>
            {!isInvalid && <span>{renderFormattedDate(values.date2)}</span>}
          </h6>
        )}
        <div className="flex justify-end gap-4">
          <Button variant="secondary" type="button" size="lg" onClick={handleClose}>
            Cancel
          </Button>
          <Button variant="primary" size="lg" type="submit" disabled={isInvalid}>
            Apply
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
