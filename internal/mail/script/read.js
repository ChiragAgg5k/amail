function addresses(recipients) {
  return recipients().map((recipient) => ({ name: recipient.name() || '', address: recipient.address() || '' }));
}

function optional(read) {
  try {
    return read();
  } catch (error) {
    return null;
  }
}

function run(argv) {
  const request = JSON.parse(argv[0]);
  const message = findMessage(request);
  const sent = optional(() => message.dateSent());
  const received = optional(() => message.dateReceived());
  return JSON.stringify({
    id: message.id(),
    messageId: message.messageId() || '',
    subject: message.subject() || '',
    sender: message.sender() || '',
    replyTo: optional(() => message.replyTo()) || '',
    to: addresses(message.toRecipients),
    cc: addresses(message.ccRecipients),
    sent: sent ? sent.toISOString() : null,
    received: received ? received.toISOString() : null,
    read: message.readStatus(),
    flagged: message.flaggedStatus(),
    content: message.content() || '',
    attachments: message.mailAttachments().map((attachment) => ({
      name: attachment.name(),
      size: optional(() => attachment.fileSize()) || 0,
      type: optional(() => attachment.mimeType()) || '',
    })),
  });
}
