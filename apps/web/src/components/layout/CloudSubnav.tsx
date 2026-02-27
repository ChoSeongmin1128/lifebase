"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { File, Clock, Users, Star, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { CLOUD_SECTION_ITEMS, parseCloudSection } from "@/lib/cloud-sections";

const ICONS = {
  "": File,
  recent: Clock,
  shared: Users,
  starred: Star,
  trash: Trash2,
} as const;

export function CloudSubnav() {
  const searchParams = useSearchParams();
  const currentSection = parseCloudSection(searchParams.get("section"));

  return (
    <div className="ml-8 mr-1.5 space-y-0.5 pb-1">
      {CLOUD_SECTION_ITEMS.map(({ section, label }) => {
        const Icon = ICONS[section];
        const isActive = currentSection === section;
        const href = section ? `/cloud?section=${section}` : "/cloud";
        return (
          <Link
            key={section}
            href={href}
            className={cn(
              "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors",
              isActive
                ? "text-primary font-medium"
                : "text-text-muted hover:text-text-secondary hover:bg-surface-accent"
            )}
          >
            <Icon size={14} />
            {label}
          </Link>
        );
      })}
    </div>
  );
}
