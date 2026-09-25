# Build context: repository root (needs packages/design-tokens).
FROM node:24-alpine AS build
WORKDIR /repo
COPY packages ./packages
COPY apps/web/package.json apps/web/package-lock.json ./apps/web/
RUN cd apps/web && npm ci
COPY apps/web ./apps/web
RUN cd apps/web && npm run build && npm prune --omit=dev

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build /repo/apps/web/build ./build
COPY --from=build /repo/apps/web/node_modules ./node_modules
COPY --from=build /repo/apps/web/package.json ./
USER node
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=3s CMD wget -qO- http://127.0.0.1:3000/ >/dev/null || exit 1
CMD ["node", "build"]
