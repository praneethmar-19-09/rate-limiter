
# ==========================================
# Stage 1: Install production dependencies
# ==========================================
FROM node:24-alpine3.23 AS dependencies

WORKDIR /app

# Copy only package.json
COPY package.json ./

# Install dependencies
RUN npm install --omit=dev

# ==========================================
# Stage 2: Runtime
# ==========================================
FROM node:24-alpine3.23

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000

# Create non-root user
RUN addgroup -S appgroup \
    && adduser -S appuser -G appgroup

# Copy production dependencies
COPY --from=dependencies --chown=appuser:appgroup /app/node_modules ./node_modules

# Copy application source
COPY --chown=appuser:appgroup app.js ./app.js

USER appuser

EXPOSE 3000

CMD ["node", "app.js"]
