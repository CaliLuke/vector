/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DropdownOption } from "@/components/dropdowns/dropdown-option";

import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import type { Placement } from "@popperjs/core";
import { Combobox } from "@headlessui/react";
import { usePopper } from "react-popper";
import { useTranslation } from "@plane/i18n";
import { SearchIcon } from "@plane/propel/icons";
import { ComboSearchInput } from "@plane/ui";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { usePlatformOS } from "@/hooks/use-platform-os";

export type CatalogueOption = {
  value: string | null;
  query: string;
  content: ReactNode;
};

type Props = {
  isError: boolean;
  label: string;
  noMatchesLabel: string;
  onQueryChange: (query: string) => void;
  options: readonly CatalogueOption[] | undefined;
  placement: Placement | undefined;
  query: string;
  referenceElement: HTMLButtonElement | null;
  restoreButtonFocusOnClose: () => void;
  retry: () => Promise<unknown>;
};

export function CatalogueOptionsPopup(props: Props) {
  const {
    isError,
    label,
    noMatchesLabel,
    onQueryChange,
    options,
    placement,
    query,
    referenceElement,
    restoreButtonFocusOnClose,
    retry,
  } = props;
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: [{ name: "preventOverflow", options: { padding: 12 } }],
  });

  useEffect(() => {
    if (!isMobile) inputRef.current?.focus();
  }, [isMobile]);

  return (
    <Combobox.Options modal={false} className="fixed z-10">
      <div
        className="my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 shadow-raised-200 focus:outline-none"
        ref={setPopperElement}
        style={styles.popper}
        {...attributes.popper}
      >
        <div className="flex items-center gap-1.5 rounded-sm border border-subtle bg-surface-2 px-2">
          <SearchIcon className="h-3.5 w-3.5 text-placeholder" strokeWidth={1.5} />
          <ComboSearchInput
            ref={inputRef}
            className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
            query={query}
            onQueryChange={onQueryChange}
            onEscapeClose={restoreButtonFocusOnClose}
            placeholder={t("common.search.label")}
          />
        </div>
        <div className="mt-2 max-h-48 space-y-1 overflow-y-scroll">
          {isError ? (
            <CatalogueReadFailure label={label} retry={retry} />
          ) : options ? (
            options.length > 0 ? (
              options.map((option) => (
                <DropdownOption key={option.value} value={option.value}>
                  {option.content}
                </DropdownOption>
              ))
            ) : (
              <p className="px-1.5 py-1 text-placeholder italic">{noMatchesLabel}</p>
            )
          ) : (
            <p className="px-1.5 py-1 text-placeholder italic">{t("common.loading")}</p>
          )}
        </div>
      </div>
    </Combobox.Options>
  );
}
