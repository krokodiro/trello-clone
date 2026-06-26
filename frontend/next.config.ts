import type { NextConfig } from "next";

function resolveApiUrl() {
  const raw = process.env.API_URL || "http://localhost:8080";
  return raw.startsWith("http") ? raw : `http://${raw}`;
}

const apiUrl = resolveApiUrl();

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/v1/:path*",
        destination: `${apiUrl}/api/v1/:path*`,
      },
      {
        source: "/ws/:path*",
        destination: `${apiUrl}/ws/:path*`,
      },
      {
        source: "/ws",
        destination: `${apiUrl}/ws`,
      },
    ];
  },
};

export default nextConfig;
