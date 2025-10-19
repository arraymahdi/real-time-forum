# ---- Build Stage ----
FROM node:20 AS builder

WORKDIR /app

# Install dependencies
COPY frontend/package*.json ./
RUN npm ci

# Copy source code and build
COPY frontend/ .
RUN npm run build

# ---- Runtime Stage ----
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000

# Copy only what’s needed for runtime
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package*.json ./

# Install only production dependencies
RUN npm ci --omit=dev

EXPOSE 3000

# Start Next.js server
CMD ["npm", "start"]
