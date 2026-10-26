/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { useStore } from "@tanstack/react-form";
import type { InvitationFormApi } from "@/hooks/use-workspace-invitation";
import { usePopper } from "react-popper";
import { XCircle } from "lucide-react";
import { Listbox } from "@headlessui/react";
import { ROLE, ROLE_DETAILS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CheckIcon, ChevronDownIcon } from "@plane/propel/icons";
import { Input } from "@plane/ui";

type InviteMemberFormProps = {
  emailColumnClassName: string;
  index: number;
  count: number;
  form: InvitationFormApi;
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
const InviteMemberInput = function InviteMemberInput(props: InviteMemberFormProps) {
  const { form, emailColumnClassName, index, count } = props;
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const email = useStore(form.store, (state) => state.values.emails[index]?.email ?? "");

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const { t } = useTranslation();

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
          <form.Field
            name={`emails[${index}].email`}
            validators={{
              onChange: ({ value }) => (!value || emailRegex.test(value) ? undefined : "Invalid Email ID"),
            }}
          >
            {(field) => (
              <Input
                id={field.name}
                name={field.name}
                type="email"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                hasError={!field.state.meta.isValid}
                placeholder={placeholderEmails[index % placeholderEmails.length]}
                className="w-full border-strong text-11 placeholder:text-placeholder sm:text-13"
                autoComplete="off"
                disabled={isSubmitting}
              />
            )}
          </form.Field>
        </div>
        <div className="col-span-4 mr-8">
          <form.Field name={`emails[${index}].role`}>
            {(field) => (
              <Listbox
                as="div"
                value={field.state.value}
                onChange={field.handleChange}
                disabled={isSubmitting}
                className="w-full flex-shrink-0 text-left"
              >
                <Listbox.Button
                  type="button"
                  ref={setReferenceElement}
                  className="flex w-full items-center justify-between gap-1 rounded-md border-[0.5px] border-strong px-2.5 py-2 text-13"
                >
                  <span
                    className={`text-13 ${
                      !email && !field.state.meta.isDirty ? "text-placeholder" : "text-primary"
                    } sm:text-13`}
                  >
                    {ROLE[field.state.value]}
                  </span>

                  <ChevronDownIcon
                    className={`size-3 ${
                      !email && !field.state.meta.isDirty ? "stroke-placeholder" : "stroke-primary"
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
          </form.Field>
        </div>
        {count > 1 && (
          <button
            type="button"
            className="absolute right-0 hidden place-items-center self-center rounded-sm group-hover:grid"
            aria-label="Remove invitation"
            disabled={isSubmitting}
            onClick={() => {
              void form.removeFieldValue("emails", index);
            }}
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
};

export function InviteMemberFields({
  form,
  emailColumnClassName,
}: {
  form: InvitationFormApi;
  emailColumnClassName: string;
}) {
  return (
    <form.Field name="emails" mode="array">
      {(field) =>
        field.state.value.map((entry, index) => (
          <InviteMemberInput
            key={entry.id}
            form={form}
            index={index}
            count={field.state.value.length}
            emailColumnClassName={emailColumnClassName}
          />
        ))
      }
    </form.Field>
  );
}
