FROM node:14.15.0 as application

WORKDIR /app

COPY package.json package.json
COPY package-lock.json package-lock.json
COPY tsconfig.json tsconfig.json
RUN npm ci

COPY src src
RUN npm run build
RUN npm ci --production



FROM node:14.15.0-slim

WORKDIR /app
COPY --from=application /app/node_modules /app/node_modules
COPY --from=application /app/dist /app/dist

ENV NODE_ENV production

EXPOSE 80
EXPOSE 443

CMD ["node", "dist/start"]


