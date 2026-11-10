FROM node:22-alpine AS build

WORKDIR /app
COPY package.json angular.json tsconfig.json tsconfig.app.json ./
COPY src ./src
RUN npm install --no-audit --no-fund && npm run build

FROM nginx:1.27-alpine AS runtime
COPY --from=build /app/dist/bank-sierra/browser /usr/share/nginx/html
