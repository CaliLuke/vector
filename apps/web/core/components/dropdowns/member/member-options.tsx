/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import type { Placement } from "@popperjs/core";

import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
import { Combobox } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { CheckIcon, SearchIcon, SuspendedUserIcon } from "@plane/propel/icons";
import { EPillSize, EPillVariant, Pill } from "@plane/propel/pill";
import type { IUserLite } from "@plane/types";
import { Avatar, ComboSearchInput } from "@plane/ui";
import { Button } from "@plane/propel/button";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
import { sortByCurrentUserThenSelected } from "@plane/utils/array";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { usePlatformOS } from "@/hooks/use-platform-os";

interface Props {
  className?: string;
  getUserDetails: (userId: string) => IUserLite | undefined;
  getIsUserSuspended: (userId: string) => boolean;
  isRosterLoading: boolean;
  isRosterError: boolean;
  onRosterRetry: () => void;
  memberIds?: string[];
  optionsClassName?: string;
  placement: Placement | undefined;
  referenceElement: HTMLButtonElement | null;
  restoreButtonFocusOnClose: () => void;
  value?: string[] | string | null;
}

export function MemberOptions(props: Props) {
  const {
    getUserDetails,
    getIsUserSuspended,
    isRosterLoading,
    isRosterError,
    onRosterRetry,
    memberIds,
    optionsClassName = "",
    placement,
    referenceElement,
    restoreButtonFocusOnClose,
    value,
  } = props;
  // refs
  const inputRef = useRef<HTMLInputElement | null>(null);
  // states
  const [query, setQuery] = useState("");
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { isMobile } = usePlatformOS();
  // popper-js init
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: [
      {
        name: "preventOverflow",
        options: {
          padding: 12,
        },
      },
    ],
  });

  useEffect(() => {
    if (!isMobile) inputRef.current?.focus();
  }, [isMobile]);

  const options = memberIds
    ?.map((userId) => {
      const userDetails = getUserDetails(userId);
      return {
        value: userId,
        query: `${userDetails?.display_name} ${userDetails?.first_name} ${userDetails?.last_name}`,
        content: (
          <div className="flex items-center gap-2">
            <div className="w-4">
              {getIsUserSuspended(userId) ? (
                <SuspendedUserIcon className="h-3.5 w-3.5 text-placeholder" />
              ) : (
                <Avatar name={userDetails?.display_name} src={getFileURL(userDetails?.avatar_url ?? "")} />
              )}
            </div>
            <span className={cn("flex-grow truncate", getIsUserSuspended(userId) ? "text-placeholder" : "")}>
              {currentUser?.id === userId ? t("you") : userDetails?.display_name}
            </span>
          </div>
        ),
      };
    })
    .filter((o) => !!o);

  const filteredOptions = sortByCurrentUserThenSelected(
    query === "" ? options : options?.filter((o) => o?.query.toLowerCase().includes(query.toLowerCase())),
    value,
    currentUser?.id
  );

  return createPortal(
    <Combobox.Options modal={false} data-prevent-outside-click>
      <div
        className={cn(
          "z-30 my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 shadow-raised-200 focus:outline-none",
          optionsClassName
        )}
        ref={setPopperElement}
        style={{
          ...styles.popper,
        }}
        {...attributes.popper}
      >
        <div className="flex items-center gap-1.5 rounded-sm border border-subtle bg-surface-2 px-2">
          <SearchIcon className="h-3.5 w-3.5 text-placeholder" strokeWidth={1.5} />
          <ComboSearchInput
            ref={inputRef}
            className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
            query={query}
            onQueryChange={setQuery}
            onEscapeClose={restoreButtonFocusOnClose}
            placeholder={t("search")}
          />
        </div>
        <div className="mt-2 max-h-48 space-y-1 overflow-y-scroll">
          {isRosterLoading ? (
            <p className="px-1.5 py-1 text-placeholder italic">{t("loading")}</p>
          ) : isRosterError ? (
            <div className="flex items-center justify-between gap-2 px-1.5 py-1">
              <p className="text-placeholder italic">{t("something_went_wrong_please_try_again")}</p>
              <Button type="button" variant="link" size="sm" onClick={onRosterRetry}>
                {t("retry")}
              </Button>
            </div>
          ) : filteredOptions ? (
            filteredOptions.length > 0 ? (
              filteredOptions.map(
                (option) =>
                  option && (
                    <Combobox.Option
                      key={option.value}
                      value={option.value}
                      className={({ active, selected }) =>
                        cn(
                          "flex w-full items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none",
                          active && "bg-layer-transparent-hover",
                          selected ? "text-primary" : "text-secondary",
                          getIsUserSuspended(option.value) ? "cursor-not-allowed" : "cursor-pointer"
                        )
                      }
                      disabled={getIsUserSuspended(option.value)}
                    >
                      {({ selected }) => (
                        <>
                          <span className="flex-grow truncate">{option.content}</span>
                          {selected && <CheckIcon className="h-3.5 w-3.5 flex-shrink-0" />}
                          {getIsUserSuspended(option.value) && (
                            <Pill variant={EPillVariant.DEFAULT} size={EPillSize.XS} className="border-none">
                              Suspended
                            </Pill>
                          )}
                        </>
                      )}
                    </Combobox.Option>
                  )
              )
            ) : (
              <p className="px-1.5 py-1 text-placeholder italic">{t("no_matching_results")}</p>
            )
          ) : null}
        </div>
      </div>
    </Combobox.Options>,
    document.body
  );
}
