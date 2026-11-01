/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DropdownPortal } from "./portal";
import type { TContextMenuItem } from "./context-menu/root";
import { Menu } from "@headlessui/react";
import { MoreHorizontal } from "lucide-react";
import * as React from "react";
import ReactDOM from "react-dom";
import { usePopper } from "react-popper";
import { NESTED_MENU_POSITIONING } from "./nested-menu-position";
import { useOutsideClickDetector } from "@plane/hooks";
import { ChevronDownIcon, ChevronRightIcon } from "@plane/propel/icons";
// plane helpers
// helpers
import { cn } from "../utils";
import { getDropdownEventTarget, isCustomButtonDisabled } from "./helper";
// hooks
// types
import type {
  ICustomMenuDropdownProps,
  ICustomMenuItemProps,
  ICustomSubMenuProps,
  ICustomSubMenuTriggerProps,
  ICustomSubMenuContentProps,
} from "./helper";

function HeadlessMenuCloseNotification(props: { open: boolean; onClose?: () => void }) {
  const { open, onClose } = props;
  const wasOpen = React.useRef(false);

  React.useEffect(() => {
    if (wasOpen.current && !open) onClose?.();
    wasOpen.current = open;
  }, [open, onClose]);

  return null;
}

function Portal({
  asChild = false,
  ...props
}: Omit<React.ComponentProps<typeof DropdownPortal>, "wrap"> & { asChild?: boolean }) {
  return <DropdownPortal {...props} wrap={!asChild} />;
}

// Context for main menu to communicate with submenus
const MenuContext = React.createContext<{
  closeAllSubmenus: () => void;
  registerSubmenu: (closeSubmenu: () => void) => () => void;
} | null>(null);

function CustomMenu(props: ICustomMenuDropdownProps) {
  const {
    ariaLabel,
    buttonClassName = "",
    customButtonClassName = "",
    customButtonTabIndex = 0,
    placement,
    children,
    className = "",
    customButton,
    customButtonAsChild = false,
    disabled = false,
    ellipsis = false,
    label,
    noBorder = false,
    noChevron = false,
    verticalEllipsis = false,
    portalElement,
    menuButtonOnClick,
    onMenuClose,
    tabIndex,
    closeOnSelect,
    openOnHover = false,
    useCaptureForOutsideClick = false,
  } = props;

  const [referenceElement, setReferenceElement] = React.useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = React.useState<HTMLDivElement | null>(null);
  const [isOpen, setIsOpen] = React.useState(false);
  const customButtonDisabled = isCustomButtonDisabled(disabled, customButton);
  // refs
  const dropdownRef = React.useRef<HTMLDivElement | null>(null);
  const headlessCloseRef = React.useRef<(() => void) | null>(null);
  const submenuClosersRef = React.useRef<Set<() => void>>(new Set());

  const popper = usePopper(referenceElement, popperElement, {
    placement: placement ?? "auto",
  });

  const closeAllSubmenus = React.useCallback(() => {
    submenuClosersRef.current.forEach((closeSubmenu) => closeSubmenu());
  }, []);

  const handleHeadlessMenuClose = React.useCallback(() => {
    closeAllSubmenus();
    setIsOpen(false);
    onMenuClose?.();
  }, [closeAllSubmenus, onMenuClose]);

  const registerSubmenu = React.useCallback((closeSubmenu: () => void) => {
    submenuClosersRef.current.add(closeSubmenu);
    return () => {
      submenuClosersRef.current.delete(closeSubmenu);
    };
  }, []);

  const openDropdown = () => {
    if (openOnHover) setIsOpen(true);
    if (referenceElement) referenceElement.focus();
  };

  const closeDropdown = React.useCallback(() => {
    if (isOpen) setIsOpen(false);
    headlessCloseRef.current?.();
  }, [isOpen]);

  const handleOnClick = () => {
    if (closeOnSelect) closeDropdown();
  };

  const handleMenuButtonClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    menuButtonOnClick?.();
  };

  const handleCustomButtonClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    menuButtonOnClick?.();
  };

  const handleCustomButtonKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (customButtonDisabled) return;

    if (["Enter", " ", "ArrowDown", "ArrowUp"].includes(event.key)) {
      menuButtonOnClick?.();
    }
  };

  const handleMouseEnter = () => {
    if (openOnHover) openDropdown();
  };

  const handleMouseLeave = () => {
    if (openOnHover && isOpen) {
      setTimeout(() => {
        // Only close if menu is still open
        if (isOpen) {
          closeDropdown();
        }
      }, 150); // Small delay to allow moving to submenu
    }
  };

  useOutsideClickDetector(dropdownRef, closeDropdown, useCaptureForOutsideClick);

  // Custom handler for submenu portal clicks
  React.useEffect(() => {
    const handleDocumentClick = (event: MouseEvent) => {
      const target = getDropdownEventTarget(event);
      if (!target) return;
      const isSubmenuClick = target.closest('[data-prevent-outside-click="true"]');
      const isMainMenuClick = dropdownRef.current?.contains(target);

      // If it's a submenu click or main menu click, don't close
      if (isSubmenuClick || isMainMenuClick) {
        return;
      }

      // If menu is open and it's an outside click, close it
      if (isOpen) {
        closeDropdown();
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleDocumentClick, useCaptureForOutsideClick);

      return () => {
        document.removeEventListener("mousedown", handleDocumentClick, useCaptureForOutsideClick);
      };
    }

    return undefined;
  }, [isOpen, closeDropdown, useCaptureForOutsideClick]);

  const menuContextValue = React.useMemo(
    () => ({ closeAllSubmenus, registerSubmenu }),
    [closeAllSubmenus, registerSubmenu]
  );

  let menuItems = (
    <CustomMenuPanel appearance={props} popper={popper} setPopperElement={setPopperElement}>
      <MenuContext.Provider value={menuContextValue}>{children}</MenuContext.Provider>
    </CustomMenuPanel>
  );

  if (portalElement) {
    menuItems = ReactDOM.createPortal(menuItems, portalElement);
  }

  return (
    <Menu
      as="div"
      ref={dropdownRef}
      tabIndex={tabIndex}
      className={cn("relative w-min text-left", className)}
      role="presentation"
      onClick={(e) => {
        e.stopPropagation();
        handleOnClick();
      }}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      data-main-menu="true"
    >
      {({ open, close }) => {
        headlessCloseRef.current = close;
        return (
          <>
            <HeadlessMenuCloseNotification open={open || isOpen} onClose={handleHeadlessMenuClose} />
            {customButton ? (
              customButtonAsChild ? (
                <Menu.Button
                  as={React.Fragment}
                  {...({
                    ref: setReferenceElement,
                    onClick: handleCustomButtonClick,
                    onKeyDown: handleCustomButtonKeyDown,
                    className: customButtonClassName,
                    tabIndex: customButtonTabIndex,
                    disabled: customButtonDisabled,
                    "aria-label": ariaLabel,
                  } satisfies React.ComponentPropsWithRef<"button">)}
                >
                  {customButton}
                </Menu.Button>
              ) : (
                <Menu.Button as={React.Fragment}>
                  <button
                    ref={setReferenceElement}
                    type="button"
                    onClick={handleMenuButtonClick}
                    onKeyDown={handleCustomButtonKeyDown}
                    className={customButtonClassName}
                    tabIndex={customButtonTabIndex}
                    disabled={customButtonDisabled}
                    aria-label={ariaLabel}
                  >
                    {customButton}
                  </button>
                </Menu.Button>
              )
            ) : (
              <>
                {ellipsis || verticalEllipsis ? (
                  <Menu.Button as={React.Fragment}>
                    <button
                      ref={setReferenceElement}
                      type="button"
                      onClick={handleMenuButtonClick}
                      onKeyDown={handleCustomButtonKeyDown}
                      disabled={disabled}
                      className={`relative grid place-items-center rounded-sm p-1 text-secondary outline-none hover:text-primary ${
                        disabled ? "cursor-not-allowed" : "cursor-pointer hover:bg-layer-transparent-hover"
                      } ${buttonClassName}`}
                      tabIndex={customButtonTabIndex}
                      aria-label={ariaLabel}
                    >
                      <MoreHorizontal className={`h-3.5 w-3.5 ${verticalEllipsis ? "rotate-90" : ""}`} />
                    </button>
                  </Menu.Button>
                ) : (
                  <Menu.Button as={React.Fragment}>
                    <button
                      ref={setReferenceElement}
                      type="button"
                      className={`flex items-center justify-between gap-1 rounded-md px-2.5 py-1 text-11 whitespace-nowrap duration-300 ${
                        open ? "text-primary" : "text-secondary"
                      } ${noBorder ? "" : "shadow-sm border border-strong focus:outline-none"} ${
                        disabled
                          ? "cursor-not-allowed text-secondary"
                          : "cursor-pointer hover:bg-layer-transparent-hover"
                      } ${buttonClassName}`}
                      onClick={handleMenuButtonClick}
                      onKeyDown={handleCustomButtonKeyDown}
                      tabIndex={customButtonTabIndex}
                      disabled={disabled}
                      aria-label={ariaLabel}
                    >
                      {label}
                      {!noChevron && <ChevronDownIcon className="h-3.5 w-3.5" />}
                    </button>
                  </Menu.Button>
                )}
              </>
            )}
            {(open || (openOnHover && isOpen)) && menuItems}
          </>
        );
      }}
    </Menu>
  );
}

function CustomMenuPanel({
  appearance,
  popper,
  setPopperElement,
  children,
}: {
  appearance: ICustomMenuDropdownProps;
  popper: ReturnType<typeof usePopper>;
  setPopperElement: React.Dispatch<React.SetStateAction<HTMLDivElement | null>>;
  children: React.ReactNode;
}) {
  const { portalElement, menuItemsClassName = "", optionsClassName = "", maxHeight = "md" } = appearance;
  return (
    <Menu.Items
      data-prevent-outside-click={!!portalElement}
      className={cn(
        "fixed z-30 translate-y-0",
        menuItemsClassName
      )} /** translate-y-0 is a hack to create new stacking context. Required for safari  */
      static
    >
      <div
        className={cn(
          "shadow-md my-1 min-w-[12rem] overflow-y-scroll rounded-md border border-strong-1 bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap ring-1 ring-strong-1/15 outline-none focus:outline-none",
          {
            "max-h-60": maxHeight === "lg",
            "max-h-48": maxHeight === "md",
            "max-h-36": maxHeight === "rg",
            "max-h-28": maxHeight === "sm",
          },
          optionsClassName
        )}
        ref={setPopperElement}
        style={popper.styles.popper}
        {...popper.attributes.popper}
      >
        {children}
      </div>
    </Menu.Items>
  );
}

// SubMenu context for closing submenu from nested items
const SubMenuContext = React.createContext<{ closeSubmenu: () => void } | null>(null);

// Hook to use submenu context
const useSubMenu = () => React.useContext(SubMenuContext);

// SubMenu implementation
function SubMenu(props: ICustomSubMenuProps) {
  const {
    children,
    trigger,
    disabled = false,
    className = "",
    contentClassName = "",
    placement = "right-start",
  } = props;

  const [isOpen, setIsOpen] = React.useState(false);
  const [referenceElement, setReferenceElement] = React.useState<HTMLSpanElement | null>(null);
  const [popperElement, setPopperElement] = React.useState<HTMLDivElement | null>(null);
  const submenuRef = React.useRef<HTMLDivElement | null>(null);

  const menuContext = React.useContext(MenuContext);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    ...NESTED_MENU_POSITIONING,
    placement: placement,
  });

  const closeSubmenu = React.useCallback(() => {
    setIsOpen(false);
  }, []);

  // Register this submenu with the main menu context
  React.useEffect(() => {
    if (menuContext) {
      return menuContext.registerSubmenu(closeSubmenu);
    }

    return undefined;
  }, [menuContext, closeSubmenu]);

  const toggleSubmenu = () => {
    if (!disabled) {
      // Close other submenus when opening this one
      if (!isOpen && menuContext) {
        menuContext.closeAllSubmenus();
      }
      setIsOpen(!isOpen);
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    toggleSubmenu();
  };

  const subMenuContextValue = React.useMemo(() => ({ closeSubmenu }), [closeSubmenu]);

  // Close submenu when clicking on other menu items
  React.useEffect(() => {
    const handleMenuItemClick = (e: Event) => {
      const eventTarget = e.target;
      const target =
        eventTarget instanceof Element ? eventTarget : eventTarget instanceof Node ? eventTarget.parentElement : null;
      if (!target) return;
      // Check if the click is on a menu item that's not part of this submenu
      if (target.closest('[role="menuitem"]') && !submenuRef.current?.contains(target)) {
        closeSubmenu();
      }
    };

    document.addEventListener("click", handleMenuItemClick);
    return () => {
      document.removeEventListener("click", handleMenuItemClick);
    };
  }, [closeSubmenu]);

  return (
    <div ref={submenuRef} className={cn("relative", className)}>
      <span ref={setReferenceElement} className="w-full">
        <Menu.Item as="div" disabled={disabled}>
          {({ active }) => (
            <button
              type="button"
              className={cn(
                "font-inherit flex w-full cursor-pointer items-center justify-between rounded-sm border-0 bg-transparent px-1 py-1.5 text-left text-secondary outline-none select-none",
                {
                  "bg-layer-transparent-hover": active && !disabled,
                  "text-placeholder": disabled,
                  "cursor-not-allowed": disabled,
                }
              )}
              onClick={handleClick}
              disabled={disabled}
            >
              <span className="flex-1">{trigger}</span>
              <ChevronRightIcon className="h-3.5 w-3.5 flex-shrink-0" />
            </button>
          )}
        </Menu.Item>
      </span>

      {isOpen && (
        <Portal>
          <div
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className={cn(
              "shadow-md fixed z-30 min-w-[12rem] overflow-hidden rounded-md border border-strong-1 bg-surface-1 p-1 text-11 ring-1 ring-strong-1/15",
              contentClassName
            )}
            data-prevent-outside-click="true"
            onMouseEnter={() => {
              // Notify parent menu that we're hovering over submenu
              const mainMenuElement = document.querySelector('[data-main-menu="true"]');
              if (mainMenuElement) {
                const mouseEnterEvent = new MouseEvent("mouseenter", { bubbles: true });
                mainMenuElement.dispatchEvent(mouseEnterEvent);
              }
            }}
            onMouseLeave={() => {
              // Notify parent menu that we're leaving submenu
              const mainMenuElement = document.querySelector('[data-main-menu="true"]');
              if (mainMenuElement) {
                const mouseLeaveEvent = new MouseEvent("mouseleave", { bubbles: true });
                mainMenuElement.dispatchEvent(mouseLeaveEvent);
              }
            }}
          >
            <SubMenuContext.Provider value={subMenuContextValue}>{children}</SubMenuContext.Provider>
          </div>
        </Portal>
      )}
    </div>
  );
}

function MenuItem(props: ICustomMenuItemProps) {
  const { children, disabled = false, onClick, className } = props;
  const submenuContext = useSubMenu();

  return (
    <Menu.Item as={React.Fragment} disabled={disabled}>
      {({ active, close }) => (
        <button
          type="button"
          className={cn(
            "w-full truncate rounded-sm px-1 py-1.5 text-left text-secondary select-none",
            {
              "bg-layer-transparent-hover": active && !disabled,
              "text-placeholder": disabled,
            },
            className
          )}
          onClick={(e) => {
            close();
            onClick?.(e);
            // Close submenu if this item is inside a submenu
            submenuContext?.closeSubmenu();
          }}
          disabled={disabled}
        >
          {children}
        </button>
      )}
    </Menu.Item>
  );
}

function ActionItem({
  item,
  onClick,
  iconClassName = "h-3 w-3 flex-shrink-0",
  contentClassName,
  children,
}: {
  item: TContextMenuItem;
  onClick?: React.MouseEventHandler<HTMLButtonElement>;
  iconClassName?: string;
  contentClassName?: string;
  children?: React.ReactNode;
}) {
  if (item.shouldRender === false) return null;
  return (
    <MenuItem
      onClick={onClick ?? item.action}
      disabled={item.disabled}
      className={cn("flex items-center gap-2", { "text-placeholder": item.disabled }, item.className)}
    >
      {item.icon && <item.icon className={cn(iconClassName, item.iconClassName)} />}
      <div className={contentClassName}>
        <h5>{item.title}</h5>
        {item.description && (
          <p className={cn("whitespace-pre-line text-tertiary", { "text-placeholder": item.disabled })}>
            {item.description}
          </p>
        )}
      </div>
      {children}
    </MenuItem>
  );
}

function SubMenuTrigger(props: ICustomSubMenuTriggerProps) {
  const { children, disabled = false, className } = props;

  return (
    <Menu.Item as="div" disabled={disabled}>
      {({ active }) => (
        <div
          className={cn(
            "flex w-full items-center justify-between rounded-sm px-1 py-1.5 text-left text-secondary select-none",
            {
              "bg-layer-transparent-hover": active && !disabled,
              "text-placeholder": disabled,
              "cursor-pointer": !disabled,
              "cursor-not-allowed": disabled,
            },
            className
          )}
        >
          <span className="flex-1">{children}</span>
          <ChevronRightIcon className="h-3.5 w-3.5 flex-shrink-0" />
        </div>
      )}
    </Menu.Item>
  );
}

function SubMenuContent(props: ICustomSubMenuContentProps) {
  const { children, className } = props;

  return (
    <div
      className={cn(
        "shadow-md z-[15] min-w-[12rem] overflow-hidden rounded-md border border-strong-1 bg-surface-1 p-1 text-11 ring-1 ring-strong-1/15",
        className
      )}
    >
      {children}
    </div>
  );
}

// Add all components as static properties for external use
CustomMenu.Portal = Portal;
CustomMenu.MenuItem = MenuItem;
CustomMenu.ActionItem = ActionItem;
CustomMenu.SubMenu = SubMenu;
CustomMenu.SubMenuTrigger = SubMenuTrigger;
CustomMenu.SubMenuContent = SubMenuContent;

export { CustomMenu };
