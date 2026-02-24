# Build stage
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

# draw.io assets stage
FROM node:22-alpine AS drawio
ARG DRAWIO_VERSION=24.7.17
RUN apk add --no-cache curl unzip && \
    curl -L "https://github.com/jgraph/drawio/releases/download/v${DRAWIO_VERSION}/draw.io-${DRAWIO_VERSION}.war" \
    -o drawio.war && \
    unzip -q drawio.war -d /drawio-assets

# Production stage
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY --from=drawio /drawio-assets /usr/share/nginx/html/drawio
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
