FROM node:20-alpine

WORKDIR /app
COPY alertReceiver.js .

EXPOSE 9094

HEALTHCHECK --interval=30s --timeout=3s \
  CMD node alertReceiver.js --health || exit 1

CMD ["node", "alertReceiver.js"]
