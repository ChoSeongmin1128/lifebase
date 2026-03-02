import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

const allowedHosts = new Set(["admin.lifebase.cc", "localhost:39001", "127.0.0.1:39001"]);

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

