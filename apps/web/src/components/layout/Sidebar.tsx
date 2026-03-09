"use client";

import Link from "next/link";
import { useMemo } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import Image from "next/image";
import { LogOut, PanelLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { SidebarItem } from "./SidebarItem";
import { APP_NAV_ITEMS, getSidebarSubnavItems, isNavItemActive, normalizeSettingsHref } from "./navigation";

interface SidebarProps {
  expanded: boolean;
  onToggle: () => void;
  onLogout: () => void;
}

export function Sidebar({ expanded, onToggle, onLogout }: SidebarProps) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const subnavItems = useMemo(
    () => getSidebarSubnavItems(pathname, new URLSearchParams(searchParams.toString())),
    [pathname, searchParams]
  );

  return (
    <aside
      className={cn(
        "hidden md:flex flex-col border-r border-border bg-background transition-all duration-200",
        expanded ? "w-[208px]" : "w-[52px]"
      )}
    >
      {/* Header */}
      <div className="flex min-h-[64px] items-center gap-2 border-b border-border px-3 py-3">
        <button
          onClick={onToggle}
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-text-secondary hover:bg-surface-accent transition-colors"
        >
          <PanelLeft size={18} />
        </button>
        <Link
          href="/home"
          className="flex items-center gap-2 transition-all duration-200 overflow-hidden"
        >
          <Image src="/logo.svg" alt="LifeBase" width={24} height={24} />
          <span
            className={cn(
              "text-sm font-semibold text-text-strong whitespace-nowrap transition-all duration-200",
              expanded ? "opacity-100 w-auto" : "opacity-0 w-0 overflow-hidden"
            )}
          >
            LifeBase
          </span>
        </Link>
      </div>

      {/* Nav */}
      <nav className="flex-1 space-y-1 px-2 py-2">
        {APP_NAV_ITEMS.map(({ href, label, icon: Icon, hasSubnav }) => {
          const isActive = isNavItemActive(pathname, href);
          const isOpen = expanded && hasSubnav && isActive;
          const linkHref = normalizeSettingsHref(href);
          return (
            <div key={href}>
              <SidebarItem
                href={linkHref}
                label={label}
                icon={<Icon size={18} />}
                isActive={isActive}
                expanded={expanded}
                endIcon={hasSubnav ? (
                  <ChevronRight
                    size={14}
                    className={cn(
                      "transition-transform duration-200",
                      isOpen ? "rotate-90" : ""
                    )}
                  />
                ) : undefined}
              />
              {isOpen && subnavItems.length > 0 && (
                <div className="ml-8 mr-1.5 space-y-0.5 pb-1">
                  {subnavItems.map((item) => (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={cn(
                        "flex items-center rounded-md px-2.5 py-1.5 text-xs transition-colors",
                        item.isActive
                          ? "bg-surface-accent/80 font-medium text-text-strong"
                          : "text-text-muted hover:bg-surface-accent/70 hover:text-text-secondary"
                      )}
                    >
                      {item.label}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </nav>

      {/* Logout */}
      <div className="border-t border-border p-1.5">
        <button
          onClick={onLogout}
          className={cn(
            "flex items-center gap-3 w-full px-2.5 py-2 rounded-lg text-sm text-text-secondary hover:bg-surface-accent transition-colors cursor-pointer"
          )}
        >
          <LogOut size={18} className="shrink-0" />
          <span
            className={cn(
              "truncate transition-all duration-200",
              expanded ? "opacity-100 w-auto" : "opacity-0 w-0 overflow-hidden"
            )}
          >
            로그아웃
          </span>
        </button>
      </div>
    </aside>
  );
}
