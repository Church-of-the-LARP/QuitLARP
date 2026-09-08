FROM node:24-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
EXPOSE 5173
CMD ["sh", "-c", "until wget -q -O /dev/null http://backend:8888/openapi.json; do sleep 1; done && npx openapi-typescript http://backend:8888/openapi.json -o src/api/schema.ts && npm run dev -- --host 0.0.0.0"]
