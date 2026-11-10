# syntax=docker/dockerfile:1
FROM node:22-alpine AS dependencies
WORKDIR /app
COPY package.json ./
RUN npm install --ignore-scripts

FROM dependencies AS test
COPY tsconfig.json ./
COPY src ./src
RUN npm test

FROM test AS runtime
ENV NODE_ENV=production
ENV PORT=8090
RUN npm prune --omit=dev
USER node
EXPOSE 8090
CMD ["node", "dist/server.js"]
