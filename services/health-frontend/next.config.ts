import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  allowedDevOrigins: ["http://192.168.0.17:3003"],

  // Same reasoning as the finances dashboard: proxied here the requests are
  // same-origin, so there is no CORS preflight to be rejected by the gateway,
  // and the API's public hostname can go away.
  async rewrites() {
    const target = process.env.HEALTH_API_INTERNAL_URL ?? "http://localhost:3336";
    return [{ source: "/api/v1/:path*", destination: `${target}/api/v1/:path*` }];
  },
};

export default nextConfig;
