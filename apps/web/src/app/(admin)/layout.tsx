"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  clearAdminTokens,
  isAdminAuthenticated,
  isAdminTokenExpiringSoon,
  refreshAdminAccessToken,
} from "@/features/admin/infrastructure/admin-auth";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Check, Monitor, Moon, Sun } from "lucide-react";

const PUBLIC_PATHS = new Set(["/admin/login", "/admin/auth/callback"]);
const APP_THEME_STORAGE_KEY = "lifebase-theme";
const ADMIN_THEME_STORAGE_KEY = "lifebase-admin-theme";

type ThemeMode = "light" | "dark" | "system";

function parseStoredTheme(raw: string | null, fallback: ThemeMode): ThemeMode {
  if (raw === "light" || raw === "dark" || raw === "system") return raw;
  return fallback;
}

function applyThemeToDocument(theme: ThemeMode) {
  const root = document.documentElement;
  if (theme === "system") {
    root.removeAttribute("data-theme");
    return;
  }
  root.setAttribute("data-theme", theme);
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const isPublicPath = pathname ? PUBLIC_PATHS.has(pathname) : false;
  const [adminTheme, setAdminTheme] = useState<ThemeMode>(() => {
    if (typeof window === "undefined") return "light";
    return parseStoredTheme(window.localStorage.getItem(ADMIN_THEME_STORAGE_KEY), "light");
  });

  useEffect(() => {
    if (isPublicPath) return;
    if (!isAdminAuthenticated()) {
      router.replace("/admin/login");
      return;
    }
    if (isAdminTokenExpiringSoon()) {
      refreshAdminAccessToken().then((token) => {
        if (!token) {
          clearAdminTokens();
          router.replace("/admin/login");
        }
      });
    }
  }, [isPublicPath, router]);

  useEffect(() => {
    // Admin 영역은 사용자 앱 테마와 독립적으로 동작한다.
    applyThemeToDocument(adminTheme);
  }, [adminTheme]);

  useEffect(() => {
    return () => {
      const appTheme = parseStoredTheme(localStorage.getItem(APP_THEME_STORAGE_KEY), "system");
      applyThemeToDocument(appTheme);
    };
  }, []);

  const updateAdminTheme = (nextTheme: ThemeMode) => {
    setAdminTheme(nextTheme);
    localStorage.setItem(ADMIN_THEME_STORAGE_KEY, nextTheme);
    applyThemeToDocument(nextTheme);
  };

  const getThemeIcon = (theme: ThemeMode) => {
    if (theme === "light") return <Sun size={14} />;
    if (theme === "dark") return <Moon size={14} />;
    return <Monitor size={14} />;
  };

  const getThemeLabel = (theme: ThemeMode) => {
    if (theme === "light") return "라이트";
    if (theme === "dark") return "다크";
    return "시스템";
  };

  const handleLogout = () => {
    clearAdminTokens();
    router.replace("/admin/login");
  };

  if (isPublicPath) {
    return <>{children}</>;
  }

  return (
    <div className="min-h-screen bg-background text-text-primary">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-5">
            <Link href="/admin" className="text-sm font-semibold text-text-strong">
              LifeBase 관리자
            </Link>
            <nav className="flex items-center gap-3 text-sm text-text-secondary">
              <Link href="/admin/users" className="hover:text-text-strong">
                사용자
              </Link>
              <Link href="/admin/admins" className="hover:text-text-strong">
                관리자
              </Link>
            </nav>
          </div>
          <div className="flex items-center gap-2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="secondary" size="sm" className="gap-1.5">
                  {getThemeIcon(adminTheme)}
                  {getThemeLabel(adminTheme)}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {(["light", "dark", "system"] as ThemeMode[]).map((theme) => (
                  <DropdownMenuItem key={theme} onClick={() => updateAdminTheme(theme)}>
                    <span className="inline-flex w-4 items-center justify-center">
                      {adminTheme === theme ? <Check size={14} /> : null}
                    </span>
                    {getThemeIcon(theme)}
                    {getThemeLabel(theme)}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
            <Button variant="secondary" size="sm" onClick={handleLogout}>
              로그아웃
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto w-full max-w-6xl px-4 py-6">{children}</main>
    </div>
  );
}
