function main(request) {
  return { source: findMessage(request).message.source() || '' };
}
