# Dockerfile
FROM node:20-slim AS builder

WORKDIR /usr/src/app

COPY package*.json ./
COPY tsconfig.json ./

RUN npm ci

COPY src/ ./src/

RUN npm run build

# ---- Final image ----
FROM node:20-slim

WORKDIR /usr/src/app

COPY package*.json ./

# Install production dependencies only
RUN npm ci --omit=dev

COPY --from=builder /usr/src/app/dist ./dist

CMD ["node", "dist/index.js"]
