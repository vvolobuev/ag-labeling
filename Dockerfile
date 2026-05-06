FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

FROM golang:1.25-alpine AS backend-build
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM nginx:alpine

COPY --from=frontend-build /app/frontend/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=backend-build /app/backend/main /app/backend/main
COPY --from=backend-build /app/backend/database /app/backend/database

COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh

RUN mkdir -p /app/storage

EXPOSE 80 443
WORKDIR /app
CMD ["/app/start.sh"]
