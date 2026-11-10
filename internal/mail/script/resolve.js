function main(request) {
  const found = findMessage(request);
  return describe(found.message, found.mailbox);
}
