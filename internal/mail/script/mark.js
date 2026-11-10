// Ticket 7 assumption, verify when built: changing the read status keeps the
// message in its mailbox, so the numeric id stays valid and describe can read it back.
function main(request) {
  const found = findMessage(request);
  found.message.readStatus = request.read === true;
  return describe(found.message, found.mailbox);
}
