"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { APP_NAV_ITEMS, isNavItemActive, normalizeSettingsHref } from "./navigation";

export function BottomTabBar() {
  const pathname = usePathname();

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 flex md:hidden border-t border-border bg-background h-14">
      {APP_NAV_ITEMS.map(({ href, label, icon: Icon }) => {
        const isActive = isNavItemActive(pathname, href);
        return (
          <Link
            key={href}
            href={normalizeSettingsHref(href)}
            className={cn(
              "flex flex-1 flex-col items-center justify-center gap-0.5 text-[10px] transition-colors",
              isActive
                ? "text-primary font-medium"
                : "text-text-muted"
            )}
          >
            <Icon size={20} />
            {label}
          </Link>
        );
      })}
    </nav>
  );
}
