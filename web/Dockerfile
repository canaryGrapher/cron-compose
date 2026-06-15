# Multi-stage build for the CronCompose web UI.
# Uses Next.js standalone output so the runtime image is tiny.

FROM node:20-alpine AS deps
WORKDIR /app
COPY web/package.json ./
RUN npm install --no-audit --no-fund

FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY web/ ./
# next.config.ts must include `output: "standalone"`. We set NEXT_TELEMETRY_DISABLED
# for predictable CI builds.
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:20-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
