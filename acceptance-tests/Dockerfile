FROM mcr.microsoft.com/playwright:v1.58.2-jammy

WORKDIR /work/acceptance-tests

COPY acceptance-tests/package*.json ./
RUN npm ci

COPY acceptance-tests/ ./

CMD ["npm", "run", "test"]
