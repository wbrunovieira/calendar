import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  allowedDevOrigins: ["http://192.168.0.17:3003"],

  // Proxy the API through this origin instead of letting the browser call it
  // directly on a sibling subdomain.
  //
  // The browser call was cross-origin, which meant CORS, which meant a
  // preflight OPTIONS before every request — and a preflight carries no
  // cookies by specification. That is why the API domain could not simply be
  // put behind the auth gateway: every preflight would be rejected and the
  // dashboard would stop working entirely.
  //
  // Proxied here, requests are same-origin: no CORS, no preflight, and this
  // origin is already behind the gateway. The hop to the API happens
  // server-side over the Docker network and never touches the internet, which
  // is what lets the API's own public hostname be removed.
  async rewrites() {
    const target = process.env.FINANCES_API_INTERNAL_URL ?? "http://localhost:3335";
    return [{ source: "/api/v1/:path*", destination: `${target}/api/v1/:path*` }];
  },
};

export default nextConfig;
