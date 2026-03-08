import path from "node:path";
import type { NextConfig } from "next";

function normalizeApiOrigin(value?: string) {
  if (!value) return "";
  return value.trim().replace(/\/+$/, "").replace(/\/api\/v1$/i, "");
}

const configuredApiOrigin = normalizeApiOrigin(process.env.NEXT_PUBLIC_API_URL || process.env.API_URL);
const devApiOrigin =
  process.env.NODE_ENV === "development" ? configuredApiOrigin || "http://localhost:38117" : "";

const nextConfig: NextConfig = {
  output: process.env.TAURI_ENV ? "export" : undefined,
  transpilePackages: ["@lifebase/features-todo", "@lifebase/design-tokens"],
  turbopack: {
    root: path.join(__dirname, "../.."),
  },
  images: {
    remotePatterns: [
      {
        protocol: "http",
        hostname: "localhost",
        port: "38117",
        pathname: "/api/v1/gallery/thumbnails/**",
      },
      {
        protocol: "https",
        hostname: "api.lifebase.cc",
        pathname: "/api/v1/gallery/thumbnails/**",
      },
    ],
  },
  async rewrites() {
    const apiOrigin = process.env.TAURI_ENV ? "" : devApiOrigin || configuredApiOrigin;
    if (!apiOrigin) {
      return [];
    }

    return [
      {
        source: "/api/v1/:path*",
        destination: `${apiOrigin}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
