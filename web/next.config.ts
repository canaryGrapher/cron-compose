import type { NextConfig } from "next";

const config: NextConfig = {
  // Standalone output for tiny production Docker images.
  output: "standalone",
  // The server-side API base. Override via API_BASE in the environment.
  env: {
    API_BASE: process.env.API_BASE ?? "http://localhost:8080/api/v1",
  },
};

export default config;
