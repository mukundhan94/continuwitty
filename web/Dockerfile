FROM node:22-alpine

WORKDIR /srv/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./

EXPOSE 5173

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5173"]
