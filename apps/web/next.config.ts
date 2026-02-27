import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: process.env.TAURI_ENV ? "export" : undefined,
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
};

export default nextConfig;
