FROM node:22-alpine AS build

WORKDIR /app
COPY package.json tsconfig.json next-env.d.ts next.config.mjs ./
COPY app ./app
RUN npm install --no-audit --no-fund && npm run build

FROM node:22-alpine AS runtime
ENV NODE_ENV=production
WORKDIR /app
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static

CMD ["node", "server.js"]
