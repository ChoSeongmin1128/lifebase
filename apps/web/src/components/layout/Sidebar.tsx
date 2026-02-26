"use client";

import { Suspense } from "react";
import { usePathname } from "next/navigation";
import Image from "next/image";
import {
  Cloud,
  Calendar,
  CheckCircle2,
  Image as ImageIcon,
  Settings,
  LogOut,
  PanelLeft,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { SidebarItem } from "./SidebarItem";
import { CloudSubnav } from "./CloudSubnav";

const NAV_ITEMS = [
  { href: "/cloud", label: "Cloud", icon: Cloud },
  { href: "/calendar", label: "Calendar", icon: Calendar },
  { href: "/todo", label: "Todo", icon: CheckCircle2 },
  { href: "/gallery", label: "Gallery", icon: ImageIcon },
  { href: "/settings", label: "Settings", icon: Settings },
] as const;

interface SidebarProps {
  expanded: boolean;
  onToggle: () => void;
  onLogout: () => void;
}

export function Sidebar({ expanded, onToggle, onLogout }: SidebarProps) {
  const pathname = usePathname();

  return (
    <aside
      className={cn(
        "hidden md:flex flex-col border-r border-border bg-background transition-all duration-200",
        expanded ? "w-[208px]" : "w-[52px]"
      )}
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-3 border-b border-border">
        <button
          onClick={onToggle}
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-text-secondary hover:bg-surface-accent transition-colors"
        >
          <PanelLeft size={18} />
        </button>
        <div
          className={cn(
            "flex items-center gap-2 transition-all duration-200 overflow-hidden",
            expanded ? "opacity-100 w-auto" : "opacity-0 w-0"
          )}
        >
          <Image src="/logo.svg" alt="LifeBase" width={24} height={24} />
          <span className="text-sm font-semibold text-text-strong whitespace-nowrap">LifeBase</span>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 py-2 space-y-0.5">
        {NAV_ITEMS.map(({ href, label, icon: Icon }) => {
          const isActive = pathname.startsWith(href);
          return (
            <div key={href}>
              <SidebarItem
                href={href}
                label={label}
                icon={<Icon size={18} />}
                isActive={isActive}
                expanded={expanded}
              />
              {href === "/cloud" && isActive && expanded && (
                <Suspense fallback={null}>
                  <CloudSubnav />
                </Suspense>
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
