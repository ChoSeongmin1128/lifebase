"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  House,
  Cloud,
  Calendar,
  CheckCircle2,
  Image as ImageIcon,
  Settings,
} from "lucide-react";
import { cn } from "@/lib/utils";

const TAB_ITEMS = [
  { href: "/home", label: "Home", icon: House },
  { href: "/cloud", label: "Cloud", icon: Cloud },
  { href: "/calendar", label: "Calendar", icon: Calendar },
  { href: "/todo", label: "Todo", icon: CheckCircle2 },
  { href: "/gallery", label: "Gallery", icon: ImageIcon },
  { href: "/settings", label: "Settings", icon: Settings },
] as const;

export function BottomTabBar() {
  const pathname = usePathname();

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 flex md:hidden border-t border-border bg-background h-14">
      {TAB_ITEMS.map(({ href, label, icon: Icon }) => {
        const isActive = pathname.startsWith(href);
        return (
          <Link
            key={href}
            href={href}
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
