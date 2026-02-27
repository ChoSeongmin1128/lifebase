"use client";

import Link from "next/link";
import { cn } from "@/lib/utils";

interface SidebarItemProps {
  href: string;
  label: string;
  icon: React.ReactNode;
  isActive: boolean;
  expanded: boolean;
  onClick?: React.MouseEventHandler<HTMLAnchorElement>;
  endIcon?: React.ReactNode;
}

export function SidebarItem({ href, label, icon, isActive, expanded, onClick, endIcon }: SidebarItemProps) {
  return (
    <Link
      href={href}
      onClick={onClick}
      className={cn(
        "flex items-center gap-3 mx-1.5 px-2.5 py-2 rounded-lg text-sm transition-colors",
        isActive
          ? "bg-primary/10 text-primary font-medium"
          : "text-text-secondary hover:bg-surface-accent hover:text-text-primary"
      )}
    >
      <span className="shrink-0">{icon}</span>
      <span
        className={cn(
          "truncate transition-all duration-200",
          expanded ? "opacity-100 w-auto" : "opacity-0 w-0 overflow-hidden"
        )}
      >
        {label}
      </span>
      {expanded && endIcon ? <span className="ml-auto shrink-0">{endIcon}</span> : null}
    </Link>
  );
}
