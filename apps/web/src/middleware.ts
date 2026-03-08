import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

function addHost(target: Set<string>, value?: string) {
  if (!value) {
    return;
  }

  try {
    const host = new URL(value).host.toLowerCase();
    if (!host) {
      return;
    }
    target.add(host);

    const [hostname, port] = host.split(":");
    if (hostname === "localhost" && port) {
      target.add(`127.0.0.1:${port}`);
    }
  } catch {
    // Ignore invalid host configuration and keep defaults.
  }
}

function resolveAllowedHosts() {
  const hosts = new Set(["admin.lifebase.cc", "localhost:39001", "127.0.0.1:39001"]);

  addHost(hosts, process.env.WEB_URL);
  addHost(hosts, process.env.ADMIN_URL);

  return hosts;
}

const allowedHosts = resolveAllowedHosts();

function resolveHost(req: NextRequest): string {
  const forwarded = req.headers.get("x-forwarded-host");
  if (forwarded) {
    return forwarded.split(",")[0]?.trim().toLowerCase() || "";
  }
  return req.headers.get("host")?.toLowerCase() || "";
}

export function middleware(req: NextRequest) {
  const host = resolveHost(req);
  if (allowedHosts.has(host)) {
    return NextResponse.next();
  }

  return new NextResponse("Forbidden", { status: 403 });
}

export const config = {
  matcher: ["/admin/:path*"],
};
