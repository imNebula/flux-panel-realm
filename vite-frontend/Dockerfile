# syntax=docker/dockerfile:1.7
# 多阶段构建 - 构建阶段
FROM --platform=$BUILDPLATFORM node:20.19.0 AS builder

WORKDIR /app

ARG NPM_REGISTRY=https://registry.npmmirror.com
ENV npm_config_registry=${NPM_REGISTRY} \
    npm_config_audit=false \
    npm_config_fund=false

COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --legacy-peer-deps --prefer-offline --no-audit --fund=false

COPY . .
RUN npm run build

# 生产阶段
FROM --platform=$TARGETPLATFORM nginx:stable-alpine AS production-stage

COPY nginx.conf /etc/nginx/nginx.conf

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
