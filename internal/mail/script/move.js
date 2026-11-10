// Ticket 7 assumptions, verify when built: a move gives the message a new numeric
// id in the destination and the old byId specifier goes stale, so everything shown
// afterwards is captured before the move and the message is found again by its
// Message-ID. Structural operations can take tens of seconds on Google accounts,
// so the new id may not be visible yet; the result then carries id 0.
function main(request) {
  const found = findMessage(request);
  const message = found.message;
  const destination = findMailbox({ account: request.account, mailbox: request.destination });
  // The requested path, not the leaf name, so output matches findMessage (e.g. Import/fixture).
  const destinationName = request.destination || 'Inbox';
  const received = optional(() => message.dateReceived());
  const captured = {
    id: 0,
    messageId: message.messageId() || '',
    mailbox: destinationName,
    subject: message.subject() || '',
    sender: message.sender() || '',
    received: received ? received.toISOString() : null,
    read: message.readStatus(),
    flagged: message.flaggedStatus(),
  };
  Mail.move(message, { to: destination });
  if (captured.messageId === '') return captured;
  for (let attempt = 0; attempt < 10; attempt++) {
    const ids = destination.messages.whose({ messageId: captured.messageId }).id();
    if (ids.length > 0) return describe(destination.messages.byId(ids[0]), destinationName);
    delay(0.5);
  }
  return captured;
}
