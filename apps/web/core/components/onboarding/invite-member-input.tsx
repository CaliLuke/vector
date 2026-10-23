/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { observer } from "mobx-react";
import type {
  Control,
  FieldArrayWithId,
  FieldErrors,
  UseFieldArrayRemove,
  UseFormGetValues,
  UseFormSetValue,
  UseFormWatch,
} from "react-hook-form";
import { Controller } from "react-hook-form";
import { usePopper } from "react-popper";
import { XCircle } from "lucide-react";
import { Listbox } from "@headlessui/react";
import type { EUserPermissions } from "@plane/constants";
import { ROLE, ROLE_DETAILS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CheckIcon, ChevronDownIcon } from "@plane/propel/icons";
import { Input } from "@plane/ui";

type EmailRole = {
  email: string;
  role: EUserPermissions;
  role_active: boolean;
};

export type FormValues = {
  emails: EmailRole[];
};

type InviteMemberFormProps = {
  emailColumnClassName: string;
  index: number;
  remove: UseFieldArrayRemove;
  control: Control<FormValues>;
  setValue: UseFormSetValue<FormValues>;
  getValues: UseFormGetValues<FormValues>;
  watch: UseFormWatch<FormValues>;
  fields: FieldArrayWithId<FormValues, "emails">[];
  errors: FieldErrors<FormValues>;
  isInvitationDisabled: boolean;
  setIsInvitationDisabled: (value: boolean) => void;
};

const emailRegex = /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i;

const placeholderEmails = [
  "charlie.taylor@frstflt.com",
  "octave.chanute@frstflt.com",
  "george.spratt@frstflt.com",
  "frank.coffyn@frstflt.com",
  "amos.root@frstflt.com",
  "edward.deeds@frstflt.com",
  "charles.m.manly@frstflt.com",
  "glenn.curtiss@frstflt.com",
  "thomas.selfridge@frstflt.com",
  "albert.zahm@frstflt.com",
];
export const InviteMemberInput = observer(function InviteMemberInput(props: InviteMemberFormProps) {
  const {
    control,
    emailColumnClassName,
    index,
    fields,
    remove,
    errors,
    isInvitationDisabled,
    setIsInvitationDisabled,
    setValue,
    getValues,
    watch,
  } = props;

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const { t } = useTranslation();

  const email = watch(`emails.${index}.email`);

  const emailOnChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.value === "") {
      const validEmail = fields.map((_, i) => emailRegex.test(getValues(`emails.${i}.email`))).includes(true);
      if (validEmail) {
        setIsInvitationDisabled(false);
      } else {
        setIsInvitationDisabled(true);
      }

      if (getValues(`emails.${index}.role_active`)) {
        setValue(`emails.${index}.role_active`, false);
      }
    } else {
      if (!getValues(`emails.${index}.role_active`)) {
        setValue(`emails.${index}.role_active`, true);
      }
      if (isInvitationDisabled && emailRegex.test(event.target.value)) {
        setIsInvitationDisabled(false);
      } else if (!isInvitationDisabled && !emailRegex.test(event.target.value)) {
        setIsInvitationDisabled(true);
      }
    }
  };

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: "bottom-end",
    modifiers: [
      {
        name: "preventOverflow",
        options: {
          padding: 12,
        },
      },
    ],
  });

  return (
    <div>
      <div className="group relative grid grid-cols-10 gap-4">
        <div className={emailColumnClassName}>
          <Controller
            control={control}
            name={`emails.${index}.email`}
            rules={{
              pattern: {
                value: emailRegex,
                message: "Invalid Email ID",
              },
            }}
            render={({ field: { value, onChange, ref } }) => (
              <Input
                id={`emails.${index}.email`}
                name={`emails.${index}.email`}
                type="text"
                value={value}
                onChange={(event) => {
                  emailOnChange(event);
                  onChange(event);
                }}
                ref={ref}
                hasError={Boolean(errors.emails?.[index]?.email)}
                placeholder={placeholderEmails[index % placeholderEmails.length]}
                className="w-full border-strong text-11 placeholder:text-placeholder sm:text-13"
                autoComplete="off"
              />
            )}
          />
        </div>
        <div className="col-span-4 mr-8">
          <Controller
            control={control}
            name={`emails.${index}.role`}
            rules={{ required: true }}
            render={({ field: { value, onChange } }) => (
              <Listbox
                as="div"
                value={value}
                onChange={(val) => {
                  onChange(val);
                  setValue(`emails.${index}.role_active`, true);
                }}
                className="w-full flex-shrink-0 text-left"
              >
                <Listbox.Button
                  type="button"
                  ref={setReferenceElement}
                  className="flex w-full items-center justify-between gap-1 rounded-md border-[0.5px] border-strong px-2.5 py-2 text-13"
                >
                  <span
                    className={`text-13 ${
                      !getValues(`emails.${index}.role_active`) ? "text-placeholder" : "text-primary"
                    } sm:text-13`}
                  >
                    {ROLE[value]}
                  </span>

                  <ChevronDownIcon
                    className={`size-3 ${
                      !getValues(`emails.${index}.role_active`) ? "stroke-placeholder" : "stroke-primary"
                    }`}
                  />
                </Listbox.Button>

                <Listbox.Options as="div">
                  <div
                    className="shadow-sm absolute z-10 mt-1 h-fit w-48 space-y-1 rounded-md border border-strong bg-surface-1 p-2 focus:outline-none sm:w-60"
                    ref={setPopperElement}
                    style={styles.popper}
                    {...attributes.popper}
                  >
                    {Object.entries(ROLE_DETAILS).map(([key, roleDetails]) => (
                      <Listbox.Option
                        as="div"
                        key={key}
                        value={parseInt(key)}
                        className={({ active, selected }) =>
                          `cursor-pointer truncate rounded-sm px-1 py-1.5 select-none ${
                            active || selected ? "bg-onboarding-background-400/40" : ""
                          } ${selected ? "text-primary" : "text-secondary"}`
                        }
                      >
                        {({ selected }) => (
                          <div className="flex items-center gap-2 p-1 text-wrap">
                            <div className="flex flex-col">
                              <div className="text-13 font-medium">{t(roleDetails.i18n_title)}</div>
                              <div className="flex text-11 text-tertiary">{t(roleDetails.i18n_description)}</div>
                            </div>
                            {selected && <CheckIcon className="h-4 w-4 shrink-0" />}
                          </div>
                        )}
                      </Listbox.Option>
                    ))}
                  </div>
                </Listbox.Options>
              </Listbox>
            )}
          />
        </div>
        {fields.length > 1 && (
          <button
            type="button"
            className="absolute right-0 hidden place-items-center self-center rounded-sm group-hover:grid"
            onClick={() => remove(index)}
          >
            <XCircle className="h-5 w-5 pl-0.5 text-placeholder" />
          </button>
        )}
      </div>
      {email && !emailRegex.test(email) && (
        <div className="mx-8 my-1">
          <span className="text-13">🤥</span>{" "}
          <span className="mt-1 text-11 text-danger-primary">That doesn{"'"}t look like an email address.</span>
        </div>
      )}
    </div>
  );
});
