FROM node:20-alpine

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --omit=dev

COPY src ./src
COPY public ./public

RUN mkdir -p /tmp/webtorrent && chown -R node:node /tmp/webtorrent && chown -R node:node /app

USER node

ENV NODE_ENV=production
ENV PORT=7000
EXPOSE 7000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:7000/health || exit 1

CMD ["node", "src/addon.js"]
